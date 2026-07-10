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
