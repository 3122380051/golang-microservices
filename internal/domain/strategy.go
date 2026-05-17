package domain

import (
	"encoding/json"
	"time"
)

// StrategyType enumerates supported algorithm types.
type StrategyType string

const (
	StrategyTypeEMACross StrategyType = "ema_cross"
	StrategyTypeRSI      StrategyType = "rsi"
)

// Strategy represents a persisted trading strategy definition.
type Strategy struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Symbol    string          `json:"symbol"`
	Type      StrategyType    `json:"type"`
	Active    bool            `json:"active"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// EMACrossConfig controls the moving average crossover strategy.
type EMACrossConfig struct {
	Fast   int `json:"fast"`
	Slow   int `json:"slow"`
	Signal int `json:"signal"`
}

// RSIConfig controls the RSI strategy.
type RSIConfig struct {
	Period     int     `json:"period"`
	Overbought float64 `json:"overbought"`
	Oversold   float64 `json:"oversold"`
}
