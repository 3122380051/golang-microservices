package domain

import (
	"time"
)

// RiskDecision represents the outcome of risk evaluation
type RiskDecision struct {
	ID              string    `json:"id"`                // UUID
	SignalID        string    `json:"signal_id"`         // From strategy.signal.generated
	UserID          string    `json:"user_id"`
	StrategyID      string    `json:"strategy_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`              // BUY or SELL
	Quantity        float64   `json:"quantity"`
	EstimatedPrice  float64   `json:"estimated_price"`
	IsApproved      bool      `json:"is_approved"`
	RejectionReason string    `json:"rejection_reason"` // If not approved
	Checks          ChecksDetail `json:"checks"`
	TraceID         string    `json:"trace_id"`
	DecidedAt       time.Time `json:"decided_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// ChecksDetail provides detailed breakdown of risk checks
type ChecksDetail struct {
	PositionSizeCheck CheckResult `json:"position_size_check"`
	LeverageCheck     CheckResult `json:"leverage_check"`
	MarginCheck       CheckResult `json:"margin_check"`
	DailyLossCheck    CheckResult `json:"daily_loss_check"`
	ExposureCheck     CheckResult `json:"exposure_check"`
}

// CheckResult represents a single risk check outcome
type CheckResult struct {
	Passed bool   `json:"passed"`
	Reason string `json:"reason"`
	Value  string `json:"value"` // e.g., "2.5 BTC = $165,250 > max $100,000"
}

// RiskDecisionEvent is published to Kafka topic: "risk.order.approved" or "risk.order.rejected"
type RiskDecisionEvent struct {
	EventID        string    `json:"event_id"`        // UUID
	SignalID       string    `json:"signal_id"`
	UserID         string    `json:"user_id"`
	StrategyID     string    `json:"strategy_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	Quantity       float64   `json:"quantity"`
	IsApproved     bool      `json:"is_approved"`
	RejectionReason string   `json:"rejection_reason"`
	TraceID        string    `json:"trace_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// EventType returns the event topic name based on approval status
func (d *RiskDecision) EventType() string {
	if d.IsApproved {
		return "risk.order.approved"
	}
	return "risk.order.rejected"
}

// ToEvent converts RiskDecision to RiskDecisionEvent
func (d *RiskDecision) ToEvent() *RiskDecisionEvent {
	return &RiskDecisionEvent{
		EventID:         d.ID,
		SignalID:        d.SignalID,
		UserID:          d.UserID,
		StrategyID:      d.StrategyID,
		Symbol:          d.Symbol,
		Side:            d.Side,
		Quantity:        d.Quantity,
		IsApproved:      d.IsApproved,
		RejectionReason: d.RejectionReason,
		TraceID:         d.TraceID,
		Timestamp:       time.Now(),
	}
}
