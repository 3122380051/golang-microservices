package order

import (
	"fmt"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// Validator validates order data and operations
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateOrderCreation validates an order before creation
func (v *Validator) ValidateOrderCreation(userID, strategyID, symbol string, side domain.OrderSide,
	orderType domain.OrderType, quantity, price float64) error {

	if userID == "" {
		return &ValidationError{Field: "user_id", Message: "user_id is required"}
	}

	if strategyID == "" {
		return &ValidationError{Field: "strategy_id", Message: "strategy_id is required"}
	}

	if symbol == "" {
		return &ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	if side != domain.OrderSideBuy && side != domain.OrderSideSell {
		return &ValidationError{Field: "side", Message: "side must be BUY or SELL"}
	}

	if orderType != domain.OrderTypeMarket && orderType != domain.OrderTypeLimit {
		return &ValidationError{Field: "order_type", Message: "order_type must be market or limit"}
	}

	if quantity <= 0 {
		return &ValidationError{Field: "quantity", Message: "quantity must be positive"}
	}

	if orderType == domain.OrderTypeLimit && price <= 0 {
		return &ValidationError{Field: "price", Message: "price required for limit orders"}
	}

	return nil
}

// ValidateCancellation validates if an order can be canceled
func (v *Validator) ValidateCancellation(order *domain.Order) error {
	switch order.Status {
	case domain.OrderStatusCreated, domain.OrderStatusSubmitted:
		return nil // Can cancel
	case domain.OrderStatusFilled, domain.OrderStatusPartialFilled:
		return &ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("cannot cancel order in %s state", order.Status),
		}
	case domain.OrderStatusCanceled:
		return &ValidationError{
			Field:   "status",
			Message: "order already canceled",
		}
	case domain.OrderStatusRejected:
		return &ValidationError{
			Field:   "status",
			Message: "order already rejected",
		}
	case domain.OrderStatusClosed:
		return &ValidationError{
			Field:   "status",
			Message: "order already closed",
		}
	default:
		return &ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("unknown order status: %s", order.Status),
		}
	}
}

// ValidateFillUpdate validates a fill update (executed_quantity and fees)
func (v *Validator) ValidateFillUpdate(order *domain.Order, executedQty, avgPrice, fees float64) error {
	if executedQty < 0 {
		return &ValidationError{
			Field:   "executed_quantity",
			Message: "executed_quantity cannot be negative",
		}
	}

	if executedQty > order.Quantity {
		return &ValidationError{
			Field:   "executed_quantity",
			Message: fmt.Sprintf("executed_quantity %.8f exceeds order quantity %.8f", executedQty, order.Quantity),
		}
	}

	if avgPrice <= 0 {
		return &ValidationError{
			Field:   "average_fill_price",
			Message: "average_fill_price must be positive",
		}
	}

	if fees < 0 {
		return &ValidationError{
			Field:   "fees",
			Message: "fees cannot be negative",
		}
	}

	return nil
}

// ValidateOrderState validates internal order state
func (v *Validator) ValidateOrderState(order *domain.Order) error {
	// Quantity should never exceed original
	if order.ExecutedQuantity > order.Quantity {
		return &ValidationError{
			Field:   "executed_quantity",
			Message: "executed_quantity exceeds order quantity",
		}
	}

	// Fees should never be negative
	if order.Fees < 0 {
		return &ValidationError{
			Field:   "fees",
			Message: "fees cannot be negative",
		}
	}

	// If partially or fully filled, average price should be set
	if (order.Status == domain.OrderStatusPartialFilled || order.Status == domain.OrderStatusFilled) &&
		order.ExecutedQuantity > 0 && order.AverageFillPrice <= 0 {
		return &ValidationError{
			Field:   "average_fill_price",
			Message: "average_fill_price should be set for filled orders",
		}
	}

	return nil
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
