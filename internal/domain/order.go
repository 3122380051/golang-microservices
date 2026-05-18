package domain

import (
	"context"
	"time"
)

// OrderStatus represents the lifecycle state of an order
type OrderStatus string

const (
	OrderStatusCreated       OrderStatus = "created"       // Created locally
	OrderStatusSubmitted     OrderStatus = "submitted"     // Sent to exchange
	OrderStatusFilled        OrderStatus = "filled"        // Completely filled
	OrderStatusPartialFilled OrderStatus = "partial_filled" // Partially filled
	OrderStatusCanceled      OrderStatus = "canceled"      // User canceled
	OrderStatusRejected      OrderStatus = "rejected"      // Risk/system rejected
	OrderStatusClosed        OrderStatus = "closed"        // Settlement complete
)

// OrderType represents the order type
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

// OrderSide represents buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// Order represents a trading order
type Order struct {
	ID                string    `json:"id" db:"id"`                           // UUID
	ClientOrderID     string    `json:"client_order_id" db:"client_order_id"` // For idempotency
	CorrelationID     string    `json:"correlation_id" db:"correlation_id"`   // For tracing
	UserID            string    `json:"user_id" db:"user_id"`
	StrategyID        string    `json:"strategy_id" db:"strategy_id"`
	Symbol            string    `json:"symbol" db:"symbol"` // e.g., BTCUSDT
	Side              OrderSide `json:"side" db:"side"`     // BUY or SELL
	OrderType         OrderType `json:"order_type" db:"order_type"`
	Quantity          float64   `json:"quantity" db:"quantity"`
	Price             float64   `json:"price" db:"price"`                 // 0 for market orders
	ExecutedQuantity  float64   `json:"executed_quantity" db:"executed_quantity"`
	ExecutedValue     float64   `json:"executed_value" db:"executed_value"` // Notional value
	AverageFillPrice  float64   `json:"average_fill_price" db:"average_fill_price"`
	Fees              float64   `json:"fees" db:"fees"`
	Status            OrderStatus `json:"status" db:"status"`
	RiskDecisionID    string    `json:"risk_decision_id" db:"risk_decision_id"` // Link to risk check
	SignalID          string    `json:"signal_id" db:"signal_id"`               // Link to strategy signal
	CancelReason      string    `json:"cancel_reason" db:"cancel_reason"`       // Why canceled/rejected
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	SubmittedAt       *time.Time `json:"submitted_at" db:"submitted_at"`
	FilledAt          *time.Time `json:"filled_at" db:"filled_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// OrderRepository defines persistence interface for orders
type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	GetByClientOrderID(ctx context.Context, clientOrderID string) (*Order, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string, status *OrderStatus) ([]Order, error)
	ListByStrategy(ctx context.Context, strategyID string, status *OrderStatus) ([]Order, error)
	ListByStatus(ctx context.Context, status OrderStatus) ([]Order, error)
}

// NewOrder creates a new order with default values
func NewOrder(userID, strategyID, symbol string, side OrderSide, orderType OrderType, qty, price float64) *Order {
	return &Order{
		UserID:     userID,
		StrategyID: strategyID,
		Symbol:     symbol,
		Side:       side,
		OrderType:  orderType,
		Quantity:   qty,
		Price:      price,
		Status:     OrderStatusCreated,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CanTransition checks if a status transition is valid
func (o *Order) CanTransition(newStatus OrderStatus) bool {
	switch o.Status {
	case OrderStatusCreated:
		return newStatus == OrderStatusSubmitted || newStatus == OrderStatusCanceled || newStatus == OrderStatusRejected
	case OrderStatusSubmitted:
		return newStatus == OrderStatusFilled || newStatus == OrderStatusPartialFilled || newStatus == OrderStatusCanceled || newStatus == OrderStatusRejected
	case OrderStatusPartialFilled:
		return newStatus == OrderStatusFilled || newStatus == OrderStatusCanceled
	case OrderStatusFilled, OrderStatusCanceled, OrderStatusRejected, OrderStatusClosed:
		return false // Terminal states
	default:
		return false
	}
}

// TransitionTo transitions the order to a new status
func (o *Order) TransitionTo(newStatus OrderStatus) error {
	if !o.CanTransition(newStatus) {
		return &StateTransitionError{Current: o.Status, Requested: newStatus}
	}

	switch newStatus {
	case OrderStatusSubmitted:
		now := time.Now()
		o.SubmittedAt = &now
	case OrderStatusFilled:
		now := time.Now()
		o.FilledAt = &now
		o.ExecutedQuantity = o.Quantity // Mark as fully filled
	case OrderStatusPartialFilled:
		// ExecutedQuantity set separately
	case OrderStatusCanceled, OrderStatusRejected:
		// Cancel reason set separately
	}

	o.Status = newStatus
	o.UpdatedAt = time.Now()
	return nil
}

// IsFinal returns true if order is in a terminal state
func (o *Order) IsFinal() bool {
	switch o.Status {
	case OrderStatusFilled, OrderStatusCanceled, OrderStatusRejected, OrderStatusClosed:
		return true
	default:
		return false
	}
}

// StateTransitionError indicates an invalid state transition
type StateTransitionError struct {
	Current   OrderStatus
	Requested OrderStatus
}

func (e *StateTransitionError) Error() string {
	return "invalid state transition: " + string(e.Current) + " -> " + string(e.Requested)
}
