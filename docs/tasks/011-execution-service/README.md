# Task 011: Execution Service

## Mô tả
Implement Execution Service: gửi order lên sàn giao dịch, xử lý retry/timeout/idempotency, reconcile trạng thái fills từ exchange. Đây là layer chuyển đổi giữa hệ thống nội bộ và exchange APIs.

## SRS - Requirements
- [ ] Execution adapter: gửi order qua Exchange Adapter Service (Binance/Bybit/OKX).
- [ ] Consumer: subscribe order.created events.
- [ ] Retry logic: max 3 attempts, exponential backoff (1s, 2s, 4s), 10s timeout per attempt.
- [ ] Idempotency: use client_order_id to prevent duplicate orders on exchange.
- [ ] Order submission: map internal order → exchange API payload.
- [ ] Exchange response: capture exchange_order_id, status, fills.
- [ ] Reconciliation: poll exchange every 5-10s for order status updates.
- [ ] Partial fill handling: track cumulative fills, update order status accordingly.
- [ ] Fill recording: store execution records (price, qty, fee) in EXECUTIONS table.
- [ ] Event publishing: publish execution.submitted, execution.filled, execution.failed events.

## PRD - Acceptance Criteria
- [ ] Order sent to Binance → receive exchange_order_id.
- [ ] Network timeout → retry with exponential backoff (up to 3 times).
- [ ] Failed after 3 retries → mark as FAILED, publish event.
- [ ] Market order → filled immediately (in milliseconds on Binance).
- [ ] Limit order → partial_filled until canceled or expires.
- [ ] Polling every 5-10s detects fill updates, publishes execution.filled event.
- [ ] Same client_order_id sent multiple times → Binance returns same order (idempotent).
- [ ] Concurrent orders (different symbols) processed in parallel.

## Folder Structure
```
cmd/
  └── execution-service/
      └── main.go

internal/
  ├── domain/
  │   ├── execution.go
  │   ├── execution_status.go
  │   └── fill.go
  ├── application/
  │   └── execution/
  │       ├── service.go
  │       ├── submitter.go (send to exchange)
  │       ├── reconciler.go (poll for fills)
  │       └── retry_policy.go
  ├── infrastructure/
  │   ├── exchange/
  │   │   ├── executor.go (calls Binance API)
  │   │   └── binance_executor.go (Binance-specific implementation)
  │   ├── repository/
  │   │   └── execution_repository.go
  │   └── cache/
  │       └── pending_orders_cache.go
  └── transport/
      └── http/
          └── execution_handler.go (monitoring endpoints)

tests/
  ├── execution_service_test.go
  ├── submitter_test.go
  ├── reconciler_test.go
  └── retry_policy_test.go
```

## Deliverables
- [x] ✅ cmd/execution-service/main.go
- [x] ✅ internal/domain/execution.go, fill.go
- [x] ✅ internal/application/execution/service.go (main orchestrator)
- [x] ✅ internal/application/execution/submitter.go (send order logic)
- [x] ✅ internal/application/execution/reconciler.go (polling logic)
- [x] ✅ internal/application/execution/retry_policy.go (retry + backoff)
- [x] ✅ internal/infrastructure/exchange/executor.go (exchange adapter interface)
- [x] ✅ internal/infrastructure/exchange/binance_executor.go (Binance implementation)
- [x] ✅ tests/execution_service_test.go, submitter_test.go, reconciler_test.go

## Implementation Notes

### Submission Flow
```
1. Consume order.created event
2. Map to Binance request:
   {
     "symbol": order.symbol,
     "side": order.side.String(),  // BUY / SELL
     "type": order.order_type.String(),  // MARKET / LIMIT
     "quantity": order.quantity,
     "price": order.price,  // null for market orders
     "newClientOrderId": order.client_order_id  // Idempotency key
   }
3. POST /api/v3/order with retry (3 attempts, 1s/2s/4s backoff)
4. On success: capture exchange_order_id, publish execution.submitted
5. On failure after 3 retries: publish execution.failed
```

### Reconciliation Flow
```
1. Maintain pending_orders_cache (orders in submitted/partial_filled state)
2. Every 5-10s: poll each pending order
3. GET /api/v3/orders?symbol=X&orderId=exchange_order_id
4. Compare fills against last known state
5. If new fills detected:
   - Update EXECUTIONS table
   - Update ORDER status (filled/partial_filled)
   - Publish execution.filled event
6. If order expired/canceled:
   - Update ORDER status (canceled)
   - Publish execution.canceled event
```

### Retry Policy
```
Attempt 1:
  ├─ Send request
  ├─ Timeout: 10s
  └─ Backoff: 1s (if failed)

Attempt 2:
  ├─ Send same request (same client_order_id)
  ├─ Binance: "Order with this client_order_id already exists"
  ├─ Return previous result
  └─ Backoff: 2s (if still failed)

Attempt 3:
  ├─ Final retry
  ├─ If timeout/error: mark FAILED
  └─ Publish execution.failed
```

### Concurrent Processing
```
- Go routines for each pending order reconciliation
- Channel-based work queue for submissions
- sync.WaitGroup for batch operations
- Rate limit: Binance API weight limit (1200 requests per minute)
```

## Effort
8h (Backend 8)

## Timeline
Tuần 6 (Ngày 11-12)

## Status
✅ COMPLETED - Build and tests verified
