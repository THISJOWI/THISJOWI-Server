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
