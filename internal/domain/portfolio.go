package domain

import (
	"context"
	"fmt"
	"time"
)

// PortfolioStatus represents the state of a portfolio
type PortfolioStatus string

const (
	PortfolioStatusActive      PortfolioStatus = "active"
	PortfolioStatusClosed      PortfolioStatus = "closed"
	PortfolioStatusLiquidating PortfolioStatus = "liquidating"
)

// Position represents a holding in a specific symbol
type Position struct {
	Symbol            string
	Side              OrderSide // BUY or SELL (long/short)
	Quantity          float64
	AverageEntryPrice float64
	CurrentPrice      float64
	UnrealizedPnL     float64
	RealizedPnL       float64
	PositionValue     float64
	UpdatedAt         time.Time
}

// PortfolioSnapshot represents the state of a user's portfolio at a point in time
type Portfolio struct {
	ID                string
	UserID            string
	Status            PortfolioStatus
	TotalBalance      float64              // Cash + positions value
	AvailableMargin   float64              // Cash available for trading
	UsedMargin        float64              // Margin locked in positions
	MaintenanceMargin float64              // Minimum margin required
	Positions         map[string]*Position // symbol -> Position
	RealizedPnL       float64              // Cumulative realized PnL
	UnrealizedPnL     float64              // Sum of all unrealized PnL
	TotalPnL          float64              // Realized + Unrealized
	MarginRatio       float64              // AvailableMargin / TotalBalance
	TotalTrades       int
	WinRate           float64 // Winning trades / total trades
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TradeResult represents the outcome of a closed position
type TradeResult struct {
	ID              string
	ExecutionID     string
	Symbol          string
	Side            OrderSide
	EntryPrice      float64
	ExitPrice       float64
	Quantity        float64
	RealizedPnL     float64
	RealizedReturn  float64 // PnL / (Entry * Quantity) in %
	Fees            float64
	NetPnL          float64 // RealizedPnL - Fees
	EntryTime       time.Time
	ExitTime        time.Time
	DurationSeconds int64
	IsWin           bool // NetPnL > 0
}

// PnLEvent is published when PnL is realized
type PnLEvent struct {
	EventID           string
	UserID            string
	ExecutionID       string
	CorrelationID     string
	Symbol            string
	Side              OrderSide
	Quantity          float64
	EntryPrice        float64
	ExitPrice         float64
	RealizedPnL       float64
	Fees              float64
	NetPnL            float64
	TotalPortfolioPnL float64
	Timestamp         time.Time
	EventType         string // "portfolio.position_closed", "portfolio.pnl_updated"
	TraceID           string
}

// PortfolioRepository defines portfolio persistence operations
type PortfolioRepository interface {
	Create(ctx context.Context, portfolio *Portfolio) error
	GetByUserID(ctx context.Context, userID string) (*Portfolio, error)
	Update(ctx context.Context, portfolio *Portfolio) error
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]Portfolio, error)
}

// TradeResultRepository defines trade result persistence operations
type TradeResultRepository interface {
	Create(ctx context.Context, result *TradeResult) error
	GetByID(ctx context.Context, id string) (*TradeResult, error)
	ListByUser(ctx context.Context, userID string) ([]TradeResult, error)
	ListBySymbol(ctx context.Context, symbol string) ([]TradeResult, error)
}

// Position helper methods

// NewPosition creates a new position
func NewPosition(symbol string, side OrderSide, quantity, price float64) *Position {
	return &Position{
		Symbol:            symbol,
		Side:              side,
		Quantity:          quantity,
		AverageEntryPrice: price,
		CurrentPrice:      price,
		PositionValue:     quantity * price,
		UpdatedAt:         time.Now(),
	}
}

