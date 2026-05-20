# Task 020: Futures Trading Service

## Mô tả
Implement Futures Trading Service: quản lý hợp đồng futures, vị thế long/short, leverage, funding fee, liquidation price, và đồng bộ trạng thái từ execution/market events. Đây là bước mở rộng từ spot trading core hiện tại sang futures trading mode.

## SRS - Requirements
- [ ] Futures domain model: contract, position, funding fee, liquidation info, leverage.
- [ ] Futures portfolio repository: create, read, update, delete futures account state.
- [ ] Position tracking: long/short per symbol, quantity, entry price, mark price, unrealized/realized PnL.
- [ ] Consumer: subscribe execution.filled events and update futures positions.
- [ ] Consumer: subscribe market.price.updated events and recalculate mark-to-market PnL.
- [ ] Funding fee handling: apply periodic funding fee to open positions.
- [ ] Liquidation monitoring: calculate liquidation price and trigger alert when margin is low.
- [ ] Futures risk metrics: leverage, margin ratio, maintenance margin, exposure per symbol.
- [ ] Query API: return futures account snapshot and position details.
- [ ] Thread safety: concurrent symbol updates without race conditions.

## PRD - Acceptance Criteria
- [ ] POST /futures/{user_id} → initialize futures account with balance and leverage settings.
- [ ] GET /futures/{user_id} → return futures account summary.
- [ ] GET /futures/{user_id}/positions → return open futures positions.
- [ ] Execution filled BUY/SELL → create or update long/short futures position.
- [ ] Price update → unrealized_pnl and liquidation_price recalculated.
- [ ] Funding event → funding fee applied to open position and account metrics updated.
- [ ] Margin ratio below threshold → liquidation warning generated.
- [ ] Multiple symbols processed concurrently without data races.

## Folder Structure
```
cmd/
  └── futures-trading-service/
      └── main.go

internal/
  ├── domain/
  │   ├── futures_account.go
  │   ├── futures_position.go
  │   └── funding_event.go
  ├── application/
  │   └── futures/
  │       ├── service.go
  │       ├── position_manager.go
  │       ├── funding_calculator.go
  │       └── liquidation_monitor.go
  ├── infrastructure/
  │   ├── repository/
  │   │   └── futures_repository.go
  │   └── cache/
  │       └── futures_cache.go
  └── transport/
      └── http/
          └── futures_handler.go

tests/
  ├── futures_trading_service_test.go
  ├── position_manager_test.go
  ├── funding_calculator_test.go
  └── liquidation_monitor_test.go
```

## Deliverables
- [ ] ✅ cmd/futures-trading-service/main.go
- [ ] ✅ internal/domain/futures_account.go, futures_position.go, funding_event.go
- [ ] ✅ internal/application/futures/service.go (event consumers + orchestration)
- [ ] ✅ internal/application/futures/position_manager.go (long/short updates)
- [ ] ✅ internal/application/futures/funding_calculator.go (funding fee logic)
- [ ] ✅ internal/application/futures/liquidation_monitor.go (risk monitoring)
- [ ] ✅ internal/infrastructure/repository/futures_repository.go
- [ ] ✅ internal/transport/http/futures_handler.go (query endpoints)
- [ ] ✅ tests/futures_trading_service_test.go, position_manager_test.go, funding_calculator_test.go

## Implementation Notes

### Futures Trading Flow
```
1. Consume execution.filled event for a futures order.
2. Update futures position:
   - BUY opens/increases long or reduces short.
   - SELL opens/increases short or reduces long.
3. Recalculate entry price, unrealized PnL, leverage, and liquidation price.
4. On market.price.updated, refresh mark price and account metrics.
5. On funding interval, apply funding fee to open positions.
6. Emit futures.account.updated / liquidation.alert events when needed.
```

### Core Metrics
- `unrealized_pnl = (mark_price - entry_price) × qty` for long
- `unrealized_pnl = (entry_price - mark_price) × qty` for short
- `funding_fee` adjusts realized PnL or balance depending on position direction
- `liquidation_price` derived from balance, leverage, maintenance margin, and position size

### Design Notes
- Use separate futures account state from spot portfolio state.
- Keep futures position updates idempotent by execution ID.
- Prefer event-driven updates, but allow HTTP query for latest snapshot.
- Start with in-memory repository for MVP; migrate to persistent storage later.

## Effort
8h (Backend TBD)

## Timeline
TBD

## Status
⏳ TODO - Ready to implement