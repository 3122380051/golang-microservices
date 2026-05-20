package domain

import (
	"context"
	"fmt"
	"time"
)

// ExecutionStatus represents the lifecycle state of an order execution
type ExecutionStatus string

const (
	ExecutionStatusCreated       ExecutionStatus = "created"        // Created locally
	ExecutionStatusSubmitting    ExecutionStatus = "submitting"     // Attempting to submit
	ExecutionStatusSubmitted     ExecutionStatus = "submitted"      // Successfully sent to exchange
	ExecutionStatusPartialFilled ExecutionStatus = "partial_filled" // Partially filled
	ExecutionStatusFilled        ExecutionStatus = "filled"         // Completely filled
	ExecutionStatusCanceling     ExecutionStatus = "canceling"      // Attempting to cancel
	ExecutionStatusCanceled      ExecutionStatus = "canceled"       // Canceled by user/system
	ExecutionStatusFailed        ExecutionStatus = "failed"         // Submission failed
	ExecutionStatusClosed        ExecutionStatus = "closed"         // Settlement complete
)

// Execution represents an order execution at the exchange
type Execution struct {
	ID               string          `json:"id" db:"id"`                               // UUID
	OrderID          string          `json:"order_id" db:"order_id"`                   // From Order Service
	ClientOrderID    string          `json:"client_order_id" db:"client_order_id"`     // Idempotency key
	CorrelationID    string          `json:"correlation_id" db:"correlation_id"`       // For tracing
	ExchangeOrderID  string          `json:"exchange_order_id" db:"exchange_order_id"` // From Binance
	UserID           string          `json:"user_id" db:"user_id"`
	Symbol           string          `json:"symbol" db:"symbol"` // e.g., BTCUSDT
	Side             OrderSide       `json:"side" db:"side"`     // BUY or SELL
	OriginalQuantity float64         `json:"original_quantity" db:"original_quantity"`
	ExecutedQuantity float64         `json:"executed_quantity" db:"executed_quantity"`
	ExecutedValue    float64         `json:"executed_value" db:"executed_value"` // Notional
	AverageFillPrice float64         `json:"average_fill_price" db:"average_fill_price"`
	Fees             float64         `json:"fees" db:"fees"`
	Status           ExecutionStatus `json:"status" db:"status"`
	AttemptCount     int             `json:"attempt_count" db:"attempt_count"` // Submission retries
	LastAttemptError string          `json:"last_attempt_error" db:"last_attempt_error"`
	SubmittedAt      *time.Time      `json:"submitted_at" db:"submitted_at"`
	FirstFilledAt    *time.Time      `json:"first_filled_at" db:"first_filled_at"`
	ClosedAt         *time.Time      `json:"closed_at" db:"closed_at"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// FillRecord represents a single fill event from the exchange
type FillRecord struct {
	TradeID      string    `json:"trade_id"`     // Binance trade ID
	ExecutionID  string    `json:"execution_id"` // Our execution ID
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	Fee          float64   `json:"fee"`
	FeeAsset     string    `json:"fee_asset"`
	IsCommission bool      `json:"is_commission"`
	FilledAt     time.Time `json:"filled_at"`
	ReceivedAt   time.Time `json:"received_at"` // When we received it
}

// SubmissionRequest wraps data needed to submit an order to exchange
type SubmissionRequest struct {
	ExecutionID   string    `json:"execution_id"`
	ClientOrderID string    `json:"client_order_id"`
	Symbol        string    `json:"symbol"`
	Side          OrderSide `json:"side"`
	Quantity      float64   `json:"quantity"`
	OrderType     OrderType `json:"order_type"`
	Price         float64   `json:"price"` // 0 for market orders
	TimeInForce   string    `json:"time_in_force"`
}

// SubmissionResult represents the outcome of a submission attempt
type SubmissionResult struct {
	ExecutionID     string
	ExchangeOrderID string
	ClientOrderID   string
	Status          ExecutionStatus
	Error           string
	SubmittedAt     time.Time
}

// ExecutionEvent is published to Kafka
type ExecutionEvent struct {
	EventID          string          `json:"event_id"`
	ExecutionID      string          `json:"execution_id"`
	OrderID          string          `json:"order_id"`
	ClientOrderID    string          `json:"client_order_id"`
	CorrelationID    string          `json:"correlation_id"`
	ExchangeOrderID  string          `json:"exchange_order_id"`
	UserID           string          `json:"user_id"`
	Symbol           string          `json:"symbol"`
	Side             OrderSide       `json:"side"`
	OriginalQty      float64         `json:"original_qty"`
	ExecutedQty      float64         `json:"executed_qty"`
	AverageFillPrice float64         `json:"average_fill_price"`
	Fees             float64         `json:"fees"`
	Status           ExecutionStatus `json:"status"`
	EventType        string          `json:"event_type"` // "execution.submitted" or "execution.filled"
	Timestamp        time.Time       `json:"timestamp"`
	TraceID          string          `json:"trace_id"`
}

// ExecutionRepository defines persistence interface for executions
type ExecutionRepository interface {
	Create(ctx context.Context, execution *Execution) error
	GetByID(ctx context.Context, id string) (*Execution, error)
	GetByClientOrderID(ctx context.Context, clientOrderID string) (*Execution, error)
	GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*Execution, error)
	Update(ctx context.Context, execution *Execution) error
	ListByUser(ctx context.Context, userID string, status *ExecutionStatus) ([]Execution, error)
	ListByStatus(ctx context.Context, status ExecutionStatus) ([]Execution, error)
}

// NewExecution creates a new execution with default values
func NewExecution(orderID, clientOrderID, userID, symbol string, side OrderSide, quantity float64) *Execution {
	return &Execution{
		OrderID:          orderID,
		ClientOrderID:    clientOrderID,
		UserID:           userID,
		Symbol:           symbol,
		Side:             side,
		OriginalQuantity: quantity,
		Status:           ExecutionStatusCreated,
		AttemptCount:     0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// CanTransition checks if a status transition is valid
func (e *Execution) CanTransition(newStatus ExecutionStatus) bool {
	switch e.Status {
	case ExecutionStatusCreated:
		return newStatus == ExecutionStatusSubmitting || newStatus == ExecutionStatusFailed
	case ExecutionStatusSubmitting:
		return newStatus == ExecutionStatusSubmitted || newStatus == ExecutionStatusFailed
	case ExecutionStatusSubmitted:
		return newStatus == ExecutionStatusPartialFilled || newStatus == ExecutionStatusFilled ||
			newStatus == ExecutionStatusCanceling || newStatus == ExecutionStatusFailed
	case ExecutionStatusPartialFilled:
		return newStatus == ExecutionStatusFilled || newStatus == ExecutionStatusCanceling ||
			newStatus == ExecutionStatusClosed || newStatus == ExecutionStatusFailed
	case ExecutionStatusFilled:
		return newStatus == ExecutionStatusClosed || newStatus == ExecutionStatusFailed
	case ExecutionStatusCanceling:
		return newStatus == ExecutionStatusCanceled || newStatus == ExecutionStatusFailed
	case ExecutionStatusCanceled, ExecutionStatusFailed, ExecutionStatusClosed:
		return false // Terminal states
	default:
		return false
	}
}

// TransitionTo transitions the execution to a new status
func (e *Execution) TransitionTo(newStatus ExecutionStatus) error {
	if !e.CanTransition(newStatus) {
		return fmt.Errorf("invalid execution state transition: %s -> %s", e.Status, newStatus)
	}

	now := time.Now()
	switch newStatus {
	case ExecutionStatusSubmitted:
		e.SubmittedAt = &now
	case ExecutionStatusPartialFilled:
		if e.FirstFilledAt == nil {
			e.FirstFilledAt = &now
		}
	case ExecutionStatusFilled:
		if e.FirstFilledAt == nil {
			e.FirstFilledAt = &now
		}
	case ExecutionStatusClosed:
		e.ClosedAt = &now
	}

	e.Status = newStatus
	e.UpdatedAt = now
	return nil
}

// IsFinal returns true if execution is in a terminal state
func (e *Execution) IsFinal() bool {
	switch e.Status {
	case ExecutionStatusFilled, ExecutionStatusClosed, ExecutionStatusCanceled, ExecutionStatusFailed:
		return true
	default:
		return false
	}
}
