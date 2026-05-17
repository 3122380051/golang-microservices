package domain

import "time"

// MarketPrice is normalized top-of-book snapshot for a symbol.
type MarketPrice struct {
	Symbol   string    `json:"symbol"`
	Exchange string    `json:"exchange"`
	Price    float64   `json:"price"`
	Bid      float64   `json:"bid"`
	Ask      float64   `json:"ask"`
	Ts       time.Time `json:"ts"`
}

// OrderLevel represents one bid/ask level.
type OrderLevel struct {
	Price float64 `json:"price"`
	Qty   float64 `json:"qty"`
}

// OrderBook is normalized market depth payload.
type OrderBook struct {
	Symbol   string       `json:"symbol"`
	Exchange string       `json:"exchange"`
	Bids     []OrderLevel `json:"bids"`
	Asks     []OrderLevel `json:"asks"`
	Ts       time.Time    `json:"ts"`
}
