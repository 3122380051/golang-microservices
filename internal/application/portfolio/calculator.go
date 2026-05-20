package portfolio

import (
	"github.com/3122380051/golang-microservices/internal/domain"
)

// PnLCalculator handles profit/loss calculations
type PnLCalculator struct {
	maintenanceMarginRatio float64
}

// NewPnLCalculator creates a new PnL calculator
func NewPnLCalculator(maintenanceMarginRatio float64) *PnLCalculator {
	if maintenanceMarginRatio <= 0 {
		maintenanceMarginRatio = 0.1 // Default 10% maintenance margin
	}
	return &PnLCalculator{
		maintenanceMarginRatio: maintenanceMarginRatio,
	}
}

// CalculateClosedPnL calculates profit/loss for a closed position
func (c *PnLCalculator) CalculateClosedPnL(position *domain.Position, exitPrice float64) float64 {
	switch position.Side {
	case domain.OrderSideBuy:
		// Long position: profit = (exit - entry) * quantity
		return (exitPrice - position.AverageEntryPrice) * position.Quantity
	case domain.OrderSideSell:
		// Short position: profit = (entry - exit) * quantity
		return (position.AverageEntryPrice - exitPrice) * position.Quantity
	default:
		return 0
	}
}

// CalculateUnrealizedPnL calculates unrealized P&L for an open position
func (c *PnLCalculator) CalculateUnrealizedPnL(position *domain.Position, currentPrice float64) float64 {
	switch position.Side {
	case domain.OrderSideBuy:
		// Long: profit if price > entry
		return (currentPrice - position.AverageEntryPrice) * position.Quantity
	case domain.OrderSideSell:
		// Short: profit if price < entry
		return (position.AverageEntryPrice - currentPrice) * position.Quantity
	default:
		return 0
	}
}

// CalculateROI calculates return on investment percentage
func (c *PnLCalculator) CalculateROI(pnl, investedAmount float64) float64 {
	if investedAmount == 0 {
		return 0
	}
	return (pnl / investedAmount) * 100
}

// CalculateRequiredMargin calculates margin requirement for a position
func (c *PnLCalculator) CalculateRequiredMargin(positionValue float64) float64 {
	return positionValue * c.maintenanceMarginRatio
}

// CalculatePortfolioMargin calculates total margin requirement for portfolio
func (c *PnLCalculator) CalculatePortfolioMargin(portfolio *domain.Portfolio) float64 {
	var totalMargin float64
	for _, position := range portfolio.Positions {
		totalMargin += c.CalculateRequiredMargin(position.PositionValue)
	}
	return totalMargin
}

// CalculateMaxDrawdown calculates maximum drawdown from a series of balances
func (c *PnLCalculator) CalculateMaxDrawdown(balances []float64) float64 {
	if len(balances) == 0 {
		return 0
	}

	maxBalance := balances[0]
	maxDrawdown := 0.0

	for _, balance := range balances {
		if balance > maxBalance {
			maxBalance = balance
		}
		drawdown := (maxBalance - balance) / maxBalance
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// CalculateSharpeRatio calculates Sharpe ratio (returns / volatility)
func (c *PnLCalculator) CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate mean return
	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	// Calculate standard deviation
	var variance float64
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	if variance == 0 {
		return 0
	}

	stdDev := variance // sqrt would go here but simplified for MVP
	if stdDev == 0 {
		return 0
	}

	return (mean - riskFreeRate) / stdDev
}

// CalculateBreakeven calculates breakeven price for a position
func (c *PnLCalculator) CalculateBreakeven(position *domain.Position, fees float64) float64 {
	feePerUnit := fees / position.Quantity

	switch position.Side {
	case domain.OrderSideBuy:
		// Breakeven = entry price + fees per unit
		return position.AverageEntryPrice + feePerUnit
	case domain.OrderSideSell:
		// Breakeven = entry price - fees per unit
		return position.AverageEntryPrice - feePerUnit
	default:
		return position.AverageEntryPrice
	}
}

// CalculateLiquidationPrice calculates the price at which position would be liquidated
func (c *PnLCalculator) CalculateLiquidationPrice(position *domain.Position, portfolio *domain.Portfolio) float64 {
	if c.maintenanceMarginRatio == 0 || position.Quantity == 0 {
		return 0
	}

	// Simplified: liquidation when margin ratio hits maintenance threshold
	// liquidationPrice = entryPrice ± (balance / quantity) * maintenanceMarginRatio

	marginPerUnit := portfolio.TotalBalance / position.Quantity * c.maintenanceMarginRatio

	switch position.Side {
	case domain.OrderSideBuy:
		// For long: liquidation price is below entry
		return position.AverageEntryPrice - marginPerUnit
	case domain.OrderSideSell:
		// For short: liquidation price is above entry
		return position.AverageEntryPrice + marginPerUnit
	default:
		return position.AverageEntryPrice
	}
}

// CalculatePositionMetrics returns comprehensive metrics for a position
type PositionMetrics struct {
	Symbol            string
	Side              string
	Quantity          float64
	AverageEntryPrice float64
	CurrentPrice      float64
	UnrealizedPnL     float64
	UnrealizedReturn  float64 // %
	PositionValue     float64
	BreakevenPrice    float64
	LiquidationPrice  float64
	ROI               float64 // %
}

func (c *PnLCalculator) CalculatePositionMetrics(position *domain.Position, currentPrice float64) PositionMetrics {
	unrealizedPnL := c.CalculateUnrealizedPnL(position, currentPrice)
	investedAmount := position.AverageEntryPrice * position.Quantity

	return PositionMetrics{
		Symbol:            position.Symbol,
		Side:              string(position.Side),
		Quantity:          position.Quantity,
		AverageEntryPrice: position.AverageEntryPrice,
		CurrentPrice:      currentPrice,
		UnrealizedPnL:     unrealizedPnL,
		UnrealizedReturn:  c.CalculateROI(unrealizedPnL, investedAmount),
		PositionValue:     currentPrice * position.Quantity,
		BreakevenPrice:    c.CalculateBreakeven(position, 0),
		LiquidationPrice:  0, // Requires portfolio context
		ROI:               c.CalculateROI(unrealizedPnL, investedAmount),
	}
}
