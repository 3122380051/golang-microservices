package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/3122380051/golang-microservices/internal/application/risk"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
)

func TestRiskEvaluator_PositionSizeCheck(t *testing.T) {
	logger := infrastructure.NewLogger(&infrastructure.Config{})
	defer logger.Sync()
	evaluator := risk.NewEvaluator(logger)

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

	signal := &domain.StrategySignal{
		ID:         "signal-1",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   0.5,
		Confidence: 66100.0, // Price estimation
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 50000.0
	portfolio.AvailableMargin = 25000.0

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 66100.0)

	assert.True(t, decision.Checks.PositionSizeCheck.Passed)
	assert.True(t, decision.IsApproved)
}

func TestRiskEvaluator_PositionSizeExceeded(t *testing.T) {
	logger := infrastructure.NewLogger(&infrastructure.Config{})
	defer logger.Sync()
	evaluator := risk.NewEvaluator(logger)

	policy := &domain.RiskPolicy{
		MaxPositionSize: 50000.0, // Small limit
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	signal := &domain.StrategySignal{
		ID:         "signal-2",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   1.0, // 1 BTC at $66k = $66k > limit
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 100000.0
	portfolio.AvailableMargin = 50000.0

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 66000.0)

	assert.False(t, decision.Checks.PositionSizeCheck.Passed)
	assert.False(t, decision.IsApproved)
	assert.Contains(t, decision.RejectionReason, "exceeds max")
}

func TestRiskEvaluator_LeverageCheck(t *testing.T) {
	logger := infrastructure.NewLogger(&infrastructure.Config{})
	defer logger.Sync()
	evaluator := risk.NewEvaluator(logger)

	policy := &domain.RiskPolicy{
		MaxPositionSize: 200000.0,
		MaxLeverage:     2.0, // Low leverage for this test
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     100000.0, // 100k exposure limit
		IsActive:        true,
	}

	signal := &domain.StrategySignal{
		ID:         "signal-3",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   2.0, // 2 BTC at 50k = 100k
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 50000.0      // Account equity
	portfolio.TotalExposure = 50000.0     // Already at 1x leverage
	portfolio.AvailableMargin = 25000.0

	// New exposure: 50k + 100k = 150k / 50k equity = 3x leverage (exceeds 2x limit)
	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 50000.0)

	assert.False(t, decision.Checks.LeverageCheck.Passed)
	assert.False(t, decision.IsApproved)
}

func TestRiskEvaluator_MarginCheck(t *testing.T) {
	logger := infrastructure.NewLogger(&infrastructure.Config{})
	defer logger.Sync()
	evaluator := risk.NewEvaluator(logger)

	policy := &domain.RiskPolicy{
		MaxPositionSize: 100000.0,
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5, // Must keep 50% available
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	signal := &domain.StrategySignal{
		ID:         "signal-4",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   0.5,
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 100000.0
	portfolio.AvailableMargin = 10000.0   // Only 10% available (below 50% minimum)

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 50000.0)

	assert.False(t, decision.Checks.MarginCheck.Passed)
	assert.False(t, decision.IsApproved)
}

func TestRiskEvaluator_DailyLossCheck(t *testing.T) {
	logger := infrastructure.NewLogger(&infrastructure.Config{})
	defer logger.Sync()
	evaluator := risk.NewEvaluator(logger)

	policy := &domain.RiskPolicy{
		MaxPositionSize: 100000.0,
		MaxLeverage:     5.0,
		MaxDailyLoss:    -500.0,           // Max loss: -$500
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}

	signal := &domain.StrategySignal{
		ID:         "signal-5",
		UserID:     "user1",
		StrategyID: "ema-cross",
		Symbol:     "BTCUSDT",
		Side:       "BUY",
		Quantity:   0.1,
	}

	portfolio := domain.NewPortfolioSnapshot("user1")
	portfolio.TotalBalance = 100000.0
	portfolio.AvailableMargin = 50000.0
	portfolio.RealizedPnL = -600.0         // Already lost $600 > limit

	decision := evaluator.EvaluateSignal(policy, portfolio, signal, 50000.0)

	assert.False(t, decision.Checks.DailyLossCheck.Passed)
	assert.False(t, decision.IsApproved)
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
	assert.Equal(t, policy.UserID, retrieved.UserID)

	// Update
	policy.MaxLeverage = 3.0
	err = repo.Update(ctx, policy)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, policy.ID)
	require.NoError(t, err)
	assert.Equal(t, 3.0, updated.MaxLeverage)

	// List by user
	policies, err := repo.ListByUser(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 1, len(policies))

	// Delete
	err = repo.Delete(ctx, policy.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, policy.ID)
	assert.Error(t, err)
}

func TestPortfolioCache(t *testing.T) {
	cache := infrastructure.NewPortfolioCache()

	snapshot := domain.NewPortfolioSnapshot("user1")
	snapshot.TotalBalance = 50000.0
	snapshot.AvailableMargin = 25000.0
	snapshot.RealizedPnL = 100.0

	// Update cache
	err := cache.UpdateSnapshot("user1", snapshot)
	require.NoError(t, err)

	// Retrieve
	retrieved, err := cache.GetSnapshot("user1")
	require.NoError(t, err)
	assert.Equal(t, 50000.0, retrieved.TotalBalance)
	assert.Equal(t, 100.0, retrieved.RealizedPnL)

	// Invalidate
	err = cache.InvalidateSnapshot("user1")
	require.NoError(t, err)

	// Should return default empty snapshot after invalidation
	empty, err := cache.GetSnapshot("user1")
	require.NoError(t, err)
	assert.Equal(t, 0.0, empty.TotalBalance)
}
