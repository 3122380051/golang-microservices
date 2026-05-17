package domain

import "time"

// Candle is a normalized OHLCV data point.
type Candle struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Interval  string    `json:"interval"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
}
