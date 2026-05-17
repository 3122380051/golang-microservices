package strategy

import (
	"encoding/json"
	"math"

	"github.com/3122380051/golang-microservices/internal/domain"
)

type rsiState struct {
	lastPrice float64
	avgGain   float64
	avgLoss   float64
	count     int
}

// RSIStrategy evaluates RSI thresholds.
type RSIStrategy struct{}

func NewRSIStrategy() *RSIStrategy { return &RSIStrategy{} }

func (s *RSIStrategy) Type() domain.StrategyType { return domain.StrategyTypeRSI }

func (s *RSIStrategy) Evaluate(strategy domain.Strategy, price float64, state any) (domain.Signal, any, bool, error) {
	cfg, err := decodeRSIConfig(strategy.Config)
	if err != nil {
		return domain.Signal{}, state, false, err
	}
	if cfg.Period <= 0 {
		cfg.Period = 14
	}
	if cfg.Overbought == 0 {
		cfg.Overbought = 70
	}
	if cfg.Oversold == 0 {
		cfg.Oversold = 30
	}

	st, _ := state.(*rsiState)
	if st == nil {
		st = &rsiState{lastPrice: price}
		return domain.Signal{
			StrategyID: strategy.ID,
			Symbol:     strategy.Symbol,
			Action:     domain.SignalActionHold,
			Confidence: 0.5,
			Reason:     "rsi initialized",
			Metadata:   map[string]any{"strategy_type": string(strategy.Type), "rsi": 50.0},
		}, st, false, nil
	}

	delta := price - st.lastPrice
	gain := math.Max(delta, 0)
	loss := math.Max(-delta, 0)
	st.lastPrice = price
	st.count++

	if st.count == 1 {
		st.avgGain = gain
		st.avgLoss = loss
		return domain.Signal{
			StrategyID: strategy.ID,
			Symbol:     strategy.Symbol,
			Action:     domain.SignalActionHold,
			Confidence: 0.5,
			Reason:     "rsi warming up",
			Metadata:   map[string]any{"strategy_type": string(strategy.Type), "rsi": 50.0},
		}, st, false, nil
	}

	period := float64(cfg.Period)
	st.avgGain = ((st.avgGain * (period - 1)) + gain) / period
	st.avgLoss = ((st.avgLoss * (period - 1)) + loss) / period

	rsi := 50.0
	if st.avgLoss > 0 {
		rsi = 100 - (100 / (1 + (st.avgGain / st.avgLoss)))
	}

	action := domain.SignalActionHold
	reason := "rsi hold"
	confidence := 0.55
	if rsi <= cfg.Oversold {
		action = domain.SignalActionBuy
		reason = "rsi oversold"
		confidence = math.Min(0.95, 0.6+(cfg.Oversold-rsi)/100)
	}
	if rsi >= cfg.Overbought {
		action = domain.SignalActionSell
		reason = "rsi overbought"
		confidence = math.Min(0.95, 0.6+(rsi-cfg.Overbought)/100)
	}

	signal := domain.Signal{
		StrategyID: strategy.ID,
		Symbol:     strategy.Symbol,
		Action:     action,
		Confidence: confidence,
		Reason:     reason,
		Metadata: map[string]any{
			"strategy_type": string(strategy.Type),
			"rsi":           rsi,
			"period":        cfg.Period,
		},
	}
	return signal, st, action != domain.SignalActionHold, nil
}

func decodeRSIConfig(raw []byte) (domain.RSIConfig, error) {
	var cfg domain.RSIConfig
	if len(raw) == 0 {
		return domain.RSIConfig{Period: 14, Overbought: 70, Oversold: 30}, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return domain.RSIConfig{}, err
	}
	if cfg.Period == 0 {
		cfg.Period = 14
	}
	if cfg.Overbought == 0 {
		cfg.Overbought = 70
	}
	if cfg.Oversold == 0 {
		cfg.Oversold = 30
	}
	return cfg, nil
}
