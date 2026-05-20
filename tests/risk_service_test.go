package tests

import (
	"context"
	"testing"

	"github.com/3122380051/golang-microservices/internal/application/risk"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskEvaluator_PositionSizeCheck(t *testing.T) {
	appLogger := logger.New("info")
	evaluator := risk.NewEvaluator(appLogger)

	policy := &domain.RiskPolicy{
		ID:              "test-policy",
		UserID:          "user1",
		StrategyID:      "ema-cross",
		MaxPositionSize: 100000.0, // $100k
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	signal := &domain.Signal{
		ID:         "signal-1",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   0.5,
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 50000.0
	portfolio.AvailableMargin = 25000.0

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 66100.0)

	assert.True(t, decision.Checks.PositionSizeCheck.Passed)
	assert.True(t, decision.IsApproved)
}

func TestRiskEvaluator_PositionSizeExceeded(t *testing.T) {
	appLogger := logger.New("info")
	evaluator := risk.NewEvaluator(appLogger)

	policy := &domain.RiskPolicy{
		MaxPositionSize: 50000.0, // Small limit
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	signal := &domain.Signal{
		ID:         "signal-2",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   1.0,
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 100000.0
	portfolio.AvailableMargin = 50000.0

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 66000.0)

	assert.False(t, decision.Checks.PositionSizeCheck.Passed)
	assert.False(t, decision.IsApproved)
	assert.Contains(t, decision.RejectionReason, "exceeds max")
}

func TestRiskPolicyRepository_CRUD(t *testing.T) {
	repo := infrastructure.NewInMemoryRiskPolicyRepository()

	policy := &domain.RiskPolicy{
		ID:              "policy-1",
		UserID:          "user1",
		StrategyID:      "ema-cross",
		MaxPositionSize: 100000.0,
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	ctx := context.Background()

	// Create
	err := repo.Create(ctx, policy)
	require.NoError(t, err)

	// Read
	retrieved, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, retrieved.ID)

	// Update
	policy.MaxLeverage = 3.0
	err = repo.Update(ctx, policy)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)
	assert.Equal(t, 3.0, updated.MaxLeverage)

	// Delete
	err = repo.Delete(ctx, policy.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, policy.ID)
	assert.Error(t, err)
}
