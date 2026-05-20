package infrastructure

import (
	"context"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// InMemoryRiskPolicyRepository is an in-memory implementation of RiskPolicyRepository
type InMemoryRiskPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*domain.RiskPolicy
}

// NewInMemoryRiskPolicyRepository creates a new in-memory risk policy repository
func NewInMemoryRiskPolicyRepository() domain.RiskPolicyRepository {
	return &InMemoryRiskPolicyRepository{
		policies: make(map[string]*domain.RiskPolicy),
	}
}

func (r *InMemoryRiskPolicyRepository) Create(ctx context.Context, policy *domain.RiskPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	r.policies[policy.ID] = policy
	return nil
}

func (r *InMemoryRiskPolicyRepository) GetByID(ctx context.Context, id string) (*domain.RiskPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policy, exists := r.policies[id]
	if !exists {
		return nil, ErrPolicyNotFound
	}
	return policy, nil
}

func (r *InMemoryRiskPolicyRepository) GetByUserAndStrategy(ctx context.Context, userID, strategyID string) (*domain.RiskPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, policy := range r.policies {
		if policy.UserID == userID && policy.StrategyID == strategyID && policy.IsActive {
			return policy, nil
		}
	}
	return nil, ErrPolicyNotFound
}

func (r *InMemoryRiskPolicyRepository) GetDefaultPolicy(ctx context.Context) (*domain.RiskPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, policy := range r.policies {
		if policy.UserID == "default" && policy.StrategyID == "*" && policy.IsActive {
			return policy, nil
		}
	}
	return nil, ErrDefaultPolicyNotFound
}

func (r *InMemoryRiskPolicyRepository) Update(ctx context.Context, policy *domain.RiskPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.policies[policy.ID]; !exists {
		return ErrPolicyNotFound
	}

	policy.UpdatedAt = time.Now()
	r.policies[policy.ID] = policy
	return nil
}

func (r *InMemoryRiskPolicyRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.policies, id)
	return nil
}

func (r *InMemoryRiskPolicyRepository) ListByUser(ctx context.Context, userID string) ([]domain.RiskPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.RiskPolicy
	for _, policy := range r.policies {
		if policy.UserID == userID {
			result = append(result, *policy)
		}
	}
	return result, nil
}

// PortfolioCache implements PortfolioCacheProvider for MVP
type PortfolioCache struct {
	mu        sync.RWMutex
	snapshots map[string]*domain.PortfolioSnapshot
}

// NewPortfolioCache creates a new portfolio cache
func NewPortfolioCache() domain.PortfolioCacheProvider {
	return &PortfolioCache{
		snapshots: make(map[string]*domain.PortfolioSnapshot),
	}
}

func (pc *PortfolioCache) GetSnapshot(userID string) (*domain.PortfolioSnapshot, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	snapshot, exists := pc.snapshots[userID]
	if !exists {
		// Return default snapshot if not found
		return domain.NewPortfolioSnapshot(userID), nil
	}
	return snapshot, nil
}

func (pc *PortfolioCache) UpdateSnapshot(userID string, snapshot *domain.PortfolioSnapshot) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	snapshot.LastUpdatedAt = time.Now().Unix()
	pc.snapshots[userID] = snapshot
	return nil
}

func (pc *PortfolioCache) InvalidateSnapshot(userID string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	delete(pc.snapshots, userID)
	return nil
}

var (
	ErrPolicyNotFound           = &RepositoryError{"policy not found"}
	ErrDefaultPolicyNotFound    = &RepositoryError{"default policy not found"}
)

