package risk

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

// Service coordinates risk evaluation and decision publishing
type Service struct {
	logger              infrastructure.Logger
	evaluator           *Evaluator
	policyRepository    domain.RiskPolicyRepository
	portfolioCache      domain.PortfolioCacheProvider
	producer            broker.Producer
	consumer            broker.Consumer
	processedSignalsMu  sync.RWMutex
	processedSignals    map[string]bool // signalID -> processed (idempotency)
	decisionsCacheMu    sync.RWMutex
	decisionsCache      map[string]*domain.RiskDecision // id -> decision
}

// NewService creates a new risk service
func NewService(
	logger infrastructure.Logger,
	evaluator *Evaluator,
	policyRepository domain.RiskPolicyRepository,
	portfolioCache domain.PortfolioCacheProvider,
	producer broker.Producer,
	consumer broker.Consumer,
) *Service {
	return &Service{
		logger:           logger,
		evaluator:        evaluator,
		policyRepository: policyRepository,
		portfolioCache:   portfolioCache,
		producer:         producer,
		consumer:         consumer,
		processedSignals: make(map[string]bool),
		decisionsCache:   make(map[string]*domain.RiskDecision),
	}
}

// ConsumeStrategySignals starts consuming strategy.signal.generated events from Kafka
func (s *Service) ConsumeStrategySignals(ctx context.Context) {
	s.logger.Info("starting to consume strategy signals")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping signal consumer")
			return
		default:
		}

		msg, err := s.consumer.ReadMessage(ctx, time.Second*10)
		if err != nil {
			s.logger.Debug("consumer read error or timeout", "error", err)
			continue
		}

		// Parse the strategy signal
		var signal domain.StrategySignal
		if err := json.Unmarshal(msg.Value, &signal); err != nil {
			s.logger.Error("failed to parse strategy signal", "error", err, "topic", msg.Topic)
			continue
		}

		// Process signal
		s.processStrategySignal(ctx, &signal)
	}
}

// processStrategySignal evaluates a signal and publishes risk decision
func (s *Service) processStrategySignal(ctx context.Context, signal *domain.StrategySignal) {
	// Idempotency: skip if already processed
	s.processedSignalsMu.Lock()
	if s.processedSignals[signal.ID] {
		s.processedSignalsMu.Unlock()
		s.logger.Debug("signal already processed, skipping", "signal_id", signal.ID)
		return
	}
	s.processedSignals[signal.ID] = true
	s.processedSignalsMu.Unlock()

	traceID := uuid.New().String()
	s.logger.Info("processing strategy signal", "signal_id", signal.ID, "trace_id", traceID, "symbol", signal.Symbol)

	// Get applicable risk policy (user + strategy specific, or default)
	policy, err := s.policyRepository.GetByUserAndStrategy(ctx, signal.UserID, signal.StrategyID)
	if err != nil {
		// Fall back to default policy
		policy, err = s.policyRepository.GetDefaultPolicy(ctx)
		if err != nil {
			s.logger.Error("failed to get risk policy", "error", err, "signal_id", signal.ID)
			return
		}
	}

	// Get current portfolio state
	portfolio, err := s.portfolioCache.GetSnapshot(signal.UserID)
	if err != nil {
		s.logger.Error("failed to get portfolio snapshot", "error", err, "user_id", signal.UserID)
		// Use empty portfolio if cache miss (will fail safety checks)
		portfolio = domain.NewPortfolioSnapshot(signal.UserID)
	}

	// Evaluate signal against policy using estimated price
	decision := s.evaluator.EvaluateSignal(policy, portfolio, signal, signal.Confidence)
	decision.ID = uuid.New().String()
	decision.TraceID = traceID
	decision.DecidedAt = time.Now()
	decision.CreatedAt = time.Now()

	// Cache the decision
	s.decisionsCacheMu.Lock()
	s.decisionsCache[decision.ID] = decision
	s.decisionsCacheMu.Unlock()

	// Log detailed checks
	s.logRiskChecks(decision)

	// Publish decision event to appropriate Kafka topic
	if err := s.publishDecision(ctx, decision); err != nil {
		s.logger.Error("failed to publish risk decision", "error", err, "decision_id", decision.ID)
		return
	}

	s.logger.Info(
		"risk decision published",
		"decision_id", decision.ID,
		"approved", decision.IsApproved,
		"reason", decision.RejectionReason,
	)
}

// publishDecision publishes a risk decision to Kafka
func (s *Service) publishDecision(ctx context.Context, decision *domain.RiskDecision) error {
	event := decision.ToEvent()
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	topic := decision.EventType() // "risk.order.approved" or "risk.order.rejected"
	return s.producer.PublishMessage(ctx, topic, event.EventID, payload)
}

// logRiskChecks logs detailed results of all risk checks
func (s *Service) logRiskChecks(decision *domain.RiskDecision) {
	checks := decision.Checks
	s.logger.Info(
		"risk checks completed",
		"decision_id", decision.ID,
		"position_size", checks.PositionSizeCheck.Reason,
		"leverage", checks.LeverageCheck.Reason,
		"margin", checks.MarginCheck.Reason,
		"daily_loss", checks.DailyLossCheck.Reason,
		"exposure", checks.ExposureCheck.Reason,
	)
}

// GetDecision retrieves a cached risk decision
func (s *Service) GetDecision(id string) (*domain.RiskDecision, error) {
	s.decisionsCacheMu.RLock()
	decision, exists := s.decisionsCache[id]
	s.decisionsCacheMu.RUnlock()

	if !exists {
		return nil, ErrDecisionNotFound
	}
	return decision, nil
}

// GetDecisionsBySignal retrieves a decision by signal ID
func (s *Service) GetDecisionsBySignal(signalID string) (*domain.RiskDecision, error) {
	s.decisionsCacheMu.RLock()
	defer s.decisionsCacheMu.RUnlock()

	for _, decision := range s.decisionsCache {
		if decision.SignalID == signalID {
			return decision, nil
		}
	}
	return nil, ErrDecisionNotFound
}

// CreatePolicy creates a new risk policy
func (s *Service) CreatePolicy(ctx context.Context, policy *domain.RiskPolicy) error {
	return s.policyRepository.Create(ctx, policy)
}

// GetPolicy retrieves a risk policy by ID
func (s *Service) GetPolicy(ctx context.Context, id string) (*domain.RiskPolicy, error) {
	return s.policyRepository.GetByID(ctx, id)
}

// UpdatePolicy updates a risk policy
func (s *Service) UpdatePolicy(ctx context.Context, policy *domain.RiskPolicy) error {
	return s.policyRepository.Update(ctx, policy)
}

// DeletePolicy deletes a risk policy
func (s *Service) DeletePolicy(ctx context.Context, id string) error {
	return s.policyRepository.Delete(ctx, id)
}

// ListUserPolicies lists all policies for a user
func (s *Service) ListUserPolicies(ctx context.Context, userID string) ([]domain.RiskPolicy, error) {
	return s.policyRepository.ListByUser(ctx, userID)
}

// InvalidatePortfolioCache invalidates cached portfolio for a user
func (s *Service) InvalidatePortfolioCache(userID string) error {
	return s.portfolioCache.InvalidateSnapshot(userID)
}

var (
	ErrDecisionNotFound = &ServiceError{"decision not found"}
)

type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}
