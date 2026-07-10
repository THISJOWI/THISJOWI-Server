package config

import (
	"fmt"
	"os"
	"strconv"
)

const ServiceVersion = "1.0.1"

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	EncryptionKey   string
	KafkaSigningKey string
	KafkaBrokers    []string
	DBSSLMode       string
	OTLPEndpoint    string
	OTELServiceName string
	TraceSamplerArg float64
	LogLevel        string
}

func Load() Config {
	cfg := Config{
		Port:            getEnv("PORT", "8084"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""),
		KafkaSigningKey: getEnv("KAFKA_SIGNING_KEY", ""),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		OTLPEndpoint:    getEnv("OTLP_ENDPOINT", "localhost:4317"),
		OTELServiceName: getEnv("OTEL_SERVICE_NAME", "password"),
		TraceSamplerArg: parseFloatEnv("TRACE_SAMPLER_ARG", 1.0),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USERNAME", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/passwords?sslmode=%s", dbUser, dbPass, dbHost, dbPort, cfg.DBSSLMode)

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
