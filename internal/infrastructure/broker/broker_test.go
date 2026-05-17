package broker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain/event"
	"github.com/segmentio/kafka-go"
)

func TestDefaultTopicsContainCoreWorkflow(t *testing.T) {
	wanted := map[string]bool{
		event.TopicMarketPriceUpdated:      false,
		event.TopicStrategySignalGenerated: false,
		event.TopicRiskOrderApproved:       false,
		event.TopicOrderCreated:            false,
		event.TopicExecutionSubmitted:      false,
		event.TopicPortfolioUpdated:        false,
		event.TopicNotificationRequested:   false,
	}

	for _, topic := range DefaultTopics {
		if _, ok := wanted[topic]; ok {
			wanted[topic] = true
		}
	}

	for topic, found := range wanted {
		if !found {
			t.Fatalf("topic %s not found in default topics", topic)
		}
	}
}

func TestMarketPriceUpdatedPartitionKey(t *testing.T) {
	e := event.MarketPriceUpdated{Symbol: "BTCUSDT"}
	if got := e.PartitionKey(); got != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT partition key, got %s", got)
	}
}

func TestMarshalEnvelope(t *testing.T) {
	payload := event.MarketPriceUpdated{Symbol: "BTCUSDT", Price: 65000, Ts: time.Now().UTC()}
	env, err := event.MarshalEnvelope(event.TopicMarketPriceUpdated, "evt-1", "trace-1", "market-data-service", payload)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	if env.Type != event.TopicMarketPriceUpdated {
		t.Fatalf("unexpected envelope type: %s", env.Type)
	}
	if env.Version != event.SchemaVersionV1 {
		t.Fatalf("unexpected version: %d", env.Version)
	}
	if len(env.Payload) == 0 {
		t.Fatal("payload should not be empty")
	}

	var decoded event.MarketPriceUpdated
	if err := json.Unmarshal(env.Payload, &decoded); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if decoded.Symbol != "BTCUSDT" {
		t.Fatalf("unexpected symbol: %s", decoded.Symbol)
	}
}

func TestKafkaConsumerNilGuard(t *testing.T) {
	var consumer *KafkaConsumer
	if err := consumer.Close(); err != nil {
		t.Fatalf("nil close should not error: %v", err)
	}
}

func TestKafkaProducerNilGuard(t *testing.T) {
	var producer *KafkaProducer
	if err := producer.Close(); err != nil {
		t.Fatalf("nil close should not error: %v", err)
	}
	if err := producer.Publish(context.Background(), "topic", nil, nil); err == nil {
		t.Fatal("expected error from nil producer")
	}
}

func TestMessageShapeForKafka(t *testing.T) {
	msg := kafka.Message{Topic: event.TopicMarketPriceUpdated, Key: []byte("BTCUSDT"), Value: []byte(`{"price":65000}`)}
	if msg.Topic != event.TopicMarketPriceUpdated {
		t.Fatalf("unexpected topic %s", msg.Topic)
	}
}
