# Task 012: Portfolio Service

## Mô tả
Implement Portfolio Service: quản lý danh mục tài sản, vị thế, và lợi nhuận lỗ. Consume execution.filled và market.price.updated events để cập nhật positions, calculate PnL realtime. Cung cấp portfolio snapshot cho risk checks.

## SRS - Requirements
- [ ] Portfolio repository: create, read, update, delete.
- [ ] Portfolio schema: total_equity, available_balance, used_margin, realized_pnl, unrealized_pnl.
- [ ] Positions repository: track per-symbol holdings (side, qty, entry_price, mark_price, fees).
- [ ] Consumer: subscribe execution.filled events → update positions.
- [ ] Consumer: subscribe market.price.updated events → recalculate unrealized_pnl.
- [ ] PnL calculation: realized_pnl = (fill_price - entry_price) * qty - fees, unrealized_pnl = (mark_price - entry_price) * qty.
- [ ] Position averaging: if add to existing position, recalculate entry_price (weighted avg).
- [ ] Portfolio query: GET /portfolio (summary), GET /portfolio/positions (detail).
- [ ] Portfolio projection: eventually consistent, updates within 1s of fill/price event.
- [ ] Margin ratio: used_margin / total_equity, alert if < 50%.

## PRD - Acceptance Criteria
- [ ] POST /portfolio/{user_id} → initialize portfolio with balance.
- [ ] GET /portfolio/{user_id} → {total_equity, available_balance, used_margin, realized_pnl, unrealized_pnl}.
- [ ] GET /portfolio/{user_id}/positions → [{symbol, side, qty, entry_price, mark_price, unrealized_pnl}, ...].
- [ ] Execution filled (0.5 BTC @ 66000) → position created.
- [ ] Price update (66100) → unrealized_pnl recalculated (+$50).
- [ ] Partial fill (0.3 + 0.2) → qty=0.5, entry_price weighted avg.
- [ ] Margin ratio: used_margin / total_equity >= 50% (or alert if < 50%).
- [ ] Multiple symbols processed concurrently (no race conditions).

## Folder Structure
```
cmd/
  └── portfolio-service/
      └── main.go

internal/
  ├── domain/
  │   ├── portfolio.go
  │   ├── position.go
  │   ├── pnl.go
  │   └── margin_info.go
  ├── application/
  │   └── portfolio/
  │       ├── service.go
  │       ├── position_manager.go (update positions from fills)
  │       ├── pnl_calculator.go (calculate PnL)
  │       └── projector.go (eventual consistency)
  ├── infrastructure/
  │   ├── repository/
  │   │   ├── portfolio_repository.go
  │   │   └── position_repository.go
  │   └── cache/
  │       └── portfolio_cache.go
  └── transport/
      └── http/
          └── portfolio_handler.go

tests/
  ├── portfolio_service_test.go
  ├── position_manager_test.go
  ├── pnl_calculator_test.go
  └── projector_test.go
```

## Deliverables
- [ ] ✅ cmd/portfolio-service/main.go
- [ ] ✅ internal/domain/portfolio.go, position.go, pnl.go
- [ ] ✅ internal/application/portfolio/service.go (main service + event consumers)
- [ ] ✅ internal/application/portfolio/position_manager.go (update positions)
- [ ] ✅ internal/application/portfolio/pnl_calculator.go (PnL logic)
- [ ] ✅ internal/application/portfolio/projector.go (eventually consistent updates)
- [ ] ✅ internal/infrastructure/repository/portfolio_repository.go, position_repository.go
- [ ] ✅ internal/transport/http/portfolio_handler.go (query endpoints)
- [ ] ✅ tests/portfolio_service_test.go, position_manager_test.go, pnl_calculator_test.go

## Implementation Notes

### Position Update Flow (from execution.filled)
```
1. Consume execution.filled event:
   {
     "execution_id": "exec-999",
     "order_id": "ord-456",
     "symbol": "BTCUSDT",
     "side": "BUY",
     "fill_price": 66000.50,
     "fill_qty": 0.5,
     "fee": 0.0001 BTC
   }

2. Update POSITIONS table:
   - Check if position exists (symbol + side + user)
   - If new: create {qty: 0.5, entry_price: 66000.50, mark_price: 66000.50}
   - If existing: weighted average entry_price
     new_entry_price = (old_qty * old_entry_price + new_qty * new_price) / (old_qty + new_qty)
   - Add fee to cumulative fees
   - Calc realized_pnl = 0 (not closed yet)

3. Publish portfolio.updated event
```

### PnL Calculation
```
For position:
  unrealized_pnl = (mark_price - entry_price) × qty - cumulative_fees
  realized_pnl = 0  (not closed)

For portfolio (all positions):
  total_unrealized_pnl = Σ(position.unrealized_pnl)
  total_realized_pnl = Σ(closed_position.realized_pnl)
  total_pnl = total_unrealized_pnl + total_realized_pnl

For position closure (sell entire qty):
  realized_pnl = (close_price - entry_price) × qty - entry_fee - close_fee
  position deleted or marked as closed
```

### Market Price Update Flow (from market.price.updated)
```
1. Consume market.price.updated:
   {
     "symbol": "BTCUSDT",
     "price": 66100,
     "bid": 66095,
     "ask": 66105
   }

2. For each position with this symbol:
   - Update mark_price = 66100
   - Recalculate unrealized_pnl = (66100 - entry_price) × qty

3. Recalculate portfolio metrics:
   - total_equity = balance + total_unrealized_pnl
   - used_margin = Σ(position_value)  [position_value = qty × mark_price]
   - available_balance = total_equity - used_margin

4. Publish portfolio.updated event (less frequently, e.g. every 1s batch)
```

### Eventual Consistency
```
- Position updates (from execution): immediate (sync)
- Portfolio summary (from market prices): eventual (within 1s)
- Used by Risk Service to cache portfolio snapshot
- Risk Service polls Portfolio API every 5-10s for latest state
```

### Margin Calculation
```
margin_ratio = available_balance / total_equity

Example:
  total_equity = $100,000
  used_margin = $50,000
  available_balance = $50,000
  margin_ratio = 50,000 / 100,000 = 50% ✅

  If price drops:
  mark_price = 60,000
  position_value = 0.5 × 60,000 = $30,000 (decreased)
  available_balance = $70,000
  margin_ratio = 70,000 / 100,000 = 70% ✅

  If price drops more:
  mark_price = 50,000
  position_value = 0.5 × 50,000 = $25,000
  available_balance = $75,000
  margin_ratio = 75,000 / 100,000 = 75% ✅

  Risk alert: if margin_ratio < 50% → Send alert to user
```

## Effort
6h (Backend 9)

## Timeline
Tuần 6-7 (Ngày 12-13)

## Status
⏳ TODO - Ready to implement
