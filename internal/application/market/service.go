package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/domain/event"
)

// Adapter is exchange capability used by market service.
type Adapter interface {
	GetTicker(ctx context.Context, symbol string) (domain.MarketPrice, error)
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error)
	GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error)
}

// Cache stores market snapshots for quick reads.
type Cache interface {
	GetPrice(symbol string) (domain.MarketPrice, bool)
	SetPrice(symbol string, value domain.MarketPrice, ttl time.Duration)
	GetCandles(key string) ([]domain.Candle, bool)
	SetCandles(key string, value []domain.Candle, ttl time.Duration)
}

// Publisher supports event publication to broker.
type Publisher interface {
	PublishJSON(ctx context.Context, topic string, key string, value any) error
}

// Service coordinates market reads, caching and streaming.
type Service struct {
	adapter         Adapter
	cache           Cache
	publisher       Publisher
	priceTTL        time.Duration
	candleTTL       time.Duration
	subscribersMu   sync.RWMutex
	subscribersByID map[int]chan domain.MarketPrice
	nextSubID       int
}

func NewService(adapter Adapter, cache Cache, publisher Publisher) *Service {
	return &Service{
		adapter:         adapter,
		cache:           cache,
		publisher:       publisher,
		priceTTL:        10 * time.Second,
		candleTTL:       15 * time.Second,
		subscribersByID: make(map[int]chan domain.MarketPrice),
	}
}

func (s *Service) GetPrice(ctx context.Context, symbol string) (domain.MarketPrice, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return domain.MarketPrice{}, fmt.Errorf("symbol is required")
	}

	if item, ok := s.cache.GetPrice(symbol); ok {
		return item, nil
	}

	price, err := s.adapter.GetTicker(ctx, symbol)
	if err != nil {
		return domain.MarketPrice{}, err
	}

	s.cache.SetPrice(symbol, price, s.priceTTL)
	s.publishPriceUpdated(ctx, price)
	return price, nil
}

func (s *Service) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(interval) == "" {
		interval = "1h"
	}
	if limit <= 0 {
		limit = 100
	}

	cacheKey := fmt.Sprintf("%s:%s:%d", symbol, interval, limit)
	if items, ok := s.cache.GetCandles(cacheKey); ok {
		return items, nil
	}

	candles, err := s.adapter.GetCandles(ctx, symbol, interval, limit)
	if err != nil {
		return nil, err
	}

	s.cache.SetCandles(cacheKey, candles, s.candleTTL)
	return candles, nil
}

func (s *Service) GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return domain.OrderBook{}, fmt.Errorf("symbol is required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.adapter.GetOrderBook(ctx, symbol, limit)
}

func (s *Service) Subscribe() (<-chan domain.MarketPrice, func()) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.nextSubID++
	id := s.nextSubID
	ch := make(chan domain.MarketPrice, 64)
	s.subscribersByID[id] = ch

	unsubscribe := func() {
		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()
		existing, ok := s.subscribersByID[id]
		if !ok {
			return
		}
		delete(s.subscribersByID, id)
		close(existing)
	}

	return ch, unsubscribe
}

func (s *Service) StartPolling(ctx context.Context, symbol string, interval time.Duration) {
	symbol = normalizeSymbol(symbol)
	if symbol == "" {
		return
	}
	if interval <= 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			price, err := s.adapter.GetTicker(ctx, symbol)
			if err != nil {
				continue
			}
			s.cache.SetPrice(symbol, price, s.priceTTL)
			s.publishPriceUpdated(ctx, price)
			s.broadcast(price)
		}
	}
}

func (s *Service) broadcast(price domain.MarketPrice) {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for _, ch := range s.subscribersByID {
		select {
		case ch <- price:
		default:
		}
	}
}

func (s *Service) publishPriceUpdated(ctx context.Context, price domain.MarketPrice) {
	if s.publisher == nil {
		return
	}

	payload := event.MarketPriceUpdated{
		EventID:  fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		TraceID:  "n/a",
		Version:  event.SchemaVersionV1,
		Source:   "market-data-service",
		Symbol:   price.Symbol,
		Exchange: price.Exchange,
		Price:    price.Price,
		Bid:      price.Bid,
		Ask:      price.Ask,
		Key:      price.Symbol,
		Ts:       price.Ts,
	}
	env, err := event.MarshalEnvelope(payload.Topic(), payload.EventID, payload.TraceID, payload.Source, payload)
	if err != nil {
		return
	}
	_ = s.publisher.PublishJSON(ctx, payload.Topic(), payload.PartitionKey(), env)
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
