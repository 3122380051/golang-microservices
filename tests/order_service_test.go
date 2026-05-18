package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/3122380051/golang-microservices/internal/application/order"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
)

// Test StateMachine transitions

func TestStateMachine_ValidTransition_CreatedToSubmitted(t *testing.T) {
	sm := order.NewStateMachine()

	ord := &domain.Order{
		ID:     "order-1",
		Status: domain.OrderStatusCreated,
	}

	err := sm.Transition(ord, domain.OrderStatusSubmitted, "")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusSubmitted, ord.Status)
	assert.NotNil(t, ord.SubmittedAt)
}

func TestStateMachine_ValidTransition_SubmittedToFilled(t *testing.T) {
	sm := order.NewStateMachine()

	ord := &domain.Order{
		ID:     "order-1",
		Status: domain.OrderStatusSubmitted,
	}

	err := sm.Transition(ord, domain.OrderStatusFilled, "")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusFilled, ord.Status)
	assert.NotNil(t, ord.FilledAt)
	assert.Equal(t, ord.Quantity, ord.ExecutedQuantity)
}

func TestStateMachine_InvalidTransition_FilledToSubmitted(t *testing.T) {
	sm := order.NewStateMachine()

	ord := &domain.Order{
		ID:     "order-1",
		Status: domain.OrderStatusFilled,
	}

	err := sm.Transition(ord, domain.OrderStatusSubmitted, "")
	require.Error(t, err)
	assert.Equal(t, domain.OrderStatusFilled, ord.Status) // Status unchanged
}

func TestStateMachine_AllowedTransitions(t *testing.T) {
	sm := order.NewStateMachine()

	tests := []struct {
		status   domain.OrderStatus
		expected int // count of allowed transitions
	}{
		{domain.OrderStatusCreated, 3},       // submitted, canceled, rejected
		{domain.OrderStatusSubmitted, 4},     // filled, partial_filled, canceled, rejected
		{domain.OrderStatusPartialFilled, 2}, // filled, canceled
		{domain.OrderStatusFilled, 0},        // terminal
	}

	for _, tt := range tests {
		allowed := sm.AllowedTransitions(tt.status)
		assert.Equal(t, tt.expected, len(allowed), "status: "+string(tt.status))
	}
}

// Test Validator

func TestValidator_ValidateOrderCreation_Valid(t *testing.T) {
	v := order.NewValidator()

	err := v.ValidateOrderCreation(
		"user1",
		"strategy1",
		"BTCUSDT",
		domain.OrderSideBuy,
		domain.OrderTypeMarket,
		0.5,
		0,
	)
	require.NoError(t, err)
}

func TestValidator_ValidateOrderCreation_MissingUserID(t *testing.T) {
	v := order.NewValidator()

	err := v.ValidateOrderCreation(
		"",
		"strategy1",
		"BTCUSDT",
		domain.OrderSideBuy,
		domain.OrderTypeMarket,
		0.5,
		0,
	)
	require.Error(t, err)
	assert.True(t, order.IsValidationError(err))
}

func TestValidator_ValidateOrderCreation_InvalidQuantity(t *testing.T) {
	v := order.NewValidator()

	err := v.ValidateOrderCreation(
		"user1",
		"strategy1",
		"BTCUSDT",
		domain.OrderSideBuy,
		domain.OrderTypeMarket,
		0, // Invalid: zero quantity
		0,
	)
	require.Error(t, err)
	assert.True(t, order.IsValidationError(err))
}

func TestValidator_ValidateOrderCreation_LimitOrderNeedsPrice(t *testing.T) {
	v := order.NewValidator()

	err := v.ValidateOrderCreation(
		"user1",
		"strategy1",
		"BTCUSDT",
		domain.OrderSideBuy,
		domain.OrderTypeLimit,
		0.5,
		0, // Invalid: limit order needs price
	)
	require.Error(t, err)
	assert.True(t, order.IsValidationError(err))
}

func TestValidator_ValidateCancellation_CreatedOrder(t *testing.T) {
	v := order.NewValidator()

	ord := &domain.Order{
		ID:     "order-1",
		Status: domain.OrderStatusCreated,
	}

	err := v.ValidateCancellation(ord)
	require.NoError(t, err) // Can cancel created order
}

func TestValidator_ValidateCancellation_FilledOrder(t *testing.T) {
	v := order.NewValidator()

	ord := &domain.Order{
		ID:     "order-1",
		Status: domain.OrderStatusFilled,
	}

	err := v.ValidateCancellation(ord)
	require.Error(t, err) // Cannot cancel filled order
	assert.True(t, order.IsValidationError(err))
}

func TestValidator_ValidateFillUpdate_Valid(t *testing.T) {
	v := order.NewValidator()

	ord := &domain.Order{
		ID:       "order-1",
		Quantity: 1.0,
	}

	err := v.ValidateFillUpdate(ord, 0.5, 50000.0, 10.0)
	require.NoError(t, err)
}

