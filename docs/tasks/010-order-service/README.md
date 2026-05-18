# Task 010: Order Service

## Mô tả
Implement Order Service: quản lý vòng đời lệnh (order lifecycle). Tạo lệnh nội bộ từ approved risk decision, theo dõi trạng thái, lưu trữ correlation_id và idempotency key. Gắn kết order với strategy, user, và risk decision.

## SRS - Requirements
- [ ] Order repository: create, read, list, update, delete.
- [ ] Order status machine: created → submitted → filled/partial_filled → closed/canceled/rejected.
- [ ] Consumer: subscribe risk.order.approved events.
- [ ] Order creation: from approved risk decision (contains signal + risk approval).
- [ ] Idempotency key: client_order_id (UUID v4) prevents duplicate orders.
- [ ] Correlation tracking: correlation_id (UUID v4) for end-to-end traceability.
- [ ] Order fields: user_id, strategy_id, symbol, side, order_type, quantity, price, status, fees.
- [ ] Order query: GET by order_id, GET list by user_id/strategy_id, GET by status.
- [ ] Order amendment: cancel order if status is created/submitted (only).
- [ ] State validation: cannot cancel filled or closed orders.

## PRD - Acceptance Criteria
- [ ] POST /orders {signal_id, risk_decision_id} → {order_id, status: created}.
- [ ] GET /orders/{order_id} → full order detail + fills.
- [ ] GET /orders?user_id=X&status=created → list orders by filter.
- [ ] DELETE /orders/{order_id} (cancel) → status: canceled if allowed.
- [ ] Invalid cancel (filled) → 400 Bad Request.
- [ ] Duplicate client_order_id → return existing order (idempotent).
- [ ] Order state transitions validated (no invalid state jumps).
- [ ] Correlation_id present in all outbound events.

## Folder Structure
```
cmd/
  └── order-service/
      └── main.go

internal/
  ├── domain/
  │   ├── order.go
  │   ├── order_status.go
  │   ├── order_type.go
  │   └── order_side.go
  ├── application/
  │   └── order/
  │       ├── service.go
  │       ├── state_machine.go
  │       └── validator.go
  ├── infrastructure/
  │   ├── repository/
  │   │   └── order_repository.go (in-memory, phase 2: PostgreSQL)
  │   └── cache/
  │       └── order_cache.go
  └── transport/
      └── http/
          └── order_handler.go

tests/
  ├── order_service_test.go
  ├── order_state_machine_test.go
  └── order_validator_test.go
```

## Deliverables
- [ ] ✅ cmd/order-service/main.go
- [ ] ✅ internal/domain/order.go, order_status.go, order_type.go
- [ ] ✅ internal/application/order/service.go (Kafka consumer)
- [ ] ✅ internal/application/order/state_machine.go (order state transitions)
- [ ] ✅ internal/application/order/validator.go (order validation rules)
- [ ] ✅ internal/infrastructure/repository/order_repository.go
- [ ] ✅ internal/transport/http/order_handler.go (CRUD endpoints)
- [ ] ✅ tests/order_service_test.go, order_state_machine_test.go

## Implementation Notes
- Order lifecycle: created (local) → submitted (sent to exchange via Execution Service) → filled (matched) → closed (settled).
- Cancelled/rejected states occur if risk rejects or user cancels.
- State machine validates transitions: created→submitted→(filled|partial_filled), cannot go back.
- Idempotency: same client_order_id within 24h → return existing order.
- Correlation ID: generate new for each order, include in all downstream events.
- In-memory storage for MVP; phase 2 persist to PostgreSQL (orders table).
- Order creation publishes order.created event → Execution Service consumes it.

## Effort
7h (Backend 7)

## Timeline
Tuần 5-6 (Ngày 9-10)

## Status
⏳ TODO - Ready to implement
