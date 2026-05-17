package tests

import (
	"context"
	"testing"
	"time"

	marketapp "github.com/3122380051/golang-microservices/internal/application/market"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure/cache"
)

type fakeMarketAdapter struct {
	tickerCalls int
}

func (f *fakeMarketAdapter) GetTicker(_ context.Context, symbol string) (domain.MarketPrice, error) {
	f.tickerCalls++
	return domain.MarketPrice{
		Symbol:   symbol,
		Exchange: "binance",
		Price:    100 + float64(f.tickerCalls),
		Bid:      99,
		Ask:      101,
		Ts:       time.Now().UTC(),
	}, nil
}

func (f *fakeMarketAdapter) GetCandles(_ context.Context, symbol, interval string, limit int) ([]domain.Candle, error) {
	items := make([]domain.Candle, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, domain.Candle{Symbol: symbol, Exchange: "binance", Interval: interval, Close: 100})
	}
	return items, nil
}

func (f *fakeMarketAdapter) GetOrderBook(_ context.Context, symbol string, _ int) (domain.OrderBook, error) {
	return domain.OrderBook{Symbol: symbol, Exchange: "binance"}, nil
}

type fakePublisher struct {
	count int
}

func (f *fakePublisher) PublishJSON(_ context.Context, _ string, _ string, _ any) error {
	f.count++
	return nil
}

func TestMarketServiceCachesPriceAndPublishes(t *testing.T) {
	adapter := &fakeMarketAdapter{}
	pub := &fakePublisher{}
	svc := marketapp.NewService(adapter, cache.NewMarketCache(), pub)

	ctx := context.Background()
	first, err := svc.GetPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("GetPrice first: %v", err)
	}
	second, err := svc.GetPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("GetPrice second: %v", err)
	}

	if adapter.tickerCalls != 1 {
		t.Fatalf("expected adapter ticker call once due to cache, got %d", adapter.tickerCalls)
	}
	if first.Symbol != second.Symbol {
		t.Fatalf("expected same symbol")
	}
	if pub.count < 1 {
		t.Fatalf("expected at least one publish call")
	}
}

func TestMarketServiceSubscribeGetsBroadcast(t *testing.T) {
	adapter := &fakeMarketAdapter{}
	svc := marketapp.NewService(adapter, cache.NewMarketCache(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	go svc.StartPolling(ctx, "BTCUSDT", 20*time.Millisecond)

	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected broadcast tick")
	}
}
