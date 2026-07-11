package config

import (
	"fmt"
	"os"
	"strconv"
)

const ServiceVersion = "1.0.3"

type Config struct {
	Port            string
	DatabaseURL     string
	DBHost          string
	DBPort          string
	DBName          string
	DBSSLMode       string
	JWTSecret       string
	EncryptionKey   string
	KafkaSigningKey string
	KafkaBrokers    []string
	VaultEnabled    bool
	VaultAddr       string
	VaultK8sAuthRole string
	VaultDBRole     string
	OTLPEndpoint    string
	OTELServiceName string
	TraceSamplerArg float64
	LogLevel        string
}

func Load() Config {
	cfg := Config{
		Port:            getEnv("PORT", "8085"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""),
		KafkaSigningKey: getEnv("KAFKA_SIGNING_KEY", ""),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		OTLPEndpoint:    getEnv("OTLP_ENDPOINT", "localhost:4317"),
		OTELServiceName: getEnv("OTEL_SERVICE_NAME", "otp"),
		TraceSamplerArg: parseFloatEnv("TRACE_SAMPLER_ARG", 1.0),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	cfg.DBHost = getEnv("DB_HOST", "localhost")
	cfg.DBPort = getEnv("DB_PORT", "5432")
	cfg.DBName = getEnv("DB_NAME", "otp")
	dbUser := getEnv("DB_USERNAME", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	cfg.VaultEnabled = getEnv("VAULT_ENABLED", "false") == "true"
	cfg.VaultAddr = getEnv("VAULT_ADDR", "https://vault.thisjowi.com")
	cfg.VaultK8sAuthRole = getEnv("VAULT_K8S_AUTH_ROLE", "connection")
	cfg.VaultDBRole = getEnv("VAULT_DB_ROLE", "connections")
	cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)

	kafkaHost := getEnv("KAFKA_HOST", "localhost")
	kafkaPort := getEnv("KAFKA_PORT", "9092")
	cfg.KafkaBrokers = []string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if f < 0 {
				return 0
			} else if f > 1 {
				return 1
			}
			return f
		}
	}
	return fallback
}
