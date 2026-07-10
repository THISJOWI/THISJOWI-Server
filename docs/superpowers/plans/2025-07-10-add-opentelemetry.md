# OpenTelemetry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add distributed tracing, trace context propagation, and structured logging with trace correlation to all 3 Go microservices.

**Architecture:** Create `pkg/telemetry/` (OTel init, Gin middleware, Kafka propagation, pgx tracing, structured logging) — one shared package consumed by all services via `go.work`. Each service gets OTel config env vars, init in `main()`, Gin middleware, DB query tracing, Kafka trace propagation, and JSON structured logging with `trace_id`/`span_id`.

**Tech Stack:** OpenTelemetry Go SDK v1.41.0, OTLP gRPC exporter, otelgin middleware, pgx v5 QueryTracer interface, slog, segmentio/kafka-go.

## Global Constraints

- All OTel logic lives in `pkg/telemetry/` — services never import OTel packages directly
- Use `go.opentelemetry.io/otel` v1.41.0 and compatible contrib packages (matching what `core` uses)
- Use pgx v5's built-in `QueryTracer` interface (not `otelsql` which requires `database/sql`)
- W3C TraceContext format for propagation
- Insecure gRPC for OTLP exporter (same as core)
- All log output switches from `log.Printf` to `slog` JSON with `trace_id`/`span_id`
- Existing Prometheus metrics (`pkg/metrics`) remain unchanged

---

## File Structure

### Created files:
- `pkg/telemetry/config.go` — Config struct, defaults, ParseConfig
- `pkg/telemetry/provider.go` — InitTelemetry (TracerProvider, MeterProvider, global registration)
- `pkg/telemetry/shutdown.go` — ShutdownTelemetry (graceful shutdown with 5s timeout)
- `pkg/telemetry/gin.go` — GinMiddleware (otelgin wrapper), RequestIDAttribute
- `pkg/telemetry/kafka.go` — InjectTraceContext, ExtractTraceContext, WrapConsumerHandler
- `pkg/telemetry/logging.go` — NewLogger, SetupLogging, HandlerWithTrace, LoggerFromContext
- `pkg/telemetry/trace_handler.go` — TraceHandler (slog.Handler wrapper adding trace_id/span_id)
- `pkg/telemetry/pgx.go` — PgxTracer (custom pgx.QueryTracer + BatchTracer)

### Modified files:
- `pkg/database/postgres.go` — add EnableTracing wrapping pool with PgxTracer
- `pkg/kafka/producer.go` — add trace context injection in Publish
- `pkg/kafka/consumer.go` — add trace context extraction + span creation in Run
- `services/note/internal/config/config.go` — add OTel fields + env vars
- `services/note/cmd/server/main.go` — add OTel init, middleware, slog logging
- `services/otp/internal/config/config.go` — add OTel fields + env vars
- `services/otp/cmd/server/main.go` — add OTel init, middleware, slog logging
- `services/password/internal/config/config.go` — add OTel fields + env vars
- `services/password/cmd/server/main.go` — add OTel init, middleware, slog logging
- `pkg/go.mod` — add OTel dependencies
- `.env.example` — add OTel env vars

### No changes needed:
- `services/note/internal/handler/` — handlers use `c.Request.Context()` which already propagates
- `services/note/internal/service/` — services use context from handlers
- `services/note/internal/repository/` — repos use context from services (pgx tracer picks up spans)
- `pkg/metrics/` — Prometheus metrics coexist with OTel
- `pkg/middleware/` — OTel middleware is separate (otelgin is in telemetry package)

---

### Task 1: Create `pkg/telemetry/` package (all 8 files)

**Files:**
- Create: `pkg/telemetry/config.go`
- Create: `pkg/telemetry/provider.go`
- Create: `pkg/telemetry/shutdown.go`
- Create: `pkg/telemetry/gin.go`
- Create: `pkg/telemetry/kafka.go`
- Create: `pkg/telemetry/logging.go`
- Create: `pkg/telemetry/trace_handler.go`
- Create: `pkg/telemetry/pgx.go`

