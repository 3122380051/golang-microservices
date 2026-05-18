# Task 009: Risk Service

## Mô tả
Implement Risk Service: kiểm tra rủi ro trước khi lệnh được chấp nhận. Đánh giá position size, leverage, margin ratio, max daily loss, exposure limits. Phê duyệt hoặc từ chối signal dựa trên risk policies.

## SRS - Requirements
- [ ] Risk policy evaluation engine: position size, leverage, margin, daily loss checks.
- [ ] Consumer: subscribe strategy.signal.generated events.
- [ ] Risk policies: configurable per user/strategy (max_position_size, max_leverage, max_daily_loss, min_margin_ratio).
- [ ] Risk approval/rejection: publish risk.order.approved or risk.order.rejected events.
- [ ] Portfolio state integration: get current positions, balance, unrealized PnL from cache/DB.
- [ ] Alert generation: notify user if margin_ratio < threshold.
- [ ] Audit logging: log all risk decisions with trace_id.
- [ ] Concurrent policy evaluation: thread-safe risk checks.

## PRD - Acceptance Criteria
- [ ] Risk policy endpoint: GET /risk/policies, POST /risk/policies (CRUD).
- [ ] Risk check returns: approved/rejected with reason (e.g., "leverage > max_leverage").
- [ ] Position size check: internal_order.qty * price <= max_position_size.
- [ ] Leverage check: position_value / account_equity <= user_max_leverage.
- [ ] Margin check: available_margin / used_margin >= 50%.
- [ ] Daily loss check: realized_pnl_today >= -max_daily_loss.
- [ ] Multiple signals processed concurrently without race conditions.
- [ ] Risk rejection -> event published, order flow halted gracefully.

## Folder Structure
```
cmd/
  └── risk-service/
      └── main.go

internal/
  ├── domain/
  │   ├── risk_policy.go
  │   ├── risk_decision.go
  │   └── portfolio_snapshot.go
  ├── application/
  │   └── risk/
  │       ├── service.go
  │       ├── evaluator.go
  │       └── policies.go
  ├── infrastructure/
  │   ├── repository/
  │   │   └── risk_policy_repository.go
  │   └── cache/
  │       └── portfolio_cache.go (reads from Portfolio Service)
  └── transport/
      └── http/
          ├── risk_handler.go
          └── webhook_handler.go (receives portfolio updates)

tests/
  ├── risk_service_test.go
  ├── risk_evaluator_test.go
  └── risk_policies_test.go
```

## Deliverables
- [ ] ✅ cmd/risk-service/main.go
- [ ] ✅ internal/domain/risk_policy.go, risk_decision.go
- [ ] ✅ internal/application/risk/service.go (Kafka consumer, event publisher)
- [ ] ✅ internal/application/risk/evaluator.go (risk check logic)
- [ ] ✅ internal/application/risk/policies.go (predefined policies)
- [ ] ✅ internal/infrastructure/repository/risk_policy_repository.go (in-memory storage for MVP)
- [ ] ✅ internal/transport/http/risk_handler.go (CRUD endpoints)
- [ ] ✅ tests/risk_service_test.go, risk_evaluator_test.go

## Implementation Notes
- Risk evaluator dùng decision tree: check position size → leverage → margin → daily loss.
- Nếu bất kỳ check fail → return rejected với reason cụ thể.
- Portfolio snapshot lấy từ cache (or polling Portfolio Service API trên phase sau).
- Idempotency: same signal_id → same risk decision (via processedSignal map).
- Policies stored in-memory for MVP; phase 2 move to DB.
- Concurrent evaluation: dùng sync.Mutex để protect policy state.

## Effort
6h (Backend 6)

## Timeline
Tuần 5 (Ngày 8-9)

## Status
⏳ TODO - Ready to implement
