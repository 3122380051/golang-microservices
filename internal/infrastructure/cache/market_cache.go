package cache

import (
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
)

type cacheItem[T any] struct {
	value     T
	expiresAt time.Time
}

// MarketCache is an in-memory cache with TTL for market reads.
type MarketCache struct {
	mu           sync.RWMutex
	priceBySym   map[string]cacheItem[domain.MarketPrice]
	candlesByKey map[string]cacheItem[[]domain.Candle]
}

func NewMarketCache() *MarketCache {
	return &MarketCache{
		priceBySym:   make(map[string]cacheItem[domain.MarketPrice]),
		candlesByKey: make(map[string]cacheItem[[]domain.Candle]),
	}
}

func (c *MarketCache) GetPrice(symbol string) (domain.MarketPrice, bool) {
	c.mu.RLock()
	item, ok := c.priceBySym[symbol]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.priceBySym, symbol)
			c.mu.Unlock()
		}
		return domain.MarketPrice{}, false
	}
	return item.value, true
}

func (c *MarketCache) SetPrice(symbol string, value domain.MarketPrice, ttl time.Duration) {
	c.mu.Lock()
	c.priceBySym[symbol] = cacheItem[domain.MarketPrice]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

func (c *MarketCache) GetCandles(key string) ([]domain.Candle, bool) {
	c.mu.RLock()
	item, ok := c.candlesByKey[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.candlesByKey, key)
			c.mu.Unlock()
		}
		return nil, false
	}
	return item.value, true
}

func (c *MarketCache) SetCandles(key string, value []domain.Candle, ttl time.Duration) {
	c.mu.Lock()
	c.candlesByKey[key] = cacheItem[[]domain.Candle]{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}
