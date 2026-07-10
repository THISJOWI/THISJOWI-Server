package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/thisuite/thisecure/pkg/telemetry"
)

type Handler func(ctx context.Context, key string, value []byte) error

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
			slog.Warn("signature verification failed", "error", err)
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
				slog.Error("handler failed", "error", err)
				span.End()
				continue
			}
			span.End()
		} else {
			if err := c.handler(traceCtx, string(msg.Key), msg.Value); err != nil {
				slog.Error("handler failed", "error", err)
				continue
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
	}
}

func (c *Consumer) verifySignature(msg kafka.Message) error {
	var signature string
	for _, h := range msg.Headers {
		if h.Key == "X-Signature" {
			signature = string(h.Value)
			break
		}
	}
	if signature == "" {
		return fmt.Errorf("missing X-Signature header")
	}
	return c.signer.Verify(msg.Value, signature)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func Decode[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