func TestValidator_ValidateFillUpdate_ExceedsQuantity(t *testing.T) {
	v := order.NewValidator()

	ord := &domain.Order{
		ID:       "order-1",
		Quantity: 1.0,
	}

	err := v.ValidateFillUpdate(ord, 1.5, 50000.0, 10.0) // Over-fill
	require.Error(t, err)
	assert.True(t, order.IsValidationError(err))
}

// Test Order Repository

func TestOrderRepository_Create_Idempotency(t *testing.T) {
	repo := infrastructure.NewInMemoryOrderRepository()
	ctx := context.Background()

	ord1 := &domain.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Symbol:        "BTCUSDT",
	}

	// Create first order
	err := repo.Create(ctx, ord1)
	require.NoError(t, err)

	ord1ID := ord1.ID

	// Try to create with same client order ID
	ord2 := &domain.Order{
		ID:            "order-2",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Symbol:        "BTCUSDT",
	}

	err = repo.Create(ctx, ord2)
	require.NoError(t, err)

	// ord2 should now contain ord1's data (idempotent)
	assert.Equal(t, ord1ID, ord2.ID)
}

func TestOrderRepository_GetByClientOrderID(t *testing.T) {
	repo := infrastructure.NewInMemoryOrderRepository()
	ctx := context.Background()

	ord := &domain.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
	}

	err := repo.Create(ctx, ord)
	require.NoError(t, err)

	retrieved, err := repo.GetByClientOrderID(ctx, "client-1")
	require.NoError(t, err)
	assert.Equal(t, "order-1", retrieved.ID)
}

func TestOrderRepository_ListByUser(t *testing.T) {
	repo := infrastructure.NewInMemoryOrderRepository()
	ctx := context.Background()

	ord1 := &domain.Order{
		ID:     "order-1",
		UserID: "user1",
		Status: domain.OrderStatusCreated,
	}
	ord2 := &domain.Order{
		ID:     "order-2",
		UserID: "user1",
		Status: domain.OrderStatusSubmitted,
	}
	ord3 := &domain.Order{
		ID:     "order-3",
		UserID: "user2",
		Status: domain.OrderStatusCreated,
	}

	repo.Create(ctx, ord1)
	repo.Create(ctx, ord2)
	repo.Create(ctx, ord3)

	// List all for user1
	orders, err := repo.ListByUser(ctx, "user1", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(orders))

	// List created orders for user1
	statusFilter := domain.OrderStatusCreated
	orders, err = repo.ListByUser(ctx, "user1", &statusFilter)
	require.NoError(t, err)
	assert.Equal(t, 1, len(orders))
	assert.Equal(t, "order-1", orders[0].ID)
}

func TestOrderRepository_Update(t *testing.T) {
	repo := infrastructure.NewInMemoryOrderRepository()
	ctx := context.Background()

	ord := &domain.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		Status:        domain.OrderStatusCreated,
	}

	repo.Create(ctx, ord)

	ord.Status = domain.OrderStatusSubmitted
	err := repo.Update(ctx, ord)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, "order-1")
	require.NoError(t, err)
	assert.Equal(t, domain.OrderStatusSubmitted, updated.Status)
}

func TestOrderRepository_Delete(t *testing.T) {
	repo := infrastructure.NewInMemoryOrderRepository()
	ctx := context.Background()

	ord := &domain.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
	}

	repo.Create(ctx, ord)

	err := repo.Delete(ctx, "order-1")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "order-1")
	assert.Error(t, err)
}

// Test Order Domain Model

func TestOrder_CanTransition_ValidPaths(t *testing.T) {
	tests := []struct {
		from domain.OrderStatus
		to   domain.OrderStatus
		ok   bool
	}{
		{domain.OrderStatusCreated, domain.OrderStatusSubmitted, true},
		{domain.OrderStatusCreated, domain.OrderStatusCanceled, true},
		{domain.OrderStatusSubmitted, domain.OrderStatusFilled, true},
		{domain.OrderStatusSubmitted, domain.OrderStatusPartialFilled, true},
		{domain.OrderStatusPartialFilled, domain.OrderStatusFilled, true},
		{domain.OrderStatusFilled, domain.OrderStatusCanceled, false},
		{domain.OrderStatusFilled, domain.OrderStatusSubmitted, false},
	}

	for _, tt := range tests {
		ord := &domain.Order{Status: tt.from}
		result := ord.CanTransition(tt.to)
		assert.Equal(t, tt.ok, result,
			"transition %s -> %s", tt.from, tt.to)
	}
}

func TestOrder_IsFinal(t *testing.T) {
	tests := []struct {
		status domain.OrderStatus
		final  bool
	}{
		{domain.OrderStatusCreated, false},
		{domain.OrderStatusSubmitted, false},
		{domain.OrderStatusPartialFilled, false},
		{domain.OrderStatusFilled, true},
		{domain.OrderStatusCanceled, true},
		{domain.OrderStatusRejected, true},
		{domain.OrderStatusClosed, true},
	}

	for _, tt := range tests {
		ord := &domain.Order{Status: tt.status}
		assert.Equal(t, tt.final, ord.IsFinal(),
			"status: "+string(tt.status))
	}
}
