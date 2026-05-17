# Task 008: Strategy Service

## Mô tả
Implement Strategy Service: quản lý chiến lược, consume market data, sinh signal, support EMA cross, RSI strategies.

## SRS - Requirements
- [x] Strategy repository: create, read, list, activate/deactivate.
- [x] Strategy types: ema_cross, rsi (extensible via engine interface).
- [x] EMA cross strategy: fast EMA, slow EMA, signal EMA, crossover logic.
- [x] RSI strategy: RSI period, overbought/oversold threshold.
- [x] Consumer: subscribe market.price.updated, evaluate per event.
- [x] Signal generation: action (buy/sell/hold), confidence, reason.
- [x] Idempotency: same market event -> same signal (per event id).
- [x] Publisher: strategy.signal.generated event.

## PRD - Acceptance Criteria
- [x] Can create EMA cross strategy with config {fast: 12, slow: 26, signal: 9}.
- [x] Signal generated when EMA cross detected.
- [x] Signal include metadata: strategy_id, symbol, action, confidence, reason.
- [x] Activate strategy -> start consuming events, inactive -> stop.
- [ ] Backtest ready (optional): evaluate on historical data.

## Deliverables
- [x] ✅ cmd/strategy-service/main.go
- [x] ✅ internal/domain/strategy.go, signal.go
- [x] ✅ internal/application/strategy/service.go
- [x] ✅ internal/application/strategy/ema_strategy.go
- [x] ✅ internal/application/strategy/rsi_strategy.go (optional)
- [x] ✅ internal/infrastructure/repository/strategy_repository.go
- [x] ✅ internal/transport/http/strategy_handler.go
- [x] ✅ tests/strategy_service_test.go, ema_strategy_test.go

## Effort
6h (Backend 5)

## Timeline
Ngày 6-7
