package exchange

import (
	"context"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// Adapter defines exchange API capabilities consumed by market service and execution service.
type Adapter interface {
	// Market data methods
	GetTicker(ctx context.Context, symbol string) (domain.MarketPrice, error)
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error)
	GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error)

	// Execution methods
	SubmitOrder(ctx context.Context, req *OrderRequest) (string, error)
	GetOrderStatus(ctx context.Context, symbol, orderID string) (*OrderStatus, error)
}

// OrderRequest represents a request to submit an order to the exchange
type OrderRequest struct {
	ClientOrderID string
	Symbol        string
	Side          string
	Quantity      float64
	Type          string
	Price         float64
	TimeInForce   string
}

// OrderStatus represents the status of an order returned by the exchange
type OrderStatus struct {
	OrderID     string
	Symbol      string
	Status      string
	Quantity    float64
	ExecutedQty float64
	Fills       []Fill
}

// Fill represents a single trade fill from the exchange
type Fill struct {
	TradeID  string
	Quantity float64
	Price    float64
	Fee      float64
	FeeAsset string
	Time     int64
}
