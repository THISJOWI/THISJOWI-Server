package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thisuite/thisecure/otp/internal/config"
	"github.com/thisuite/thisecure/otp/internal/handler"
	"github.com/thisuite/thisecure/otp/internal/repository"
	"github.com/thisuite/thisecure/otp/internal/service"
	"github.com/thisuite/thisecure/pkg/crypto"
	"github.com/thisuite/thisecure/pkg/database"
	"github.com/thisuite/thisecure/pkg/discovery"
	"github.com/thisuite/thisecure/pkg/kafka"
	"github.com/thisuite/thisecure/pkg/metrics"
	mid "github.com/thisuite/thisecure/pkg/middleware"
	"github.com/thisuite/thisecure/pkg/models"
	"github.com/thisuite/thisecure/pkg/telemetry"
	"github.com/thisuite/thisecure/pkg/vault"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	otelCfg := telemetry.Config{
		ServiceName:     cfg.OTELServiceName,
		OTLPEndpoint:    cfg.OTLPEndpoint,
		TraceSamplerArg: cfg.TraceSamplerArg,
		LogLevel:        cfg.LogLevel,
	}
	telemetry.SetupLogging(otelCfg)

	tp, mp, err := telemetry.InitTelemetry(ctx, otelCfg)
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
	}
	defer func() {
		if err := telemetry.ShutdownTelemetry(context.Background(), tp, mp); err != nil {
			slog.Error("telemetry shutdown error", "error", err)
		}
	}()

	if cfg.JWTSecret == "" || len(cfg.JWTSecret) < 32 {
		slog.Error("JWT_SECRET must be set and at least 32 characters")
		os.Exit(1)
	}
	if cfg.EncryptionKey == "" {
		slog.Error("ENCRYPTION_KEY must be set")
		os.Exit(1)
	}
	encKey, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil {
		slog.Error("invalid ENCRYPTION_KEY hex", "error", err)
		os.Exit(1)
	}
	if err := crypto.ValidateKey(encKey); err != nil {
		slog.Error("invalid ENCRYPTION_KEY", "error", err)
		os.Exit(1)
	}

	if cfg.VaultEnabled {
		vc := vault.Config{
			Addr:        cfg.VaultAddr,
			K8sAuthRole: cfg.VaultK8sAuthRole,
			DBRole:      cfg.VaultDBRole,
		}
		vtoken, err := vault.Login(ctx, vc)
		if err != nil {
			slog.Error("vault login", "error", err)
			os.Exit(1)
		}
		creds, err := vault.FetchDBCreds(ctx, vtoken, vc)
		if err != nil {
			slog.Error("vault fetch db creds", "error", err)
			os.Exit(1)
		}
		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			creds.Username, creds.Password,
			cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
		go vault.StartRenewal(ctx, vtoken, creds, cfg.VaultAddr)
	}

	pool, err := database.NewPool(ctx, database.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	database.EnableTracing(pool, cfg.OTELServiceName)

	jwtSecret := []byte(cfg.JWTSecret)
	if cfg.KafkaSigningKey == "" {
		slog.Error("KAFKA_SIGNING_KEY must be set")
		os.Exit(1)
	}
	signer := kafka.NewSigner([]byte(cfg.KafkaSigningKey))
	syncProd := kafka.NewProducer(cfg.KafkaBrokers, "sync-events", signer)
	eventProd := kafka.NewProducer(cfg.KafkaBrokers, "otp-events", signer)

	otpRepo := repository.NewOtpRepo(pool)
	qrSvc := service.NewQrService()
	otpSvc := service.NewOtpService(otpRepo, encKey, eventProd, syncProd)
	otpH := handler.NewOtpHandler(otpSvc, qrSvc)

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "auth-events", "otp-service-group", signer, func(ctx context.Context, key string, value []byte) error {
		event, err := kafka.Decode[models.UserRegisteredEvent](value)
		if err != nil {
			return err
		}
		slog.Info("received USER_REGISTERED event", "userId", event.UserID, "email", event.Email)
		return nil
	})
	consumer.SetServiceName(cfg.OTELServiceName)
	go func() {
		if err := consumer.Run(ctx); err != nil {
			slog.Error("kafka consumer stopped", "error", err)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(telemetry.GinMiddleware(cfg.OTELServiceName))
	r.Use(telemetry.RequestIDAttribute())
	r.Use(mid.SecurityHeaders())
	r.Use(mid.CORS(nil))
	r.Use(mid.RateLimit(mid.NewRateLimiter(10, 20, time.Second)))
	r.Use(metrics.Middleware())
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "service": "otp"}) })
	r.GET("/ready", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "service": "otp"})
	})
	r.GET("/metrics", metrics.PrometheusHandler())
	r.GET("/metrics/json", metrics.JSONHandler())
	r.GET("/discovery", discovery.Handler(r, "otp"))

	v1 := r.Group("/v1/otp", mid.JWTAuth(jwtSecret))
	otpH.Register(v1)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		slog.Info("otp service listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer.Close()
	syncProd.Close()
	eventProd.Close()
	srv.Shutdown(shutdownCtx)
}