// UpdatePrice updates the current price and recalculates unrealized PnL
func (p *Position) UpdatePrice(newPrice float64) {
	p.CurrentPrice = newPrice
	p.PositionValue = p.Quantity * newPrice
	p.UpdatedAt = time.Now()

	// Calculate unrealized PnL
	switch p.Side {
	case OrderSideBuy:
		// For long positions: profit if price goes up
		p.UnrealizedPnL = (newPrice - p.AverageEntryPrice) * p.Quantity
	case OrderSideSell:
		// For short positions: profit if price goes down
		p.UnrealizedPnL = (p.AverageEntryPrice - newPrice) * p.Quantity
	}
}

// Portfolio helper methods

// NewPortfolio creates a new portfolio for a user
func NewPortfolio(userID string, initialBalance float64) *Portfolio {
	now := time.Now()
	return &Portfolio{
		ID:              fmt.Sprintf("port-%d", now.UnixNano()),
		UserID:          userID,
		Status:          PortfolioStatusActive,
		TotalBalance:    initialBalance,
		AvailableMargin: initialBalance,
		UsedMargin:      0,
		Positions:       make(map[string]*Position),
		RealizedPnL:     0,
		UnrealizedPnL:   0,
		TotalPnL:        0,
		MarginRatio:     1.0,
		TotalTrades:     0,
		WinRate:         0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// AddPosition adds or updates a position
func (p *Portfolio) AddPosition(position *Position) error {
	if position == nil {
		return fmt.Errorf("position cannot be nil")
	}
	if position.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	p.Positions[position.Symbol] = position
	p.UpdatedAt = time.Now()
	p.RecalculateTotals()
	return nil
}

// RemovePosition removes a position
func (p *Portfolio) RemovePosition(symbol string) error {
	if _, exists := p.Positions[symbol]; !exists {
		return fmt.Errorf("position for %s not found", symbol)
	}

	delete(p.Positions, symbol)
	p.UpdatedAt = time.Now()
	p.RecalculateTotals()
	return nil
}

// UpdateMargin updates margin usage
func (p *Portfolio) UpdateMargin(used float64) error {
	if used < 0 {
		return fmt.Errorf("used margin cannot be negative")
	}
	if used > p.TotalBalance {
		return fmt.Errorf("used margin exceeds total balance")
	}

	p.UsedMargin = used
	p.AvailableMargin = p.TotalBalance - used
	if p.TotalBalance > 0 {
		p.MarginRatio = p.AvailableMargin / p.TotalBalance
	}
	p.UpdatedAt = time.Now()
	return nil
}

// RealizePnL realizes profit/loss when closing a position
func (p *Portfolio) RealizePnL(pnl, fees float64) error {
	netPnL := pnl - fees
	p.RealizedPnL += netPnL
	p.TotalPnL = p.RealizedPnL + p.UnrealizedPnL
	p.TotalBalance += netPnL
	p.TotalTrades++
	p.UpdatedAt = time.Now()

	// Update win rate
	if p.TotalTrades > 0 && netPnL > 0 {
		// Count as win
	}

	return nil
}

// RecalculateTotals recalculates portfolio totals from positions
func (p *Portfolio) RecalculateTotals() {
	var unrealizedPnL float64
	var positionValue float64

	for _, pos := range p.Positions {
		unrealizedPnL += pos.UnrealizedPnL
		positionValue += pos.PositionValue
	}

	p.UnrealizedPnL = unrealizedPnL
	p.TotalPnL = p.RealizedPnL + p.UnrealizedPnL
}

// CalculateLeverage calculates current leverage
func (p *Portfolio) CalculateLeverage() float64 {
	if p.TotalBalance == 0 {
		return 0
	}
	return (p.TotalBalance + p.UsedMargin) / p.TotalBalance
}

// IsHealthy checks if portfolio meets minimum margin requirement
func (p *Portfolio) IsHealthy() bool {
	if p.TotalBalance == 0 {
		return false
	}
	// Default minimum margin ratio: 0.5 (50%)
	return p.MarginRatio >= 0.5
}

// IsForceLiquidation checks if portfolio is below liquidation threshold
func (p *Portfolio) IsForceLiquidation() bool {
	// Default liquidation threshold: 0.25 (25% margin ratio)
	return p.MarginRatio < 0.25
}