**Interfaces:**
- Produces: `InitTelemetry(ctx, Config) -> (TracerProvider, MeterProvider, error)`, `ShutdownTelemetry(ctx, tp, mp) -> error`, `GinMiddleware(serviceName) -> gin.HandlerFunc`, `RequestIDAttribute() -> gin.HandlerFunc`, `InjectTraceContext(ctx, *kafka.Message)`, `ExtractTraceContext(ctx, *kafka.Message) -> context.Context`, `SetupLogging(Config) -> *slog.Logger`, `NewQueryTracer(serviceName) -> *QueryTracer`

- [ ] **Step 1: Create `pkg/telemetry/config.go`**

```go
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
```

- [ ] **Step 2: Create `pkg/telemetry/provider.go`**

```go
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTelemetry(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, *metric.MeterProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TraceSamplerArg)),
		),
		sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(traceExporter),
		),
	)

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(metricExporter),
		),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, mp, nil
}
```

- [ ] **Step 3: Create `pkg/telemetry/shutdown.go`**

```go
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func ShutdownTelemetry(ctx context.Context, tp *sdktrace.TracerProvider, mp *metric.MeterProvider) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errs []error

	if tp != nil {
		if err := tp.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown tracer provider: %w", err))
		}
	}

	if mp != nil {
		if err := mp.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown meter provider: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
```

- [ ] **Step 4: Create `pkg/telemetry/gin.go`**

```go
package telemetry

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func GinMiddleware(serviceName string) gin.HandlerFunc {
	tp := otel.GetTracerProvider()
	return otelgin.Middleware(serviceName, otelgin.WithTracerProvider(tp))
}

func RequestIDAttribute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if requestID, exists := c.Get("request_id"); exists {
			if rid, ok := requestID.(string); ok && rid != "" {
				span := trace.SpanFromContext(c.Request.Context())
				span.SetAttributes(attribute.String("http.request_id", rid))
			}
		}
	}
}
```

- [ ] **Step 5: Create `pkg/telemetry/kafka.go`**

```go
package telemetry

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationVersion = "0.1.0"

func InjectTraceContext(ctx context.Context, msg *kafka.Message) {
	if msg == nil {
		return
	}

	carrier := make(propagation.HeaderCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	for key, values := range carrier {
		for _, value := range values {
			msg.Headers = append(msg.Headers, kafka.Header{
				Key:   key,
				Value: []byte(value),
			})
		}
	}
}

func ExtractTraceContext(ctx context.Context, msg *kafka.Message) context.Context {
	if msg == nil {
		return ctx
	}

	carrier := make(propagation.HeaderCarrier)
	for _, h := range msg.Headers {
		carrier.Set(h.Key, string(h.Value))
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

func WrapConsumerHandler(
	handler func(ctx context.Context, msg kafka.Message) error,
	serviceName string,
) func(ctx context.Context, msg kafka.Message) error {
	tracer := otel.Tracer(serviceName,
		trace.WithInstrumentationVersion(instrumentationVersion),
	)

	return func(ctx context.Context, msg kafka.Message) error {
		ctx = ExtractTraceContext(ctx, &msg)

		spanName := "process message"
		if msg.Topic != "" {
			spanName = msg.Topic + " process"
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination", msg.Topic),
				attribute.String("messaging.kafka.message_key", string(msg.Key)),
			),
		)
		defer span.End()

		err := handler(ctx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.Bool("error", true))
		}
		return err
	}
}
```

- [ ] **Step 6: Create `pkg/telemetry/logging.go`**

```go
package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

func NewLogger(cfg Config) *slog.Logger {
	level := parseLogLevel(cfg.LogLevel)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(NewTraceHandler(handler)).With("service", cfg.ServiceName)
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	return HandlerWithTrace(ctx, slog.Default())
}

func HandlerWithTrace(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}

	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return logger
	}

	return logger.With(
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	)
}

func SetupLogging(cfg Config) *slog.Logger {
	logger := NewLogger(cfg)
	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

- [ ] **Step 7: Create `pkg/telemetry/trace_handler.go`**

```go
package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	handler slog.Handler
}

