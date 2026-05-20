package tests

import (
	"context"
	"testing"
	"time"

	"github.com/3122380051/golang-microservices/internal/application/execution"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
	"github.com/3122380051/golang-microservices/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecution_StateTransitions(t *testing.T) {
	exec := &domain.Execution{
		ID:               "exec-1",
		Status:           domain.ExecutionStatusCreated,
		OriginalQuantity: 1.0,
	}

	// Created -> Submitting
	err := exec.TransitionTo(domain.ExecutionStatusSubmitting)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusSubmitting, exec.Status)

	// Submitting -> Submitted
	err = exec.TransitionTo(domain.ExecutionStatusSubmitted)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusSubmitted, exec.Status)
	assert.NotNil(t, exec.SubmittedAt)

	// Submitted -> PartialFilled
	err = exec.TransitionTo(domain.ExecutionStatusPartialFilled)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusPartialFilled, exec.Status)
	assert.NotNil(t, exec.FirstFilledAt)

	// PartialFilled -> Filled
	err = exec.TransitionTo(domain.ExecutionStatusFilled)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusFilled, exec.Status)

	// Filled -> Closed
	err = exec.TransitionTo(domain.ExecutionStatusClosed)
	require.NoError(t, err)
	assert.Equal(t, domain.ExecutionStatusClosed, exec.Status)
	assert.NotNil(t, exec.ClosedAt)
}

func TestExecution_InvalidTransition(t *testing.T) {
	exec := &domain.Execution{
		ID:     "exec-1",
		Status: domain.ExecutionStatusFilled,
	}

	// Terminal state cannot transition
	err := exec.TransitionTo(domain.ExecutionStatusSubmitted)
	require.Error(t, err)
	assert.Equal(t, domain.ExecutionStatusFilled, exec.Status)
}

func TestExecution_IsFinal(t *testing.T) {
	tests := []struct {
		status   domain.ExecutionStatus
		expected bool
	}{
		{domain.ExecutionStatusCreated, false},
		{domain.ExecutionStatusSubmitting, false},
		{domain.ExecutionStatusSubmitted, false},
		{domain.ExecutionStatusFilled, true},
		{domain.ExecutionStatusClosed, true},
		{domain.ExecutionStatusCanceled, true},
		{domain.ExecutionStatusFailed, true},
	}

	for _, tt := range tests {
		exec := &domain.Execution{Status: tt.status}
		assert.Equal(t, tt.expected, exec.IsFinal(), "status: "+string(tt.status))
	}
}

func TestExecutionRepository_Create_Idempotency(t *testing.T) {
	repo := infrastructure.NewInMemoryExecutionRepository()
	ctx := context.Background()

	exec1 := &domain.Execution{
		ID:            "exec-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Symbol:        "BTCUSDT",
	}

	// Create first execution
	err := repo.Create(ctx, exec1)
	require.NoError(t, err)

	exec1ID := exec1.ID

	// Try to create with same client order ID
	exec2 := &domain.Execution{
		ID:            "exec-2",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Symbol:        "BTCUSDT",
	}

	err = repo.Create(ctx, exec2)
	require.NoError(t, err)

	// exec2 should now contain exec1's data (idempotent)
	assert.Equal(t, exec1ID, exec2.ID)
}

func TestExecutionRepository_GetByClientOrderID(t *testing.T) {
	repo := infrastructure.NewInMemoryExecutionRepository()
	ctx := context.Background()

	exec := &domain.Execution{
		ID:            "exec-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
	}

	err := repo.Create(ctx, exec)
	require.NoError(t, err)

	retrieved, err := repo.GetByClientOrderID(ctx, "client-1")
	require.NoError(t, err)
	assert.Equal(t, "exec-1", retrieved.ID)
}

func TestExecutionRepository_ListByStatus(t *testing.T) {
	repo := infrastructure.NewInMemoryExecutionRepository()
	ctx := context.Background()

	exec1 := &domain.Execution{
		ID:            "exec-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Status:        domain.ExecutionStatusSubmitted,
	}
	exec2 := &domain.Execution{
		ID:            "exec-2",
		ClientOrderID: "client-2",
		UserID:        "user1",
		Status:        domain.ExecutionStatusPartialFilled,
	}
	exec3 := &domain.Execution{
		ID:            "exec-3",
		ClientOrderID: "client-3",
		UserID:        "user2",
		Status:        domain.ExecutionStatusSubmitted,
	}

	repo.Create(ctx, exec1)
	repo.Create(ctx, exec2)
	repo.Create(ctx, exec3)

	// List all submitted
	submitted, err := repo.ListByStatus(ctx, domain.ExecutionStatusSubmitted)
	require.NoError(t, err)
	assert.Equal(t, 2, len(submitted))

	// List all partial filled
	partial, err := repo.ListByStatus(ctx, domain.ExecutionStatusPartialFilled)
	require.NoError(t, err)
	assert.Equal(t, 1, len(partial))
}

