package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// MessageHandler handles a consumed Kafka message.
type MessageHandler func(context.Context, kafka.Message) error

// KafkaConsumer consumes Kafka messages with optional DLQ publishing.
type KafkaConsumer struct {
	reader          *kafka.Reader
	dlqPublisher    Publisher
	dlqTopic        string
}

// NewKafkaConsumer creates a Kafka consumer for a single topic and consumer group.
func NewKafkaConsumer(brokers []string, groupID, topic string, dlqTopic string, dlqPublisher Publisher) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		dlqPublisher: dlqPublisher,
		dlqTopic:     dlqTopic,
	}
}

// Consume starts the consume loop until the context is canceled.
func (c *KafkaConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	if c == nil || c.reader == nil {
		return fmt.Errorf("kafka consumer is not initialized")
	}

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err := handler(ctx, msg); err != nil {
			if c.dlqPublisher != nil && c.dlqTopic != "" {
				_ = c.publishToDLQ(ctx, msg, err)
			}
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				return fmt.Errorf("commit failed after handler error: %w", commitErr)
			}
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
	}
}

// Close releases consumer resources.
func (c *KafkaConsumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *KafkaConsumer) publishToDLQ(ctx context.Context, msg kafka.Message, handlerErr error) error {
	payload := map[string]any{
		"topic":        msg.Topic,
		"key":          string(msg.Key),
		"value":        string(msg.Value),
		"partition":    msg.Partition,
		"offset":       msg.Offset,
		"error":        handlerErr.Error(),
		"headers":      msg.Headers,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.dlqPublisher.Publish(ctx, c.dlqTopic, msg.Key, raw)
}