func NewTraceHandler(handler slog.Handler) *TraceHandler {
	return &TraceHandler{handler: handler}
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		sc := span.SpanContext()
		if sc.HasTraceID() {
			r.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
		}
		if sc.HasSpanID() {
			r.AddAttrs(slog.String("span_id", sc.SpanID().String()))
		}
	}
	return h.handler.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.handler.WithAttrs(attrs)
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}
```

- [ ] **Step 8: Create `pkg/telemetry/pgx.go`**

```go
package telemetry

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PgxTracer struct {
	tracer trace.Tracer
}

func NewPgxTracer(serviceName string) *PgxTracer {
	return &PgxTracer{
		tracer: otel.Tracer(serviceName, trace.WithInstrumentationVersion(instrumentationVersion)),
	}
}

func (t *PgxTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, _ = t.tracer.Start(ctx, data.SQL,
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.statement", truncate(data.SQL, 1000)),
		),
	)
	return ctx
}

func (t *PgxTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	span.End()
}

func (t *PgxTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	ctx, _ = t.tracer.Start(ctx, "batch",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.Int("db.batch_size", len(data.Batch.QueuedQueries)),
		),
	)
	return ctx
}

func (t *PgxTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
}

func (t *PgxTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	span.End()
}

func (t *PgxTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	ctx, _ = t.tracer.Start(ctx, "connect",
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
		),
	)
	return ctx
}

func (t *PgxTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	span.End()
}

func EnablePoolTracing(pool *pgxpool.Pool, serviceName string) {
	tracer := NewPgxTracer(serviceName)
	pool.Config().ConnConfig.Tracer = tracer
	pool.Config().ConnConfig.BatchTracer = tracer
	pool.Config().ConnConfig.ConnectTracer = tracer
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
```

- [ ] **Step 9: Verify compilation**

Run: `cd /Users/joel/Workspace/thisuite/thisecure/backend && go vet ./pkg/telemetry/...`
Expected: no errors (package doesn't compile yet — needs go.mod deps from Task 6)

---

### Task 2: Add DB tracing to `pkg/database/postgres.go`

**Files:**
- Modify: `pkg/database/postgres.go`

**Interfaces:**
- Consumes: `telemetry.EnablePoolTracing(pool, serviceName)`
- Produces: `database.NewPool(ctx, cfg) -> *pgxpool.Pool` (unchanged signature), `database.EnableTracing(pool, serviceName)` (new)

- [ ] **Step 1: Add EnableTracing + Update NewPool**

Add a new `EnableTracing` function and update `NewPool` to accept an optional service name for tracing:

```go
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
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/joel/Workspace/thisuite/thisecure/backend && go vet ./pkg/database/...`
Expected: errors about missing OTel deps (resolved in Task 6)

---

### Task 3: Add Kafka trace propagation

**Files:**
- Modify: `pkg/kafka/producer.go`
- Modify: `pkg/kafka/consumer.go`

**Interfaces:**
- Consumes: `telemetry.InjectTraceContext(ctx, *kafka.Message)`, `telemetry.ExtractTraceContext(ctx, *kafka.Message) -> context.Context`
- Produces: `Producer.Publish(ctx, key, msg)` unchanged signature, `Consumer.Run(ctx)` unchanged signature

- [ ] **Step 1: Add trace injection to Producer.Publish**

In `pkg/kafka/producer.go`, add the import and inject trace context after HMAC signing:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/thisuite/thisecure/pkg/telemetry"
)

func (p *Producer) Publish(ctx context.Context, key string, msg interface{}) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	signature := p.signer.Sign(payload)
	headers := []kafka.Header{
		{Key: "X-Signature", Value: []byte(signature)},
	}
	kafkaMsg := kafka.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
		Time:    time.Now(),
	}

	telemetry.InjectTraceContext(ctx, &kafkaMsg)

	return p.writer.WriteMessages(ctx, kafkaMsg)
}
```

- [ ] **Step 2: Add trace extraction + span to Consumer.Run**

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"github.com/thisuite/thisecure/pkg/telemetry"
)

