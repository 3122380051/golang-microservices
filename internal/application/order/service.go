package order

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
)

// Service coordinates order creation, state management, and event publishing
type Service struct {
	logger                  infrastructure.Logger
	repository              domain.OrderRepository
	stateMachine            *StateMachine
	validator               *Validator
	producer                broker.Producer
	consumer                broker.Consumer
	processedRiskDecisionMu sync.RWMutex
	processedRiskDecisions  map[string]bool // riskDecisionID -> processed (idempotency)
	orderCacheMu            sync.RWMutex
	orderCache              map[string]*domain.Order // clientOrderID -> Order
}

// NewService creates a new order service
func NewService(
	logger infrastructure.Logger,
	repository domain.OrderRepository,
	producer broker.Producer,
	consumer broker.Consumer,
) *Service {
	return &Service{
		logger:                 logger,
		repository:             repository,
		stateMachine:           NewStateMachine(),
		validator:              NewValidator(),
		producer:               producer,
		consumer:               consumer,
		processedRiskDecisions: make(map[string]bool),
		orderCache:             make(map[string]*domain.Order),
	}
}

// ConsumeRiskDecisions starts consuming risk.order.approved/rejected events
func (s *Service) ConsumeRiskDecisions(ctx context.Context) {
	s.logger.Info("starting to consume risk decisions")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping risk decision consumer")
			return
		default:
		}

		// Try to consume from risk.order.approved
		msg, err := s.consumer.ReadMessage(ctx, time.Second*10)
		if err != nil {
			s.logger.Debug("consumer read error or timeout", "error", err)
			continue
		}

		// Parse the risk decision event
		var event struct {
			EventID         string `json:"event_id"`
			SignalID        string `json:"signal_id"`
			UserID          string `json:"user_id"`
			StrategyID      string `json:"strategy_id"`
			Symbol          string `json:"symbol"`
			Side            string `json:"side"`
			Quantity        float64 `json:"quantity"`
			IsApproved      bool `json:"is_approved"`
			RejectionReason string `json:"rejection_reason"`
			TraceID         string `json:"trace_id"`
			Timestamp       time.Time `json:"timestamp"`
		}

		if err := json.Unmarshal(msg.Value, &event); err != nil {
			s.logger.Error("failed to parse risk decision event", "error", err)
			continue
		}

		// Handle the decision
		if event.IsApproved {
			s.handleApprovedSignal(ctx, &event)
		} else {
			s.handleRejectedSignal(ctx, &event)
		}
	}
}

// handleApprovedSignal creates an order from an approved risk decision
func (s *Service) handleApprovedSignal(ctx context.Context, event *interface{}) {
	e := *event.(*struct {
		EventID         string
		SignalID        string
		UserID          string
		StrategyID      string
		Symbol          string
		Side            string
		Quantity        float64
		IsApproved      bool
		RejectionReason string
		TraceID         string
		Timestamp       time.Time
	})

	traceID := e.TraceID
	riskDecisionID := e.EventID

	// Idempotency: skip if already processed
	s.processedRiskDecisionMu.Lock()
	if s.processedRiskDecisions[riskDecisionID] {
		s.processedRiskDecisionMu.Unlock()
		s.logger.Debug("risk decision already processed", "risk_decision_id", riskDecisionID)
		return
	}
	s.processedRiskDecisions[riskDecisionID] = true
	s.processedRiskDecisionMu.Unlock()

	s.logger.Info("creating order from approved risk decision",
		"risk_decision_id", riskDecisionID,
		"trace_id", traceID,
		"symbol", e.Symbol,
	)

	// Validate order creation inputs
	side := domain.OrderSideBuy
	if e.Side == "SELL" {
		side = domain.OrderSideSell
	}

	if err := s.validator.ValidateOrderCreation(
		e.UserID, e.StrategyID, e.Symbol, side,
		domain.OrderTypeMarket, e.Quantity, 0,
	); err != nil {
		s.logger.Error("order creation validation failed", "error", err, "trace_id", traceID)
		return
	}

	// Create order
	order, err := s.CreateOrder(ctx, e.UserID, e.StrategyID, e.Symbol, side, e.Quantity, e.SignalID, riskDecisionID)
	if err != nil {
		s.logger.Error("failed to create order", "error", err, "trace_id", traceID)
		return
	}

	s.logger.Info("order created from risk decision",
		"order_id", order.ID,
		"client_order_id", order.ClientOrderID,
		"trace_id", traceID,
	)

	// Publish order.created event
	if err := s.publishOrderEvent(ctx, order, "order.created"); err != nil {
		s.logger.Error("failed to publish order.created event", "error", err, "order_id", order.ID)
	}
}

// handleRejectedSignal logs rejected signals (no order created)
func (s *Service) handleRejectedSignal(ctx context.Context, event *interface{}) {
	e := *event.(*struct {
		EventID         string
		SignalID        string
		UserID          string
		StrategyID      string
		Symbol          string
		Side            string
		Quantity        float64
		IsApproved      bool
		RejectionReason string
		TraceID         string
		Timestamp       time.Time
	})

	s.logger.Info("risk decision rejected, not creating order",
		"risk_decision_id", e.EventID,
		"reason", e.RejectionReason,
		"trace_id", e.TraceID,
	)
}

