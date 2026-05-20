package domain

import "time"

// SignalAction describes the outcome of a strategy evaluation.
type SignalAction string

const (
	SignalActionBuy  SignalAction = "buy"
	SignalActionSell SignalAction = "sell"
	SignalActionHold SignalAction = "hold"
)

// Signal is emitted by Strategy Service when a strategy evaluates a market event.
type Signal struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`          // Owner of the strategy
	StrategyID string         `json:"strategy_id"`
	Symbol     string         `json:"symbol"`
	Side       string         `json:"side"`             // BUY or SELL
	Quantity   float64        `json:"quantity"`         // Order quantity
	Action     SignalAction   `json:"action"`
	Confidence float64        `json:"confidence"`       // Signal strength (0-1) or estimated price
	Reason     string         `json:"reason"`
	Metadata   map[string]any `json:"metadata"`
	EventID    string         `json:"event_id"`
	TraceID    string         `json:"trace_id"`
	CreatedAt  time.Time      `json:"created_at"`
}
