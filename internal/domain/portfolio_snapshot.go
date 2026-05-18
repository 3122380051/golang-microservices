package domain

// PortfolioSnapshot represents current portfolio state for risk evaluation
// This is read from cache or Portfolio Service
type PortfolioSnapshot struct {
	UserID            string                     `json:"user_id"`
	TotalBalance      float64                    `json:"total_balance"`       // USD
	AvailableMargin   float64                    `json:"available_margin"`    // USD
	UsedMargin        float64                    `json:"used_margin"`         // USD
	UnrealizedPnL     float64                    `json:"unrealized_pnl"`      // USD
	RealizedPnL       float64                    `json:"realized_pnl"`        // USD (today's)
	MarginRatio       float64                    `json:"margin_ratio"`        // available / total
	TotalExposure     float64                    `json:"total_exposure"`      // Sum of all position values
	OpenPositions     map[string]*PositionState `json:"open_positions"`      // Symbol -> PositionState
	LastUpdatedAt     int64                      `json:"last_updated_at"`     // Unix timestamp
}

// PositionState represents a single open position
type PositionState struct {
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	EntryPrice    float64 `json:"entry_price"`
	MarkPrice     float64 `json:"mark_price"`
	PositionValue float64 `json:"position_value"` // quantity * mark_price
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	Leverage      float64 `json:"leverage"`       // position_value / account_equity
}

// PortfolioCacheProvider defines interface for retrieving portfolio snapshots
type PortfolioCacheProvider interface {
	GetSnapshot(userID string) (*PortfolioSnapshot, error)
	UpdateSnapshot(userID string, snapshot *PortfolioSnapshot) error
	InvalidateSnapshot(userID string) error
}

// NewPortfolioSnapshot creates a new portfolio snapshot for a user
func NewPortfolioSnapshot(userID string) *PortfolioSnapshot {
	return &PortfolioSnapshot{
		UserID:        userID,
		OpenPositions: make(map[string]*PositionState),
	}
}
