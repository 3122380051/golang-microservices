package tests

import (
	"testing"

	strategyapp "github.com/3122380051/golang-microservices/internal/application/strategy"
	"github.com/3122380051/golang-microservices/internal/domain"
)

func TestEMAStrategyCross(t *testing.T) {
	engine := strategyapp.NewEMAStrategy()
	strategy := domain.Strategy{ID: "s1", Symbol: "BTCUSDT", Type: domain.StrategyTypeEMACross, Config: mustJSON(domain.EMACrossConfig{Fast: 2, Slow: 5, Signal: 9})}

	var state any
	var emitted bool
	for _, price := range []float64{100, 99, 98, 101, 103} {
		signal, nextState, shouldEmit, err := engine.Evaluate(strategy, price, state)
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		state = nextState
		if shouldEmit && signal.Action != domain.SignalActionHold {
			emitted = true
		}
	}
	if !emitted {
		t.Fatalf("expected an EMA crossover signal")
	}
}