// CreateOrder creates a new order
func (s *Service) CreateOrder(
	ctx context.Context,
	userID, strategyID, symbol string,
	side domain.OrderSide,
	quantity float64,
	signalID, riskDecisionID string,
) (*domain.Order, error) {

	// Validate
	if err := s.validator.ValidateOrderCreation(userID, strategyID, symbol, side, domain.OrderTypeMarket, quantity, 0); err != nil {
		return nil, err
	}

	// Generate IDs
	correlationID := uuid.New().String()
	clientOrderID := uuid.New().String()

	// Create order
	order := domain.NewOrder(userID, strategyID, symbol, side, domain.OrderTypeMarket, quantity, 0)
	order.ID = uuid.New().String()
	order.ClientOrderID = clientOrderID
	order.CorrelationID = correlationID
	order.SignalID = signalID
	order.RiskDecisionID = riskDecisionID

	// Persist
	if err := s.repository.Create(ctx, order); err != nil {
		return nil, err
	}

	// Cache by client order ID
	s.orderCacheMu.Lock()
	s.orderCache[clientOrderID] = order
	s.orderCacheMu.Unlock()

	return order, nil
}

// GetOrder retrieves an order by ID
func (s *Service) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.repository.GetByID(ctx, orderID)
}

// GetOrderByClientID retrieves an order by client order ID (idempotency key)
func (s *Service) GetOrderByClientID(ctx context.Context, clientOrderID string) (*domain.Order, error) {
	// Check cache first
	s.orderCacheMu.RLock()
	if order, exists := s.orderCache[clientOrderID]; exists {
		s.orderCacheMu.RUnlock()
		return order, nil
	}
	s.orderCacheMu.RUnlock()

	// Query repository
	return s.repository.GetByClientOrderID(ctx, clientOrderID)
}

// ListOrdersByUser lists orders for a user, optionally filtered by status
func (s *Service) ListOrdersByUser(ctx context.Context, userID string, status *domain.OrderStatus) ([]domain.Order, error) {
	return s.repository.ListByUser(ctx, userID, status)
}

// ListOrdersByStrategy lists orders for a strategy
func (s *Service) ListOrdersByStrategy(ctx context.Context, strategyID string, status *domain.OrderStatus) ([]domain.Order, error) {
	return s.repository.ListByStrategy(ctx, strategyID, status)
}

// UpdateOrderFill updates order with execution fill information
func (s *Service) UpdateOrderFill(
	ctx context.Context,
	orderID string,
	executedQty, avgPrice, fees float64,
	newStatus domain.OrderStatus,
) (*domain.Order, error) {

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Validate fill update
	if err := s.validator.ValidateFillUpdate(order, executedQty, avgPrice, fees); err != nil {
		return nil, err
	}

	// Transition state
	if err := s.stateMachine.Transition(order, newStatus, ""); err != nil {
		return nil, err
	}

	// Update fill details
	order.ExecutedQuantity = executedQty
	order.AverageFillPrice = avgPrice
	order.Fees = fees
	if executedQty > 0 {
		order.ExecutedValue = executedQty * avgPrice
	}

	// Persist
	if err := s.repository.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(ctx context.Context, orderID, reason string) (*domain.Order, error) {
	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Validate cancellation
	if err := s.validator.ValidateCancellation(order); err != nil {
		return nil, err
	}

	// Transition to canceled
	if err := s.stateMachine.Transition(order, domain.OrderStatusCanceled, reason); err != nil {
		return nil, err
	}

	// Persist
	if err := s.repository.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish order.canceled event
	if err := s.publishOrderEvent(ctx, order, "order.canceled"); err != nil {
		s.logger.Error("failed to publish order.canceled event", "error", err, "order_id", order.ID)
	}

	return order, nil
}

// TransitionOrder transitions an order to a new status
func (s *Service) TransitionOrder(
	ctx context.Context,
	orderID string,
	newStatus domain.OrderStatus,
	reason string,
) (*domain.Order, error) {

	order, err := s.repository.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if err := s.stateMachine.Transition(order, newStatus, reason); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, order); err != nil {
		return nil, err
	}

	return order, nil
}

// publishOrderEvent publishes an order event to Kafka
func (s *Service) publishOrderEvent(ctx context.Context, order *domain.Order, eventType string) error {
	event := map[string]interface{}{
		"event_type":       eventType,
		"event_id":         uuid.New().String(),
		"correlation_id":   order.CorrelationID,
		"order_id":         order.ID,
		"client_order_id":  order.ClientOrderID,
		"user_id":          order.UserID,
		"strategy_id":      order.StrategyID,
		"symbol":           order.Symbol,
		"side":             order.Side,
		"quantity":         order.Quantity,
		"status":           order.Status,
		"executed_quantity": order.ExecutedQuantity,
		"fees":             order.Fees,
		"timestamp":        time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.producer.PublishMessage(ctx, eventType, order.ID, payload)
}

var (
	ErrOrderNotFound = &ServiceError{"order not found"}
)

type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
