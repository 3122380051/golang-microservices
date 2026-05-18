package order

import (
	"fmt"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// StateMachine manages order state transitions with validation
type StateMachine struct{}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// Transition moves order from one state to another
// Returns error if transition is invalid
func (sm *StateMachine) Transition(order *domain.Order, newStatus domain.OrderStatus, reason string) error {
	// Validate transition
	if !order.CanTransition(newStatus) {
		return &TransitionError{
			From:    string(order.Status),
			To:      string(newStatus),
			Message: fmt.Sprintf("cannot transition from %s to %s", order.Status, newStatus),
		}
	}

	// Apply transition
	if err := order.TransitionTo(newStatus); err != nil {
		return err
	}

	// Store reason if rejecting or canceling
	if newStatus == domain.OrderStatusCanceled || newStatus == domain.OrderStatusRejected {
		order.CancelReason = reason
	}

	return nil
}

// ValidateTransition checks if a transition is allowed without modifying order
func (sm *StateMachine) ValidateTransition(from, to domain.OrderStatus) error {
	// Create temp order to check transition
	tempOrder := &domain.Order{Status: from}
	if !tempOrder.CanTransition(to) {
		return &TransitionError{
			From:    string(from),
			To:      string(to),
			Message: fmt.Sprintf("invalid transition: %s -> %s", from, to),
		}
	}
	return nil
}

// AllowedTransitions returns valid next states for an order
func (sm *StateMachine) AllowedTransitions(status domain.OrderStatus) []domain.OrderStatus {
	switch status {
	case domain.OrderStatusCreated:
		return []domain.OrderStatus{
			domain.OrderStatusSubmitted,
			domain.OrderStatusCanceled,
			domain.OrderStatusRejected,
		}
	case domain.OrderStatusSubmitted:
		return []domain.OrderStatus{
			domain.OrderStatusFilled,
			domain.OrderStatusPartialFilled,
			domain.OrderStatusCanceled,
			domain.OrderStatusRejected,
		}
	case domain.OrderStatusPartialFilled:
		return []domain.OrderStatus{
			domain.OrderStatusFilled,
			domain.OrderStatusCanceled,
		}
	case domain.OrderStatusFilled, domain.OrderStatusCanceled, domain.OrderStatusRejected, domain.OrderStatusClosed:
		return []domain.OrderStatus{} // Terminal states
	default:
		return []domain.OrderStatus{}
	}
}

// DescribeTransition provides a human-readable description of a state change
func (sm *StateMachine) DescribeTransition(from, to domain.OrderStatus) string {
	transitions := map[string]string{
		"created|submitted":        "Order submitted to exchange",
		"submitted|filled":         "Order completely filled",
		"submitted|partial_filled": "Order partially filled",
		"partial_filled|filled":    "Partial fill completed",
		"created|canceled":         "Order canceled before submission",
		"submitted|canceled":       "Order canceled after submission",
		"created|rejected":         "Order rejected by risk service",
		"submitted|rejected":       "Order rejected by exchange",
	}

	key := fmt.Sprintf("%s|%s", from, to)
	if desc, exists := transitions[key]; exists {
		return desc
	}
	return fmt.Sprintf("%s → %s", from, to)
}

// TransitionError represents an invalid state transition
type TransitionError struct {
	From    string
	To      string
	Message string
}

func (e *TransitionError) Error() string {
	return e.Message
}
