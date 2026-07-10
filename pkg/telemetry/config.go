package telemetry

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName     string
	OTLPEndpoint    string
	TraceSamplerArg float64
	LogLevel        string
}

func DefaultConfig() Config {
	return Config{
		ServiceName:     "unknown",
		OTLPEndpoint:    "localhost:4317",
		TraceSamplerArg: 1.0,
		LogLevel:        "info",
	}
}

func ParseConfig() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.ServiceName = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		cfg.OTLPEndpoint = v
	}
	if v := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if f < 0 {
				f = 0
			} else if f > 1 {
				f = 1
			}
			cfg.TraceSamplerArg = f
		}
	}
	if v := os.Getenv("OTEL_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg
}
