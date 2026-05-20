package tests

import (
	"context"
	"testing"

	"github.com/3122380051/golang-microservices/internal/application/portfolio"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortfolio_AddPosition(t *testing.T) {
	p := domain.NewPortfolio("user1", 10000)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 1.0, 50000)

	err := p.AddPosition(position)
	require.NoError(t, err)

	assert.Equal(t, 1, len(p.Positions))
	assert.Equal(t, position, p.Positions["BTCUSDT"])
}

func TestPortfolio_UpdatePrice(t *testing.T) {
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 1.0, 50000)

	// Price goes up
	position.UpdatePrice(55000)
	assert.Equal(t, 55000.0, position.CurrentPrice)
	assert.Equal(t, 5000.0, position.UnrealizedPnL) // (55000 - 50000) * 1
}

func TestPortfolio_UpdatePrice_Short(t *testing.T) {
	position := domain.NewPosition("BTCUSDT", domain.OrderSideSell, 1.0, 50000)

	// Price goes down = profit for short
	position.UpdatePrice(45000)
	assert.Equal(t, 45000.0, position.CurrentPrice)
	assert.Equal(t, 5000.0, position.UnrealizedPnL) // (50000 - 45000) * 1
}

func TestPortfolio_RealizePnL(t *testing.T) {
	p := domain.NewPortfolio("user1", 10000)
	initialBalance := p.TotalBalance

	err := p.RealizePnL(500.0, 10.0)
	require.NoError(t, err)

	assert.Equal(t, 490.0, p.RealizedPnL)
	assert.Equal(t, initialBalance+490.0, p.TotalBalance)
	assert.Equal(t, 1, p.TotalTrades)
}

func TestPortfolio_IsHealthy(t *testing.T) {
	p := domain.NewPortfolio("user1", 10000)
	p.UpdateMargin(3000)

	assert.True(t, p.IsHealthy()) // 70% available

	p.UpdateMargin(8000)
	assert.False(t, p.IsHealthy()) // 20% available (below 50% threshold)
}

func TestPortfolio_IsForceLiquidation(t *testing.T) {
	p := domain.NewPortfolio("user1", 10000)

	// 70% available - healthy
	p.UpdateMargin(3000)
	assert.False(t, p.IsForceLiquidation())

	// 15% available - below 25% threshold
	p.UpdateMargin(8500)
	assert.True(t, p.IsForceLiquidation())
}

func TestPnLCalculator_CalculateClosedPnL_Long(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 2.0, 50000)

	pnl := calc.CalculateClosedPnL(position, 55000)
	assert.Equal(t, 10000.0, pnl) // (55000 - 50000) * 2
}

func TestPnLCalculator_CalculateClosedPnL_Short(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideSell, 1.0, 50000)

	pnl := calc.CalculateClosedPnL(position, 45000)
	assert.Equal(t, 5000.0, pnl) // (50000 - 45000) * 1
}

func TestPnLCalculator_CalculateUnrealizedPnL(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 1.0, 50000)

	pnl := calc.CalculateUnrealizedPnL(position, 52000)
	assert.Equal(t, 2000.0, pnl) // (52000 - 50000) * 1
}

func TestPnLCalculator_CalculateROI(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)

	roi := calc.CalculateROI(1000, 10000)
	assert.Equal(t, 10.0, roi) // 1000 / 10000 * 100
}

func TestPnLCalculator_CalculateBreakeven_Long(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 1.0, 50000)

	breakeven := calc.CalculateBreakeven(position, 100.0)
	assert.Equal(t, 50100.0, breakeven) // entry + (fees / qty)
}

func TestPnLCalculator_CalculateBreakeven_Short(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideSell, 1.0, 50000)

	breakeven := calc.CalculateBreakeven(position, 100.0)
	assert.Equal(t, 49900.0, breakeven) // entry - (fees / qty)
}

func TestPortfolioRepository_Create(t *testing.T) {
	repo := infrastructure.NewInMemoryPortfolioRepository()
	ctx := context.Background()

	p := domain.NewPortfolio("user1", 10000)
	err := repo.Create(ctx, p)
	require.NoError(t, err)

	retrieved, err := repo.GetByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, "user1", retrieved.UserID)
	assert.Equal(t, 10000.0, retrieved.TotalBalance)
}

