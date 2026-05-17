package exchange

import (
	"context"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// Adapter defines exchange API capabilities consumed by market service.
type Adapter interface {
	GetTicker(ctx context.Context, symbol string) (domain.MarketPrice, error)
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error)
	GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error)
}
