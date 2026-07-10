package main

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thisuite/thisecure/note/internal/config"
	"github.com/thisuite/thisecure/note/internal/handler"
	"github.com/thisuite/thisecure/note/internal/repository"
	"github.com/thisuite/thisecure/note/internal/service"
	"github.com/thisuite/thisecure/pkg/crypto"
	"github.com/thisuite/thisecure/pkg/database"
	"github.com/thisuite/thisecure/pkg/discovery"
	"github.com/thisuite/thisecure/pkg/kafka"
	"github.com/thisuite/thisecure/pkg/metrics"
	mid "github.com/thisuite/thisecure/pkg/middleware"
	"github.com/thisuite/thisecure/pkg/telemetry"
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
	syncProducer := kafka.NewProducer(cfg.KafkaBrokers, "sync-events", signer)
	defer syncProducer.Close()

	noteRepo := repository.NewNoteRepo(pool)
	noteSvc := service.NewNoteService(noteRepo, encKey, syncProducer)
	noteH := handler.NewNoteHandler(noteSvc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(telemetry.GinMiddleware(cfg.OTELServiceName))
	r.Use(telemetry.RequestIDAttribute())
	r.Use(mid.SecurityHeaders())
	r.Use(mid.CORS(nil))
	r.Use(mid.RateLimit(mid.NewRateLimiter(10, 20, time.Second)))
	r.Use(metrics.Middleware())
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "service": "note"}) })
	r.GET("/ready", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "not ready", "error": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready", "service": "note"})
	})
	r.GET("/metrics", metrics.PrometheusHandler())
	r.GET("/metrics/json", metrics.JSONHandler())
	r.GET("/discovery", discovery.Handler(r, "note"))

	v1 := r.Group("/v1/notes", mid.JWTAuth(jwtSecret))
	noteH.Register(v1)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		slog.Info("note service listening", "port", cfg.Port)
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
	srv.Shutdown(shutdownCtx)
}
