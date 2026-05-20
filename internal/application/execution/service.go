package execution

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Service coordinates order execution at the exchange
type Service struct {
	logger            *slog.Logger
	repository        domain.ExecutionRepository
	producer          *broker.KafkaProducer
	consumer          *broker.KafkaConsumer
	exchangeAdapter   exchange.Adapter
	submitter         *Submitter
	reconciler        *Reconciler
	processedOrdersMu sync.RWMutex
	processedOrders   map[string]bool // orderID -> processed (idempotency)
	executionCacheMu  sync.RWMutex
	executionCache    map[string]*domain.Execution // clientOrderID -> Execution
}

// OrderCreatedEvent from Order Service
type OrderCreatedEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	ClientOrderID string    `json:"client_order_id"`
	CorrelationID string    `json:"correlation_id"`
	UserID        string    `json:"user_id"`
	StrategyID    string    `json:"strategy_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	OrderType     string    `json:"order_type"`
	Quantity      float64   `json:"quantity"`
	Price         float64   `json:"price"`
	CreatedAt     time.Time `json:"created_at"`
}

// NewService creates a new execution service
func NewService(
	logger *slog.Logger,
	repository domain.ExecutionRepository,
	producer *broker.KafkaProducer,
	consumer *broker.KafkaConsumer,
	adapter exchange.Adapter,
) *Service {
	submitter := NewSubmitter(logger, adapter, 3, time.Second)  // 3 retries, exponential backoff
	reconciler := NewReconciler(logger, adapter, 5*time.Second) // Poll every 5 seconds

	return &Service{
		logger:          logger,
		repository:      repository,
		producer:        producer,
		consumer:        consumer,
		exchangeAdapter: adapter,
		submitter:       submitter,
		reconciler:      reconciler,
		processedOrders: make(map[string]bool),
		executionCache:  make(map[string]*domain.Execution),
	}
}

// ConsumeOrderCreated consumes order.created events and submits to exchange
func (s *Service) ConsumeOrderCreated(ctx context.Context) {
	s.logger.Info("starting to consume order.created events")

	handler := func(ctx context.Context, msg kafka.Message) error {
		var event OrderCreatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			s.logger.Error("failed to parse order.created event", "error", err)
			return err
		}

		s.handleOrderCreated(ctx, &event)
		return nil
	}

	if err := s.consumer.Consume(ctx, handler); err != nil {
		s.logger.Error("consumer error", "error", err)
	}
}

// handleOrderCreated creates an execution and submits it to the exchange
func (s *Service) handleOrderCreated(ctx context.Context, event *OrderCreatedEvent) {
	// Idempotency: skip if order already processed
	s.processedOrdersMu.Lock()
	if s.processedOrders[event.OrderID] {
		s.processedOrdersMu.Unlock()
		s.logger.Debug("order already processed", "order_id", event.OrderID)
		return
	}
	s.processedOrders[event.OrderID] = true
	s.processedOrdersMu.Unlock()

	traceID := uuid.New().String()
	s.logger.Info("creating execution for order",
		"order_id", event.OrderID,
		"trace_id", traceID,
		"symbol", event.Symbol,
	)

	// Create execution
	execution := domain.NewExecution(
		event.OrderID,
		event.ClientOrderID,
		event.UserID,
		event.Symbol,
		domain.OrderSide(event.Side),
		event.Quantity,
	)
	execution.ID = uuid.New().String()
	execution.CorrelationID = event.CorrelationID

	// Persist
	if err := s.repository.Create(ctx, execution); err != nil {
		s.logger.Error("failed to create execution", "error", err, "order_id", event.OrderID)
		return
	}

	// Cache
	s.executionCacheMu.Lock()
	s.executionCache[event.ClientOrderID] = execution
	s.executionCacheMu.Unlock()

	// Submit to exchange (with retries)
	s.submitExecution(ctx, execution, traceID)
}

// submitExecution attempts to submit an order to the exchange with retry logic
func (s *Service) submitExecution(ctx context.Context, execution *domain.Execution, traceID string) {
	s.logger.Info("submitting execution to exchange",
		"execution_id", execution.ID,
		"symbol", execution.Symbol,
		"trace_id", traceID,
	)

	req := &domain.SubmissionRequest{
		ExecutionID:   execution.ID,
		ClientOrderID: execution.ClientOrderID,
		Symbol:        execution.Symbol,
		Side:          execution.Side,
		Quantity:      execution.OriginalQuantity,
		OrderType:     domain.OrderTypeMarket,
	}

	result, err := s.submitter.Submit(ctx, req)
	if err != nil {
		s.logger.Error("submission failed",
			"error", err,
			"execution_id", execution.ID,
			"trace_id", traceID,
		)
		execution.Status = domain.ExecutionStatusFailed
		execution.LastAttemptError = err.Error()
		_ = s.repository.Update(ctx, execution)
		return
	}

	// Update execution with exchange details
	execution.ExchangeOrderID = result.ExchangeOrderID
	execution.Status = domain.ExecutionStatusSubmitted
	now := time.Now()
	execution.SubmittedAt = &now
	execution.AttemptCount = s.submitter.GetAttemptCount(execution.ID)

	if err := s.repository.Update(ctx, execution); err != nil {
		s.logger.Error("failed to update execution after submission", "error", err, "execution_id", execution.ID)
		return
	}

	// Publish execution.submitted event
	s.publishExecutionEvent(ctx, execution, "execution.submitted")
}

// StartReconciliationLoop starts polling for fills
func (s *Service) StartReconciliationLoop(ctx context.Context) {
	s.logger.Info("starting reconciliation loop")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("reconciliation loop stopped")
			return
		case <-ticker.C:
			s.reconcileExecutions(ctx)
		}
	}
}

// reconcileExecutions polls for fills on active executions
func (s *Service) reconcileExecutions(ctx context.Context) {
	// Get all submitted/partial executions
	submitted, err := s.repository.ListByStatus(ctx, domain.ExecutionStatusSubmitted)
	if err != nil {
		s.logger.Error("failed to list submitted executions", "error", err)
		return
	}

	partial, err := s.repository.ListByStatus(ctx, domain.ExecutionStatusPartialFilled)
	if err != nil {
		s.logger.Error("failed to list partial executions", "error", err)
		return
	}

	executions := append(submitted, partial...)
	if len(executions) == 0 {
		return
	}

	s.logger.Debug("reconciling executions", "count", len(executions))

	for i := range executions {
		exec := &executions[i]
		if exec.ExchangeOrderID == "" {
			continue
		}

		// Get latest status and fills from exchange
		fills, err := s.reconciler.GetFills(ctx, exec.ExchangeOrderID, exec.Symbol)
		if err != nil {
			s.logger.Error("failed to get fills",
				"error", err,
				"execution_id", exec.ID,
				"exchange_order_id", exec.ExchangeOrderID,
			)
			continue
		}

		// Update execution with fills
		if len(fills) > 0 {
			s.applyFills(ctx, exec, fills)
		}
	}
}

// applyFills updates execution with fills from exchange
func (s *Service) applyFills(ctx context.Context, execution *domain.Execution, fills []domain.FillRecord) {
	totalExecutedQty := execution.ExecutedQuantity
	totalExecutedValue := execution.ExecutedValue
	totalFees := execution.Fees

	for _, fill := range fills {
		totalExecutedQty += fill.Quantity
		totalExecutedValue += fill.Quantity * fill.Price
		totalFees += fill.Fee
	}

	// Determine new status
	newStatus := domain.ExecutionStatusPartialFilled
	if totalExecutedQty >= execution.OriginalQuantity {
		totalExecutedQty = execution.OriginalQuantity // Cap at original
		newStatus = domain.ExecutionStatusFilled
	}

	execution.ExecutedQuantity = totalExecutedQty
	execution.ExecutedValue = totalExecutedValue
	execution.Fees = totalFees

	if totalExecutedQty > 0 {
		execution.AverageFillPrice = totalExecutedValue / totalExecutedQty
	}

	if err := execution.TransitionTo(newStatus); err != nil {
		s.logger.Error("failed to transition status",
			"error", err,
			"execution_id", execution.ID,
			"new_status", newStatus,
		)
		return
	}

	if err := s.repository.Update(ctx, execution); err != nil {
		s.logger.Error("failed to update execution with fills",
			"error", err,
			"execution_id", execution.ID,
		)
		return
	}

	// Publish fill event if filled
	if newStatus == domain.ExecutionStatusFilled {
		s.publishExecutionEvent(ctx, execution, "execution.filled")
	}
}

// GetExecution retrieves an execution by ID
func (s *Service) GetExecution(ctx context.Context, executionID string) (*domain.Execution, error) {
	return s.repository.GetByID(ctx, executionID)
}

// ListExecutionsByUser lists executions for a user
func (s *Service) ListExecutionsByUser(ctx context.Context, userID string, status *domain.ExecutionStatus) ([]domain.Execution, error) {
	return s.repository.ListByUser(ctx, userID, status)
}

// publishExecutionEvent publishes an execution event to Kafka
func (s *Service) publishExecutionEvent(ctx context.Context, execution *domain.Execution, eventType string) error {
	event := &domain.ExecutionEvent{
		EventID:          uuid.New().String(),
		ExecutionID:      execution.ID,
		OrderID:          execution.OrderID,
		ClientOrderID:    execution.ClientOrderID,
		CorrelationID:    execution.CorrelationID,
		ExchangeOrderID:  execution.ExchangeOrderID,
		UserID:           execution.UserID,
		Symbol:           execution.Symbol,
		Side:             execution.Side,
		OriginalQty:      execution.OriginalQuantity,
		ExecutedQty:      execution.ExecutedQuantity,
		AverageFillPrice: execution.AverageFillPrice,
		Fees:             execution.Fees,
		Status:           execution.Status,
		EventType:        eventType,
		Timestamp:        time.Now(),
		TraceID:          uuid.New().String(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.producer.Publish(ctx, eventType, []byte(execution.ID), payload)
}

var (
	ErrExecutionNotFound = &ServiceError{"execution not found"}
)

type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
