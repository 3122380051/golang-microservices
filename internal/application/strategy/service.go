package strategy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/domain/event"
)

// Repository provides strategy persistence operations.
type Repository interface {
	Create(context.Context, domain.Strategy) (domain.Strategy, error)
	GetByID(context.Context, string) (domain.Strategy, error)
	List(context.Context) ([]domain.Strategy, error)
	ListActive(context.Context) ([]domain.Strategy, error)
	Activate(context.Context, string) (domain.Strategy, error)
	Deactivate(context.Context, string) (domain.Strategy, error)
}

// Publisher sends generated signals to Kafka.
type Publisher interface {
	PublishJSON(context.Context, string, string, any) error
}

// Engine evaluates one strategy type.
type Engine interface {
	Type() domain.StrategyType
	Evaluate(strategy domain.Strategy, price float64, state any) (domain.Signal, any, bool, error)
}

type strategyRuntimeState struct {
	engine Engine
	state  any
}

// Service orchestrates strategy CRUD and market event evaluation.
type Service struct {
	repo           Repository
	publisher      Publisher
	engines        map[domain.StrategyType]Engine
	mu             sync.Mutex
	runtimeState   map[string]*strategyRuntimeState
	processedEvent map[string]struct{}
}

func NewService(repo Repository, publisher Publisher, engines ...Engine) *Service {
	engineMap := make(map[domain.StrategyType]Engine, len(engines))
	for _, engine := range engines {
		engineMap[engine.Type()] = engine
	}
	if _, ok := engineMap[domain.StrategyTypeEMACross]; !ok {
		engineMap[domain.StrategyTypeEMACross] = NewEMAStrategy()
	}
	if _, ok := engineMap[domain.StrategyTypeRSI]; !ok {
		engineMap[domain.StrategyTypeRSI] = NewRSIStrategy()
	}

	return &Service{
		repo:           repo,
		publisher:      publisher,
		engines:        engineMap,
		runtimeState:   make(map[string]*strategyRuntimeState),
		processedEvent: make(map[string]struct{}),
	}
}

func (s *Service) CreateStrategy(ctx context.Context, strategy domain.Strategy) (domain.Strategy, error) {
	if strategy.Type == "" {
		return domain.Strategy{}, errors.New("strategy type is required")
	}
	if strategy.Symbol == "" {
		return domain.Strategy{}, errors.New("symbol is required")
	}
	if _, ok := s.engines[strategy.Type]; !ok {
		return domain.Strategy{}, fmt.Errorf("unsupported strategy type: %s", strategy.Type)
	}
	return s.repo.Create(ctx, strategy)
}

func (s *Service) GetStrategy(ctx context.Context, id string) (domain.Strategy, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListStrategies(ctx context.Context) ([]domain.Strategy, error) {
	return s.repo.List(ctx)
}

func (s *Service) Activate(ctx context.Context, id string) (domain.Strategy, error) {
	return s.repo.Activate(ctx, id)
}

func (s *Service) Deactivate(ctx context.Context, id string) (domain.Strategy, error) {
	return s.repo.Deactivate(ctx, id)
}

func (s *Service) HandleMarketPriceUpdated(ctx context.Context, envelope event.Envelope) ([]domain.Signal, error) {
	if envelope.EventID == "" {
		return nil, errors.New("event id is required")
	}

	s.mu.Lock()
	if _, exists := s.processedEvent[envelope.EventID]; exists {
		s.mu.Unlock()
		return nil, nil
	}
	s.processedEvent[envelope.EventID] = struct{}{}
	s.mu.Unlock()

	var marketEvent event.MarketPriceUpdated
	if err := json.Unmarshal(envelope.Payload, &marketEvent); err != nil {
		return nil, fmt.Errorf("decode market event: %w", err)
	}

	strategies, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	var signals []domain.Signal
	for _, strategyItem := range strategies {
		if strategyItem.Symbol != marketEvent.Symbol {
			continue
		}
		engine := s.engines[strategyItem.Type]
		if engine == nil {
			continue
		}

		stateKey := strategyItem.ID + ":" + strategyItem.Symbol
		state := s.runtimeState[stateKey]
		if state == nil {
			state = &strategyRuntimeState{engine: engine}
			s.runtimeState[stateKey] = state
		}

		signal, nextState, shouldEmit, err := engine.Evaluate(strategyItem, marketEvent.Price, state.state)
		if err != nil {
			continue
		}
		state.state = nextState
		if !shouldEmit || signal.Action == domain.SignalActionHold {
			continue
		}

		signal.ID = fmt.Sprintf("sig-%d", time.Now().UnixNano())
		signal.EventID = envelope.EventID
		signal.TraceID = envelope.TraceID
		signal.CreatedAt = time.Now().UTC()
		signal.Metadata = mergeMetadata(signal.Metadata, map[string]any{
			"market_event_id": envelope.EventID,
			"market_ts":       marketEvent.Ts,
		})
		signals = append(signals, signal)
		if s.publisher != nil {
			_ = s.publisher.PublishJSON(ctx, event.TopicStrategySignalGenerated, signal.StrategyID, signal)
		}
	}

	return signals, nil
}

func mergeMetadata(base, extra map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for key, value := range extra {
		base[key] = value
	}
	return base
}
