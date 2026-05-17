package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Publisher defines the minimal contract used by producers and DLQ publishers.
type Publisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
	PublishJSON(ctx context.Context, topic string, key string, value any) error
	Close() error
}

// KafkaProducer publishes messages to Kafka topics.
type KafkaProducer struct {
	writer *kafka.Writer
}

// NewKafkaProducer creates a Kafka producer configured for at-least-once delivery.
func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Hash{},
			RequiredAcks:  kafka.RequireAll,
			Async:        false,
			BatchTimeout:  0,
			AllowAutoTopicCreation: false,
		},
	}
}

// Publish sends a raw Kafka message.
func (p *KafkaProducer) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	return p.writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: key, Value: value})
}

// PublishJSON serializes a value and publishes it to Kafka.
func (p *KafkaProducer) PublishJSON(ctx context.Context, topic string, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.Publish(ctx, topic, []byte(key), raw)
}

// Close releases producer resources.
func (p *KafkaProducer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
