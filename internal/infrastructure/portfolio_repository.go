package infrastructure

import (
	"context"
	"sync"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// InMemoryPortfolioRepository is an in-memory implementation of PortfolioRepository
type InMemoryPortfolioRepository struct {
	mu         sync.RWMutex
	portfolios map[string]*domain.Portfolio // userID -> Portfolio
}

// NewInMemoryPortfolioRepository creates a new in-memory portfolio repository
func NewInMemoryPortfolioRepository() domain.PortfolioRepository {
	return &InMemoryPortfolioRepository{
		portfolios: make(map[string]*domain.Portfolio),
	}
}

func (r *InMemoryPortfolioRepository) Create(ctx context.Context, portfolio *domain.Portfolio) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.portfolios[portfolio.UserID]; exists {
		return ErrPortfolioAlreadyExists
	}

	r.portfolios[portfolio.UserID] = portfolio
	return nil
}

func (r *InMemoryPortfolioRepository) GetByUserID(ctx context.Context, userID string) (*domain.Portfolio, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	portfolio, exists := r.portfolios[userID]
	if !exists {
		return nil, ErrPortfolioNotFound
	}
	return portfolio, nil
}

func (r *InMemoryPortfolioRepository) Update(ctx context.Context, portfolio *domain.Portfolio) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.portfolios[portfolio.UserID]; !exists {
		return ErrPortfolioNotFound
	}

	r.portfolios[portfolio.UserID] = portfolio
	return nil
}

func (r *InMemoryPortfolioRepository) Delete(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.portfolios[userID]; !exists {
		return ErrPortfolioNotFound
	}

	delete(r.portfolios, userID)
	return nil
}

func (r *InMemoryPortfolioRepository) ListAll(ctx context.Context) ([]domain.Portfolio, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Portfolio, 0, len(r.portfolios))
	for _, portfolio := range r.portfolios {
		result = append(result, *portfolio)
	}
	return result, nil
}

// InMemoryTradeResultRepository is an in-memory implementation of TradeResultRepository
type InMemoryTradeResultRepository struct {
	mu           sync.RWMutex
	tradesByID   map[string]*domain.TradeResult
	tradesByUser map[string][]*domain.TradeResult // userID -> []TradeResult
}

// NewInMemoryTradeResultRepository creates a new in-memory trade result repository
func NewInMemoryTradeResultRepository() domain.TradeResultRepository {
	return &InMemoryTradeResultRepository{
		tradesByID:   make(map[string]*domain.TradeResult),
		tradesByUser: make(map[string][]*domain.TradeResult),
	}
}

func (r *InMemoryTradeResultRepository) Create(ctx context.Context, result *domain.TradeResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tradesByID[result.ID]; exists {
		return ErrTradeResultAlreadyExists
	}

	r.tradesByID[result.ID] = result
	// Note: we can't track by user from trade result directly
	// In production with proper DB schema, would have user_id field
	return nil
}

func (r *InMemoryTradeResultRepository) GetByID(ctx context.Context, id string) (*domain.TradeResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	trade, exists := r.tradesByID[id]
	if !exists {
		return nil, ErrTradeResultNotFound
	}
	return trade, nil
}

func (r *InMemoryTradeResultRepository) ListByUser(ctx context.Context, userID string) ([]domain.TradeResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	trades, exists := r.tradesByUser[userID]
	if !exists {
		return []domain.TradeResult{}, nil
	}

	result := make([]domain.TradeResult, len(trades))
	for i, trade := range trades {
		result[i] = *trade
	}
	return result, nil
}

func (r *InMemoryTradeResultRepository) ListBySymbol(ctx context.Context, symbol string) ([]domain.TradeResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.TradeResult
	for _, trade := range r.tradesByID {
		if trade.Symbol == symbol {
			result = append(result, *trade)
		}
	}
	return result, nil
}

var (
	ErrPortfolioNotFound        = &RepositoryError{"portfolio not found"}
	ErrPortfolioAlreadyExists   = &RepositoryError{"portfolio already exists"}
	ErrTradeResultNotFound      = &RepositoryError{"trade result not found"}
	ErrTradeResultAlreadyExists = &RepositoryError{"trade result already exists"}
)

type RepositoryError struct {
	Message string
}

func (e *RepositoryError) Error() string {
	return e.Message
}
