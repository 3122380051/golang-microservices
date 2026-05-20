package execution

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
)

// Submitter handles order submission with retry logic
type Submitter struct {
	logger       *slog.Logger
	adapter      exchange.Adapter
	maxRetries   int
	baseInterval time.Duration
	attemptsMu   sync.RWMutex
	attempts     map[string]int // executionID -> attempt count
}

// NewSubmitter creates a new submitter with retry configuration
func NewSubmitter(logger *slog.Logger, adapter exchange.Adapter, maxRetries int, baseInterval time.Duration) *Submitter {
	return &Submitter{
		logger:       logger,
		adapter:      adapter,
		maxRetries:   maxRetries,
		baseInterval: baseInterval,
		attempts:     make(map[string]int),
	}
}

// Submit attempts to submit an order with exponential backoff retry logic
func (s *Submitter) Submit(ctx context.Context, req *domain.SubmissionRequest) (*domain.SubmissionResult, error) {
	s.attemptsMu.Lock()
	s.attempts[req.ExecutionID] = 0
	s.attemptsMu.Unlock()

	var lastErr error

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		s.logger.Info("submission attempt",
			"execution_id", req.ExecutionID,
			"attempt", attempt+1,
			"max_retries", s.maxRetries+1,
		)

		// Update attempt count
		s.attemptsMu.Lock()
		s.attempts[req.ExecutionID] = attempt + 1
		s.attemptsMu.Unlock()

		// Try to submit
		exchangeOrderID, err := s.adapter.SubmitOrder(ctx, &exchange.OrderRequest{
			ClientOrderID: req.ClientOrderID,
			Symbol:        req.Symbol,
			Side:          string(req.Side),
			Quantity:      req.Quantity,
			Type:          string(req.OrderType),
			Price:         req.Price,
			TimeInForce:   "GTC", // Good Till Cancelled
		})

		if err == nil {
			s.logger.Info("order submitted successfully",
				"execution_id", req.ExecutionID,
				"exchange_order_id", exchangeOrderID,
				"attempt", attempt+1,
			)

			return &domain.SubmissionResult{
				ExecutionID:     req.ExecutionID,
				ExchangeOrderID: exchangeOrderID,
				ClientOrderID:   req.ClientOrderID,
				Status:          domain.ExecutionStatusSubmitted,
				SubmittedAt:     time.Now(),
			}, nil
		}

		lastErr = err
		s.logger.Warn("submission attempt failed",
			"error", err,
			"execution_id", req.ExecutionID,
			"attempt", attempt+1,
		)

		// Don't retry if this was the last attempt
		if attempt >= s.maxRetries {
			break
		}

		// Exponential backoff: 1s, 2s, 4s
		backoff := s.baseInterval * time.Duration(1<<uint(attempt))
		s.logger.Info("retrying after backoff",
			"execution_id", req.ExecutionID,
			"backoff_ms", backoff.Milliseconds(),
		)

		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, fmt.Errorf("submission canceled during backoff: %w", ctx.Err())
		}
	}

	return nil, fmt.Errorf("submission failed after %d attempts: %w", s.maxRetries+1, lastErr)
}

// GetAttemptCount returns the number of attempts for an execution
func (s *Submitter) GetAttemptCount(executionID string) int {
	s.attemptsMu.RLock()
	defer s.attemptsMu.RUnlock()
	return s.attempts[executionID]
}

// ClearAttempts clears the attempt count for completed executions (cleanup)
func (s *Submitter) ClearAttempts(executionID string) {
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()
	delete(s.attempts, executionID)
}
