package event

import (
	"encoding/json"
	"time"
)

const (
	TopicMarketPriceUpdated      = "market.price.updated"
	TopicMarketCandleCreated     = "market.candle.created"
	TopicStrategySignalGenerated = "strategy.signal.generated"
	TopicRiskOrderApproved       = "risk.order.approved"
	TopicOrderCreated            = "order.created"
	TopicExecutionSubmitted      = "execution.submitted"
	TopicPortfolioUpdated        = "portfolio.updated"
	TopicNotificationRequested   = "notification.send.requested"
)

const SchemaVersionV1 = 1

// Envelope wraps every broker message with tracing and version metadata.
type Envelope struct {
	EventID   string          `json:"event_id"`
	TraceID   string          `json:"trace_id"`
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// MarketPriceUpdated is the v1 schema for market price events.
type MarketPriceUpdated struct {
	EventID string    `json:"event_id"`
	TraceID string    `json:"trace_id"`
	Version int       `json:"version"`
	Source  string    `json:"source"`
	Symbol  string    `json:"symbol"`
	Exchange string   `json:"exchange"`
	Price   float64   `json:"price"`
	Bid     float64   `json:"bid"`
	Ask     float64   `json:"ask"`
	Key     string    `json:"key"`
	Ts      time.Time `json:"ts"`
}

// PartitionKey returns the broker partition key for this event.
func (e MarketPriceUpdated) PartitionKey() string {
	if e.Key != "" {
		return e.Key
	}
	return e.Symbol
}

// Topic returns the canonical Kafka topic for the event.
func (e MarketPriceUpdated) Topic() string {
	return TopicMarketPriceUpdated
}

// MarshalEnvelope wraps a payload into a versioned envelope.
func MarshalEnvelope(eventType, eventID, traceID, source string, payload any) (Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		EventID:   eventID,
		TraceID:   traceID,
		Version:   SchemaVersionV1,
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Payload:   raw,
	}, nil
}
