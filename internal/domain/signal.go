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
	StrategyID string         `json:"strategy_id"`
	Symbol     string         `json:"symbol"`
	Action     SignalAction   `json:"action"`
	Confidence float64        `json:"confidence"`
	Reason     string         `json:"reason"`
	Metadata   map[string]any `json:"metadata"`
	EventID    string         `json:"event_id"`
	TraceID    string         `json:"trace_id"`
	CreatedAt  time.Time      `json:"created_at"`
}