func TestPortfolioRepository_Update(t *testing.T) {
	repo := infrastructure.NewInMemoryPortfolioRepository()
	ctx := context.Background()

	p := domain.NewPortfolio("user1", 10000)
	repo.Create(ctx, p)

	p.RealizedPnL = 500
	err := repo.Update(ctx, p)
	require.NoError(t, err)

	retrieved, _ := repo.GetByUserID(ctx, "user1")
	assert.Equal(t, 500.0, retrieved.RealizedPnL)
}

func TestPortfolioRepository_ListAll(t *testing.T) {
	repo := infrastructure.NewInMemoryPortfolioRepository()
	ctx := context.Background()

	repo.Create(ctx, domain.NewPortfolio("user1", 10000))
	repo.Create(ctx, domain.NewPortfolio("user2", 20000))

	portfolios, err := repo.ListAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(portfolios))
}

func TestPortfolioRepository_Delete(t *testing.T) {
	repo := infrastructure.NewInMemoryPortfolioRepository()
	ctx := context.Background()

	repo.Create(ctx, domain.NewPortfolio("user1", 10000))
	err := repo.Delete(ctx, "user1")
	require.NoError(t, err)

	_, err = repo.GetByUserID(ctx, "user1")
	assert.Error(t, err)
}

func TestTradeResultRepository_Create(t *testing.T) {
	repo := infrastructure.NewInMemoryTradeResultRepository()
	ctx := context.Background()

	trade := &domain.TradeResult{
		ID:          "trade-1",
		ExecutionID: "exec-1",
		Symbol:      "BTCUSDT",
		Side:        domain.OrderSideBuy,
		EntryPrice:  50000,
		ExitPrice:   55000,
		Quantity:    1.0,
		RealizedPnL: 5000,
		Fees:        10,
		NetPnL:      4990,
		IsWin:       true,
	}

	err := repo.Create(ctx, trade)
	require.NoError(t, err)

	retrieved, _ := repo.GetByID(ctx, "trade-1")
	assert.Equal(t, 5000.0, retrieved.RealizedPnL)
}

func TestTradeResultRepository_ListBySymbol(t *testing.T) {
	repo := infrastructure.NewInMemoryTradeResultRepository()
	ctx := context.Background()

	repo.Create(ctx, &domain.TradeResult{ID: "trade-1", Symbol: "BTCUSDT"})
	repo.Create(ctx, &domain.TradeResult{ID: "trade-2", Symbol: "ETHUSDT"})
	repo.Create(ctx, &domain.TradeResult{ID: "trade-3", Symbol: "BTCUSDT"})

	trades, _ := repo.ListBySymbol(ctx, "BTCUSDT")
	assert.Equal(t, 2, len(trades))
}

func TestPortfolio_CalculateLeverage(t *testing.T) {
	p := domain.NewPortfolio("user1", 10000)
	p.UpdateMargin(5000)

	// Leverage = (TotalBalance + UsedMargin) / TotalBalance
	// = (10000 + 5000) / 10000 = 1.5x
	leverage := p.CalculateLeverage()
	assert.Equal(t, 1.5, leverage)
}

func TestPnLCalculator_CalculateMaxDrawdown(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)

	balances := []float64{10000, 12000, 11000, 15000, 10000, 9000}
	maxDD := calc.CalculateMaxDrawdown(balances)

	// From peak 15000 to low 9000 = 6000/15000 = 0.4 (40%)
	assert.Greater(t, maxDD, 0.39)
	assert.Less(t, maxDD, 0.41)
}

func TestPnLCalculator_CalculatePositionMetrics(t *testing.T) {
	calc := portfolio.NewPnLCalculator(0.1)
	position := domain.NewPosition("BTCUSDT", domain.OrderSideBuy, 1.0, 50000)

	metrics := calc.CalculatePositionMetrics(position, 55000)

	assert.Equal(t, "BTCUSDT", metrics.Symbol)
	assert.Equal(t, 1.0, metrics.Quantity)
	assert.Equal(t, 50000.0, metrics.AverageEntryPrice)
	assert.Equal(t, 55000.0, metrics.CurrentPrice)
	assert.Equal(t, 5000.0, metrics.UnrealizedPnL)
	assert.Equal(t, 10.0, metrics.UnrealizedReturn) // 5000 / 50000 * 100
}

// MockPriceFetcher for testing
type MockPriceFetcher struct{}

func (m *MockPriceFetcher) GetPrice(ctx context.Context, symbol string) (float64, error) {
	return 50000.0, nil
}

func (m *MockPriceFetcher) GetPrices(ctx context.Context, symbols []string) (map[string]float64, error) {
	result := make(map[string]float64)
	for _, sym := range symbols {
		result[sym] = 50000.0
	}
	return result, nil
}
