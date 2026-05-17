# Task 008: Strategy Service

## Mô tả
Implement Strategy Service: quản lý chiến lược, consume market data, sinh signal, support EMA cross, RSI strategies.

## SRS - Requirements
- [ ] Strategy repository: create, read, list, activate/deactivate.
- [ ] Strategy types: ema_cross, rsi (extensible via plugin).
- [ ] EMA cross strategy: fast EMA, slow EMA, signal EMA, crossover logic.
- [ ] RSI strategy: RSI period, overbought/oversold threshold.
- [ ] Consumer: subscribe market.price.updated, evaluate per event.
- [ ] Signal generation: action (buy/sell/hold), confidence, reason.
- [ ] Idempotency: same market event -> same signal (per candle).
- [ ] Publisher: strategy.signal.generated event.

## PRD - Acceptance Criteria
- [ ] Can create EMA cross strategy with config {fast: 12, slow: 26, signal: 9}.
- [ ] Signal generated when EMA cross detected.
- [ ] Signal include metadata: strategy_id, symbol, action, confidence, reason.
- [ ] Activate strategy -> start consuming events, inactive -> stop.
- [ ] Backtest ready (optional): evaluate on historical data.

## Deliverables
- [ ] ✅ cmd/strategy-service/main.go
- [ ] ✅ internal/domain/strategy.go, signal.go
- [ ] ✅ internal/application/strategy/service.go
- [ ] ✅ internal/application/strategy/ema_strategy.go
- [ ] ✅ internal/application/strategy/rsi_strategy.go (optional)
- [ ] ✅ internal/infrastructure/repository/strategy_repository.go
- [ ] ✅ internal/transport/http/strategy_handler.go
- [ ] ✅ tests/strategy_service_test.go, ema_strategy_test.go

## Effort
6h (Backend 5)

## Timeline
Ngày 6-7
