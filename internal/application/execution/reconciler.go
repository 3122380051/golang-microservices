package execution

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
)

// Reconciler handles polling for order fills from the exchange
type Reconciler struct {
	logger       *slog.Logger
	adapter      exchange.Adapter
	pollInterval time.Duration
}

// NewReconciler creates a new reconciler
func NewReconciler(logger *slog.Logger, adapter exchange.Adapter, pollInterval time.Duration) *Reconciler {
	return &Reconciler{
		logger:       logger,
		adapter:      adapter,
		pollInterval: pollInterval,
	}
}

// GetFills retrieves fills for an order from the exchange
func (r *Reconciler) GetFills(ctx context.Context, exchangeOrderID, symbol string) ([]domain.FillRecord, error) {
	if exchangeOrderID == "" {
		return nil, fmt.Errorf("exchange_order_id is required")
	}

	// Query exchange for order status and fills
	orderStatus, err := r.adapter.GetOrderStatus(ctx, symbol, exchangeOrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	// Convert exchange fills to domain fills
	fills := make([]domain.FillRecord, 0, len(orderStatus.Fills))
	now := time.Now()

	for _, exFill := range orderStatus.Fills {
		fills = append(fills, domain.FillRecord{
			TradeID:      exFill.TradeID,
			ExecutionID:  exchangeOrderID,
			Quantity:     exFill.Quantity,
			Price:        exFill.Price,
			Fee:          exFill.Fee,
			FeeAsset:     exFill.FeeAsset,
			IsCommission: true,
			FilledAt:     time.UnixMilli(exFill.Time),
			ReceivedAt:   now,
		})
	}

	return fills, nil
}

// GetPollInterval returns the polling interval
func (r *Reconciler) GetPollInterval() time.Duration {
	return r.pollInterval
}
