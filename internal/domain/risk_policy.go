package domain

import (
	"context"
	"time"
)

// RiskPolicy defines risk constraints for a user/strategy combination
type RiskPolicy struct {
	ID              string    `json:"id" db:"id"`
	UserID          string    `json:"user_id" db:"user_id"`
	StrategyID      string    `json:"strategy_id" db:"strategy_id"`
	MaxPositionSize float64   `json:"max_position_size" db:"max_position_size"` // USD
	MaxLeverage     float64   `json:"max_leverage" db:"max_leverage"`           // e.g., 5.0
	MaxDailyLoss    float64   `json:"max_daily_loss" db:"max_daily_loss"`       // USD, negative (e.g., -1000)
	MinMarginRatio  float64   `json:"min_margin_ratio" db:"min_margin_ratio"`   // e.g., 0.5 (50%)
	MaxExposure     float64   `json:"max_exposure" db:"max_exposure"`           // Total USD exposure
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// RiskPolicyRepository defines persistence interface for risk policies
type RiskPolicyRepository interface {
	Create(ctx context.Context, policy *RiskPolicy) error
	GetByID(ctx context.Context, id string) (*RiskPolicy, error)
	GetByUserAndStrategy(ctx context.Context, userID, strategyID string) (*RiskPolicy, error)
	GetDefaultPolicy(ctx context.Context) (*RiskPolicy, error)
	Update(ctx context.Context, policy *RiskPolicy) error
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]RiskPolicy, error)
}
