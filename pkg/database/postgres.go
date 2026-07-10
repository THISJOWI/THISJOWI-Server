package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thisuite/thisecure/pkg/telemetry"
)

type Config struct {
	DSN                 string
	MaxConns            int32
	MinConns            int32
	MaxConnLifetime     time.Duration
	MaxConnIdleTime     time.Duration
	HealthCheckInterval time.Duration
}

func DefaultConfig(dsn string) Config {
	return Config{
		DSN:                dsn,
		MaxConns:           25,
		MinConns:           5,
		MaxConnLifetime:    30 * time.Minute,
		MaxConnIdleTime:    5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func EnableTracing(pool *pgxpool.Pool, serviceName string) {
	telemetry.EnablePoolTracing(pool, serviceName)
}
