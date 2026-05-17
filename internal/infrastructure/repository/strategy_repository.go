package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// StrategyRepository stores strategy definitions in-memory for the current phase.
type StrategyRepository struct {
	mu         sync.RWMutex
	strategies map[string]domain.Strategy
	nextID     int
}

func NewStrategyRepository() *StrategyRepository {
	return &StrategyRepository{strategies: make(map[string]domain.Strategy)}
}

func (r *StrategyRepository) Create(ctx context.Context, strategy domain.Strategy) (domain.Strategy, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	strategy.ID = fmt.Sprintf("str-%d", r.nextID)
	strategy.Name = strings.TrimSpace(strategy.Name)
	strategy.Symbol = strings.ToUpper(strings.TrimSpace(strategy.Symbol))
	strategy.Active = strategy.Active
	strategy.CreatedAt = time.Now().UTC()
	strategy.UpdatedAt = strategy.CreatedAt
	if len(strategy.Config) == 0 {
		strategy.Config = json.RawMessage(`{}`)
	}
	r.strategies[strategy.ID] = strategy
	return strategy, nil
}

func (r *StrategyRepository) GetByID(ctx context.Context, id string) (domain.Strategy, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy, ok := r.strategies[id]
	if !ok {
		return domain.Strategy{}, errors.New("not found")
	}
	return strategy, nil
}

func (r *StrategyRepository) List(ctx context.Context) ([]domain.Strategy, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Strategy, 0, len(r.strategies))
	for _, item := range r.strategies {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (r *StrategyRepository) ListActive(ctx context.Context) ([]domain.Strategy, error) {
	items, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Strategy, 0, len(items))
	for _, item := range items {
		if item.Active {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *StrategyRepository) Activate(ctx context.Context, id string) (domain.Strategy, error) {
	return r.toggle(ctx, id, true)
}

func (r *StrategyRepository) Deactivate(ctx context.Context, id string) (domain.Strategy, error) {
	return r.toggle(ctx, id, false)
}

func (r *StrategyRepository) toggle(ctx context.Context, id string, active bool) (domain.Strategy, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	strategy, ok := r.strategies[id]
	if !ok {
		return domain.Strategy{}, errors.New("not found")
	}
	strategy.Active = active
	strategy.UpdatedAt = time.Now().UTC()
	r.strategies[id] = strategy
	return strategy, nil
}
