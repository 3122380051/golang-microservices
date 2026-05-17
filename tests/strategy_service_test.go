package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	strategyapp "github.com/3122380051/golang-microservices/internal/application/strategy"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/domain/event"
)

type memoryStrategyRepo struct {
	items map[string]domain.Strategy
	next  int
}

func newMemoryStrategyRepo() *memoryStrategyRepo {
	return &memoryStrategyRepo{items: make(map[string]domain.Strategy)}
}

func (r *memoryStrategyRepo) Create(_ context.Context, strategy domain.Strategy) (domain.Strategy, error) {
	r.next++
	strategy.ID = "s" + time.Now().Format("150405")
	strategy.CreatedAt = time.Now().UTC()
	strategy.UpdatedAt = strategy.CreatedAt
	r.items[strategy.ID] = strategy
	return strategy, nil
}
func (r *memoryStrategyRepo) GetByID(_ context.Context, id string) (domain.Strategy, error) {
	return r.items[id], nil
}
func (r *memoryStrategyRepo) List(_ context.Context) ([]domain.Strategy, error) {
	out := make([]domain.Strategy, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}
func (r *memoryStrategyRepo) ListActive(ctx context.Context) ([]domain.Strategy, error) {
	items, _ := r.List(ctx)
	out := make([]domain.Strategy, 0)
	for _, item := range items {
		if item.Active {
			out = append(out, item)
		}
	}
	return out, nil
}
func (r *memoryStrategyRepo) Activate(_ context.Context, id string) (domain.Strategy, error) {
	s := r.items[id]
	s.Active = true
	r.items[id] = s
	return s, nil
}
func (r *memoryStrategyRepo) Deactivate(_ context.Context, id string) (domain.Strategy, error) {
	s := r.items[id]
	s.Active = false
	r.items[id] = s
	return s, nil
}

type memorySignalPublisher struct{ published []domain.Signal }

func (p *memorySignalPublisher) PublishJSON(_ context.Context, _ string, _ string, value any) error {
	p.published = append(p.published, value.(domain.Signal))
	return nil
}

func TestEMAAndStrategySignalGeneration(t *testing.T) {
	repo := newMemoryStrategyRepo()
	pub := &memorySignalPublisher{}
	svc := strategyapp.NewService(repo, pub)

	created, err := svc.CreateStrategy(context.Background(), domain.Strategy{
		Name:   "ema btc",
		Symbol: "BTCUSDT",
		Type:   domain.StrategyTypeEMACross,
		Active: true,
		Config: mustJSON(domain.EMACrossConfig{Fast: 2, Slow: 3, Signal: 9}),
	})
	if err != nil {
		t.Fatalf("CreateStrategy: %v", err)
	}

	prices := []float64{100, 101, 102, 103, 104, 105}
	var signals []domain.Signal
	for i, price := range prices {
		env := event.Envelope{EventID: time.Now().Format("evt-150405") + string(rune('0'+i)), TraceID: "trace-1", Version: 1, Type: event.TopicMarketPriceUpdated, Source: "market", Timestamp: time.Now().UTC(), Payload: mustJSONRaw(event.MarketPriceUpdated{EventID: "evt", TraceID: "trace", Version: 1, Source: "market", Symbol: "BTCUSDT", Exchange: "binance", Price: price, Ts: time.Now().UTC()})}
		out, err := svc.HandleMarketPriceUpdated(context.Background(), env)
		if err != nil {
			t.Fatalf("HandleMarketPriceUpdated: %v", err)
		}
		signals = append(signals, out...)
	}
	if created.ID == "" {
		t.Fatalf("strategy id missing")
	}
	if len(pub.published) == 0 {
		t.Fatalf("expected published signal")
	}
	if len(signals) == 0 {
		t.Fatalf("expected generated signal")
	}
}

func TestStrategyActivationAndIdempotency(t *testing.T) {
	repo := newMemoryStrategyRepo()
	svc := strategyapp.NewService(repo, nil)
	created, err := svc.CreateStrategy(context.Background(), domain.Strategy{Name: "rsi", Symbol: "ETHUSDT", Type: domain.StrategyTypeRSI, Active: true})
	if err != nil {
		t.Fatalf("CreateStrategy: %v", err)
	}

	env := event.Envelope{EventID: "evt-1", TraceID: "trace-1", Version: 1, Type: event.TopicMarketPriceUpdated, Source: "market", Timestamp: time.Now().UTC(), Payload: mustJSONRaw(event.MarketPriceUpdated{EventID: "evt-1", TraceID: "trace-1", Version: 1, Source: "market", Symbol: "ETHUSDT", Exchange: "binance", Price: 90, Ts: time.Now().UTC()})}
	_, err = svc.HandleMarketPriceUpdated(context.Background(), env)
	if err != nil {
		t.Fatalf("first handle: %v", err)
	}
	second, err := svc.HandleMarketPriceUpdated(context.Background(), env)
	if err != nil {
		t.Fatalf("second handle: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("expected idempotent no-op")
	}
	if created.ID == "" {
		t.Fatalf("strategy id missing")
	}
}

func mustJSON(v any) json.RawMessage    { raw, _ := json.Marshal(v); return raw }
func mustJSONRaw(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
