package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/thisuite/thisecure/pkg/telemetry"
)

type Producer struct {
	writer *kafka.Writer
	signer *Signer
}

func NewProducer(brokers []string, topic string, signer *Signer) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
		},
		signer: signer,
	}
}

func (p *Producer) Publish(ctx context.Context, key string, msg interface{}) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	signature := p.signer.Sign(payload)
	kafkaMsg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "X-Signature", Value: []byte(signature)},
		},
		Time: time.Now(),
	}

	telemetry.InjectTraceContext(ctx, &kafkaMsg)

	return p.writer.WriteMessages(ctx, kafkaMsg)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
