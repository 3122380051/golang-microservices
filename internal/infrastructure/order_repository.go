package infrastructure

import (
	"context"
	"sync"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// InMemoryOrderRepository is an in-memory implementation of OrderRepository
type InMemoryOrderRepository struct {
	mu              sync.RWMutex
	ordersById      map[string]*domain.Order
	ordersByClientID map[string]*domain.Order
}

// NewInMemoryOrderRepository creates a new in-memory order repository
func NewInMemoryOrderRepository() domain.OrderRepository {
	return &InMemoryOrderRepository{
		ordersById:       make(map[string]*domain.Order),
		ordersByClientID: make(map[string]*domain.Order),
	}
}

func (r *InMemoryOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate client order ID (idempotency)
	if existing, exists := r.ordersByClientID[order.ClientOrderID]; exists {
		// Return existing order (idempotent behavior)
		*order = *existing
		return nil
	}

	r.ordersById[order.ID] = order
	r.ordersByClientID[order.ClientOrderID] = order
	return nil
}

func (r *InMemoryOrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.ordersById[id]
	if !exists {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (r *InMemoryOrderRepository) GetByClientOrderID(ctx context.Context, clientOrderID string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.ordersByClientID[clientOrderID]
	if !exists {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

func (r *InMemoryOrderRepository) Update(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.ordersById[order.ID]; !exists {
		return ErrOrderNotFound
	}

	r.ordersById[order.ID] = order
	r.ordersByClientID[order.ClientOrderID] = order
	return nil
}

func (r *InMemoryOrderRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, exists := r.ordersById[id]
	if !exists {
		return ErrOrderNotFound
	}

	delete(r.ordersById, id)
	delete(r.ordersByClientID, order.ClientOrderID)
	return nil
}

func (r *InMemoryOrderRepository) ListByUser(ctx context.Context, userID string, status *domain.OrderStatus) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Order
	for _, order := range r.ordersById {
		if order.UserID == userID {
			if status == nil || order.Status == *status {
				result = append(result, *order)
			}
		}
	}
	return result, nil
}

func (r *InMemoryOrderRepository) ListByStrategy(ctx context.Context, strategyID string, status *domain.OrderStatus) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Order
	for _, order := range r.ordersById {
		if order.StrategyID == strategyID {
			if status == nil || order.Status == *status {
				result = append(result, *order)
			}
		}
	}
	return result, nil
}

func (r *InMemoryOrderRepository) ListByStatus(ctx context.Context, status domain.OrderStatus) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Order
	for _, order := range r.ordersById {
		if order.Status == status {
			result = append(result, *order)
		}
	}
	return result, nil
}

var (
	ErrOrderNotFound = &OrderRepositoryError{"order not found"}
)

type OrderRepositoryError struct {
	Message string
}

func (e *OrderRepositoryError) Error() string {
	return e.Message
}
