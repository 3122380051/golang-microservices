package infrastructure

import (
	"context"
	"sync"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// InMemoryExecutionRepository is an in-memory implementation of ExecutionRepository
type InMemoryExecutionRepository struct {
	mu           sync.RWMutex
	executions   map[string]*domain.Execution
	byExchangeID map[string]*domain.Execution
}

// NewInMemoryExecutionRepository creates a new in-memory execution repository
func NewInMemoryExecutionRepository() domain.ExecutionRepository {
	return &InMemoryExecutionRepository{
		executions:   make(map[string]*domain.Execution),
		byExchangeID: make(map[string]*domain.Execution),
	}
}

func (r *InMemoryExecutionRepository) Create(ctx context.Context, execution *domain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate client order ID (idempotency)
	for _, existing := range r.executions {
		if existing.ClientOrderID == execution.ClientOrderID {
			// Return existing (idempotent behavior)
			*execution = *existing
			return nil
		}
	}

	r.executions[execution.ID] = execution
	return nil
}

func (r *InMemoryExecutionRepository) GetByID(ctx context.Context, id string) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, exists := r.executions[id]
	if !exists {
		return nil, ErrExecutionNotFound
	}
	return execution, nil
}

func (r *InMemoryExecutionRepository) GetByClientOrderID(ctx context.Context, clientOrderID string) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, execution := range r.executions {
		if execution.ClientOrderID == clientOrderID {
			return execution, nil
		}
	}
	return nil, ErrExecutionNotFound
}

func (r *InMemoryExecutionRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, exists := r.byExchangeID[exchangeOrderID]
	if !exists {
		return nil, ErrExecutionNotFound
	}
	return execution, nil
}

func (r *InMemoryExecutionRepository) Update(ctx context.Context, execution *domain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.executions[execution.ID]; !exists {
		return ErrExecutionNotFound
	}

	// Update mappings
	if execution.ExchangeOrderID != "" {
		r.byExchangeID[execution.ExchangeOrderID] = execution
	}

	r.executions[execution.ID] = execution
	return nil
}

func (r *InMemoryExecutionRepository) ListByUser(ctx context.Context, userID string, status *domain.ExecutionStatus) ([]domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Execution
	for _, execution := range r.executions {
		if execution.UserID == userID {
			if status == nil || execution.Status == *status {
				result = append(result, *execution)
			}
		}
	}
	return result, nil
}

func (r *InMemoryExecutionRepository) ListByStatus(ctx context.Context, status domain.ExecutionStatus) ([]domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Execution
	for _, execution := range r.executions {
		if execution.Status == status {
			result = append(result, *execution)
		}
	}
	return result, nil
}

var (
	ErrExecutionNotFound = &ExecutionRepositoryError{"execution not found"}
)

type ExecutionRepositoryError struct {
	Message string
}

func (e *ExecutionRepositoryError) Error() string {
	return e.Message
}