func TestExecutionRepository_ListByUser(t *testing.T) {
	repo := infrastructure.NewInMemoryExecutionRepository()
	ctx := context.Background()

	exec1 := &domain.Execution{
		ID:            "exec-1",
		ClientOrderID: "client-1",
		UserID:        "user1",
		Status:        domain.ExecutionStatusSubmitted,
	}
	exec2 := &domain.Execution{
		ID:            "exec-2",
		ClientOrderID: "client-2",
		UserID:        "user1",
		Status:        domain.ExecutionStatusFilled,
	}
	exec3 := &domain.Execution{
		ID:            "exec-3",
		ClientOrderID: "client-3",
		UserID:        "user2",
		Status:        domain.ExecutionStatusSubmitted,
	}

	repo.Create(ctx, exec1)
	repo.Create(ctx, exec2)
	repo.Create(ctx, exec3)

	// List all for user1
	user1Execs, err := repo.ListByUser(ctx, "user1", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, len(user1Execs))

	// List filled for user1
	statusFilled := domain.ExecutionStatusFilled
	filledExecs, err := repo.ListByUser(ctx, "user1", &statusFilled)
	require.NoError(t, err)
	assert.Equal(t, 1, len(filledExecs))
	assert.Equal(t, "exec-2", filledExecs[0].ID)
}

func TestSubmitter_Idempotency(t *testing.T) {
	appLogger := logger.New("info")
	mockAdapter := &MockExchangeAdapter{}
	submitter := execution.NewSubmitter(appLogger, mockAdapter, 3, time.Millisecond*10)

	req := &domain.SubmissionRequest{
		ExecutionID:   "exec-1",
		ClientOrderID: "client-1",
		Symbol:        "BTCUSDT",
		Side:          domain.OrderSideBuy,
		Quantity:      1.0,
		OrderType:     domain.OrderTypeMarket,
	}

	// First submission succeeds
	result, err := submitter.Submit(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ExchangeOrderID)
	assert.Equal(t, 1, submitter.GetAttemptCount("exec-1"))

	// Verify attempt count
	assert.Equal(t, 1, submitter.GetAttemptCount("exec-1"))
}

// MockExchangeAdapter for testing
type MockExchangeAdapter struct {
	submitOrderFn    func(ctx context.Context, req *exchange.OrderRequest) (string, error)
	getOrderStatusFn func(ctx context.Context, symbol, orderID string) (*exchange.OrderStatus, error)
	getTickerFn      func(ctx context.Context, symbol string) (domain.MarketPrice, error)
	getCandlesFn     func(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error)
	getOrderBookFn   func(ctx context.Context, symbol string, limit int) (domain.OrderBook, error)
}

func (m *MockExchangeAdapter) SubmitOrder(ctx context.Context, req *exchange.OrderRequest) (string, error) {
	if m.submitOrderFn != nil {
		return m.submitOrderFn(ctx, req)
	}
	return "binance-order-123", nil
}

func (m *MockExchangeAdapter) GetOrderStatus(ctx context.Context, symbol, orderID string) (*exchange.OrderStatus, error) {
	if m.getOrderStatusFn != nil {
		return m.getOrderStatusFn(ctx, symbol, orderID)
	}
	return &exchange.OrderStatus{
		OrderID:     orderID,
		Symbol:      symbol,
		Status:      "FILLED",
		Quantity:    1.0,
		ExecutedQty: 1.0,
		Fills: []exchange.Fill{
			{
				TradeID:  "trade-1",
				Quantity: 1.0,
				Price:    50000.0,
				Fee:      10.0,
				FeeAsset: "USDT",
				Time:     time.Now().UnixMilli(),
			},
		},
	}, nil
}

func (m *MockExchangeAdapter) GetTicker(ctx context.Context, symbol string) (domain.MarketPrice, error) {
	if m.getTickerFn != nil {
		return m.getTickerFn(ctx, symbol)
	}
	return domain.MarketPrice{}, nil
}

func (m *MockExchangeAdapter) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error) {
	if m.getCandlesFn != nil {
		return m.getCandlesFn(ctx, symbol, interval, limit)
	}
	return nil, nil
}

func (m *MockExchangeAdapter) GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error) {
	if m.getOrderBookFn != nil {
		return m.getOrderBookFn(ctx, symbol, limit)
	}
	return domain.OrderBook{}, nil
}

// Ensure MockExchangeAdapter implements Adapter
var _ exchange.Adapter = (*MockExchangeAdapter)(nil)
