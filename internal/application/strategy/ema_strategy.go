package strategy

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/3122380051/golang-microservices/internal/domain"
)

type emaState struct {
	fastEMA     float64
	slowEMA     float64
	prevDiff    float64
	initialized bool
}

// EMAStrategy evaluates EMA crossover signals.
type EMAStrategy struct{}

func NewEMAStrategy() *EMAStrategy { return &EMAStrategy{} }

func (s *EMAStrategy) Type() domain.StrategyType { return domain.StrategyTypeEMACross }

func (s *EMAStrategy) Evaluate(strategy domain.Strategy, price float64, state any) (domain.Signal, any, bool, error) {
	cfg, err := decodeEMACrossConfig(strategy.Config)
	if err != nil {
		return domain.Signal{}, state, false, err
	}
	if cfg.Fast <= 0 || cfg.Slow <= 0 || cfg.Signal < 0 || cfg.Fast >= cfg.Slow {
		return domain.Signal{}, state, false, fmt.Errorf("invalid ema config")
	}

	st, _ := state.(*emaState)
	if st == nil {
		st = &emaState{}
	}

	fastAlpha := 2.0 / float64(cfg.Fast+1)
	slowAlpha := 2.0 / float64(cfg.Slow+1)

	if !st.initialized {
		st.fastEMA = price
		st.slowEMA = price
		st.prevDiff = 0
		st.initialized = true
		return noSignal(strategy, price, stateFromEMA(st), "ema initialized"), stateFromEMA(st), false, nil
	}

	st.fastEMA = fastAlpha*price + (1-fastAlpha)*st.fastEMA
	st.slowEMA = slowAlpha*price + (1-slowAlpha)*st.slowEMA
	diff := st.fastEMA - st.slowEMA

	var action domain.SignalAction = domain.SignalActionHold
	reason := "ema hold"
	confidence := 0.55
	if st.prevDiff <= 0 && diff > 0 {
		action = domain.SignalActionBuy
		reason = "fast ema crossed above slow ema"
		confidence = math.Min(0.95, 0.7+math.Abs(diff))
	}
	if st.prevDiff >= 0 && diff < 0 {
		action = domain.SignalActionSell
		reason = "fast ema crossed below slow ema"
		confidence = math.Min(0.95, 0.7+math.Abs(diff))
	}
	st.prevDiff = diff

	signal := domain.Signal{
		StrategyID: strategy.ID,
		Symbol:     strategy.Symbol,
		Action:     action,
		Confidence: confidence,
		Reason:     reason,
		Metadata: map[string]any{
			"strategy_type": string(strategy.Type),
			"fast_ema":      st.fastEMA,
			"slow_ema":      st.slowEMA,
			"price":         price,
		},
	}

	return signal, stateFromEMA(st), action != domain.SignalActionHold, nil
}

func decodeEMACrossConfig(raw []byte) (domain.EMACrossConfig, error) {
	var cfg domain.EMACrossConfig
	if len(raw) == 0 {
		cfg = domain.EMACrossConfig{Fast: 12, Slow: 26, Signal: 9}
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return domain.EMACrossConfig{}, err
	}
	if cfg.Fast == 0 {
		cfg.Fast = 12
	}
	if cfg.Slow == 0 {
		cfg.Slow = 26
	}
	if cfg.Signal == 0 {
		cfg.Signal = 9
	}
	return cfg, nil
}

func stateFromEMA(st *emaState) *emaState {
	if st == nil {
		return nil
	}
	copy := *st
	return &copy
}

func noSignal(strategy domain.Strategy, price float64, state any, reason string) domain.Signal {
	return domain.Signal{
		StrategyID: strategy.ID,
		Symbol:     strategy.Symbol,
		Action:     domain.SignalActionHold,
		Confidence: 0.5,
		Reason:     reason,
		Metadata: map[string]any{
			"strategy_type": string(strategy.Type),
			"price":         price,
		},
	}
}