// Add a ServiceName field to Consumer and NewConsumer
type Consumer struct {
	reader      *kafka.Reader
	signer      *Signer
	handler     Handler
	serviceName string
}

func NewConsumer(brokers []string, topic, groupID string, signer *Signer, handler Handler) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 10,
			MaxBytes: 10e6,
		}),
		signer:  signer,
		handler: handler,
	}
}

// Add SetServiceName method
func (c *Consumer) SetServiceName(name string) {
	c.serviceName = name
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
		if err := c.verifySignature(msg); err != nil {
			log.Printf("WARN: signature verification failed: %v", err)
			continue
		}

		traceCtx := telemetry.ExtractTraceContext(ctx, &msg)

		if c.serviceName != "" {
			tracer := otel.Tracer(c.serviceName)
			spanName := "process message"
			if msg.Topic != "" {
				spanName = msg.Topic + " process"
			}
			spanCtx, span := tracer.Start(traceCtx, spanName,
				trace.WithAttributes(
					attribute.String("messaging.system", "kafka"),
					attribute.String("messaging.destination", msg.Topic),
					attribute.String("messaging.kafka.message_key", string(msg.Key)),
				),
			)
			if err := c.handler(spanCtx, string(msg.Key), msg.Value); err != nil {
				span.RecordError(err)
				span.SetAttributes(attribute.Bool("error", true))
				log.Printf("ERROR: handler failed: %v", err)
				span.End()
				continue
			}
			span.End()
		} else {
			if err := c.handler(traceCtx, string(msg.Key), msg.Value); err != nil {
				log.Printf("ERROR: handler failed: %v", err)
				continue
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/joel/Workspace/thisuite/thisecure/backend && go vet ./pkg/kafka/...`
Expected: errors about missing OTel deps (resolved in Task 6)

---

### Task 4: Add OTel config to each service config.go

**Files:**
- Modify: `services/note/internal/config/config.go`
- Modify: `services/otp/internal/config/config.go`
- Modify: `services/password/internal/config/config.go`

**Interfaces:**
- Produces: `config.Config` with new fields `OTLPEndpoint`, `OTELServiceName`, `TraceSamplerArg`, `LogLevel`

- [ ] **Step 1: Update `services/note/internal/config/config.go`**

Add new fields and env var parsing:

```go
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
		Port:            getEnv("PORT", "8083"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""),
		KafkaSigningKey: getEnv("KAFKA_SIGNING_KEY", ""),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		OTLPEndpoint:    getEnv("OTLP_ENDPOINT", "localhost:4317"),
		OTELServiceName: getEnv("OTEL_SERVICE_NAME", "note"),
		TraceSamplerArg: parseFloatEnv("TRACE_SAMPLER_ARG", 1.0),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}
	// ... rest unchanged
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
```

Add imports: `"strconv"`

- [ ] **Step 2: Update `services/otp/internal/config/config.go`**

Same changes with default OTELServiceName = "otp" and default port = "8085".

- [ ] **Step 3: Update `services/password/internal/config/config.go`**

Same changes with default OTELServiceName = "password" and default port = "8084".

---

### Task 5: Update each service's main.go

**Files:**
- Modify: `services/note/cmd/server/main.go`
- Modify: `services/otp/cmd/server/main.go`
- Modify: `services/password/cmd/server/main.go`

**Interfaces:**
- Consumes: See `pkg/telemetry/` interfaces + config changes from Task 4

- [ ] **Step 1: Update `services/note/cmd/server/main.go`**

Replace imports — add `slog`, `telemetry` package, remove `log`:

```go
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
```

Replace the main function body:

```go
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
```

Key changes:
- Replace `gin.Default()` with `gin.New() + gin.Recovery()` to avoid gin.Logger (we use OTel + slog)
- Add `telemetry.GinMiddleware` as first middleware
- Add `telemetry.RequestIDAttribute` after it
- Add `telemetry.SetupLogging` at the start
- Add `telemetry.InitTelemetry` after logging
- Add `defer telemetry.ShutdownTelemetry`
- Add `database.EnableTracing(pool, serviceName)`
- Switch all `log.Printf`/`log.Fatal` to `slog.Info`/`slog.Error` + `os.Exit(1)`

- [ ] **Step 2: Update `services/otp/cmd/server/main.go`**

Same pattern as step 1, plus:
- Set service name on consumer: `consumer.SetServiceName(cfg.OTELServiceName)`
- Switch `log.Printf` to `slog` in the Kafka consumer handler inline

- [ ] **Step 3: Update `services/password/cmd/server/main.go`**

Same pattern as step 1 (no consumer — same as note).

---

### Task 6: Update go.mod files + tidy

**Files:**
- Modify: `pkg/go.mod`

- [ ] **Step 1: Add OTel dependencies to `pkg/go.mod`**

Run:

```bash
cd /Users/joel/Workspace/thisuite/thisecure/backend/pkg && go get go.opentelemetry.io/otel@v1.41.0 && go get go.opentelemetry.io/otel/trace@v1.41.0 && go get go.opentelemetry.io/otel/sdk@v1.41.0 && go get go.opentelemetry.io/otel/sdk/metric@v1.41.0 && go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.41.0 && go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.41.0 && go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin@v0.60.0 && go mod tidy
```

- [ ] **Step 2: Tidy all Go modules**

```bash
cd /Users/joel/Workspace/thisuite/thisecure/backend/pkg && go mod tidy
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/note && go mod tidy
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/otp && go mod tidy
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/password && go mod tidy
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/joel/Workspace/thisuite/thisecure/backend && go build ./pkg/... && go vet ./pkg/...
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/note && go build ./... && go vet ./...
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/otp && go build ./... && go vet ./...
cd /Users/joel/Workspace/thisuite/thisecure/backend/services/password && go build ./... && go vet ./...
```

---

### Task 7: Update .env.example

**Files:**
- Modify: `.env.example`

- [ ] **Step 1: Add OTel env vars**

Add to `.env.example` after the KAFKA_SIGNING_KEY line:

```
# OpenTelemetry
OTLP_ENDPOINT=localhost:4317
OTEL_SERVICE_NAME=note
TRACE_SAMPLER_ARG=1.0
LOG_LEVEL=info
```

---

## Self-Review

**1. Spec coverage:**
- ✅ Task 1: All 8 pkg/telemetry/ files — covers OTel init, shutdown, Gin middleware, Kafka propagation, logging, pgx tracing
- ✅ Task 2: DB tracing via pgx QueryTracer (no otelsql dependency)
- ✅ Task 3: Kafka trace injection/extraction
- ✅ Task 4: Service config OTel env vars
- ✅ Task 5: main.go OTel init + middleware + slog
- ✅ Task 6: Dependencies
- ✅ Task 7: .env.example

**2. Placeholder scan:** No "TBD", "TODO", "implement later" patterns found. All code is concrete.

**3. Type consistency:** 
- `telemetry.Config` defined in Task 1, used in Task 4-5 ✓
- `telemetry.InitTelemetry`, `ShutdownTelemetry`, `GinMiddleware`, `RequestIDAttribute` defined in Task 1, used in Task 5 ✓
- `telemetry.InjectTraceContext`, `ExtractTraceContext` defined in Task 1, used in Task 3 ✓
- `database.EnableTracing` defined in Task 2, used in Task 5 ✓
- `config.Config.OTLPEndpoint`, `OTELServiceName`, `TraceSamplerArg`, `LogLevel` defined in Task 4, used in Task 5 ✓
- `Consumer.SetServiceName` defined in Task 3, used in Task 5 ✓
