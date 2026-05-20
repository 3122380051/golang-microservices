# Task-011: Execution Service

## Overview

The **Execution Service** is responsible for submitting orders to Binance, handling fills, and publishing execution events to Kafka. It operates as a critical link in the order fulfillment pipeline, receiving `order.created` events from the Order Service and orchestrating exchange interactions.

**Port**: 8087  
**Consumes**: `order.created` events from Kafka  
**Produces**: `execution.submitted`, `execution.filled` events to Kafka

---

## Domain Models

### Execution
Represents a single order execution at the exchange:

```go
type Execution struct {
    ID                string          // UUID
    OrderID           string          // From Order Service
    ClientOrderID     string          // Idempotency key
    CorrelationID     string          // For tracing
    ExchangeOrderID   string          // From Binance
    UserID            string
    Symbol            string          // e.g., BTCUSDT
    Side              OrderSide       // BUY or SELL
    OriginalQuantity  float64
    ExecutedQuantity  float64
    ExecutedValue     float64         // Notional value
    AverageFillPrice  float64
    Fees              float64
    Status            ExecutionStatus // created → submitted → filled → closed
    AttemptCount      int             // Submission retry count
    LastAttemptError  string
    SubmittedAt       *time.Time
    FirstFilledAt     *time.Time
    ClosedAt          *time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

### ExecutionStatus
State machine for order execution:

- `created` - Initial state
- `submitting` - Attempting to submit to exchange
- `submitted` - Successfully sent to Binance
- `partial_filled` - Partially filled
- `filled` - Completely filled
- `canceling` - Attempting to cancel
- `canceled` - User/system canceled
- `failed` - Submission failed after retries
- `closed` - Settlement complete (terminal)

### FillRecord
Represents a single fill event from Binance:

```go
type FillRecord struct {
    TradeID       string    // Binance trade ID
    ExecutionID   string
    Quantity      float64
    Price         float64
    Fee           float64
    FeeAsset      string
    FilledAt      time.Time
    ReceivedAt    time.Time
}
```

### ExecutionEvent
Published to Kafka when state changes:

```go
type ExecutionEvent struct {
    EventID         string      // UUID
    ExecutionID     string
    OrderID         string
    ClientOrderID   string
    CorrelationID   string
    ExchangeOrderID string
    UserID          string
    Symbol          string
    Side            OrderSide
    OriginalQty     float64
    ExecutedQty     float64
    AverageFillPrice float64
    Fees            float64
    Status          ExecutionStatus
    EventType       string      // "execution.submitted" or "execution.filled"
    Timestamp       time.Time
    TraceID         string
}
```

---

## Application Layer

### Service
Main orchestrator for execution operations:

```go
type Service struct {
    logger          *slog.Logger
    repository      domain.ExecutionRepository
    producer        *broker.KafkaProducer
    consumer        *broker.KafkaConsumer
    exchangeAdapter exchange.ExchangeAdapter
    submitter       *Submitter
    reconciler      *Reconciler
}
```

**Key Methods**:

#### ConsumeOrderCreated(ctx context.Context)
- Listens for `order.created` events from Kafka
- Creates Execution records
- Triggers submission to Binance
- Idempotent: skips if order already processed

#### submitExecution(ctx, execution, traceID)
- Calls `submitter.Submit()` with retry logic
- Updates execution status to `submitted` on success
- Publishes `execution.submitted` event
- Handles and logs submission failures

#### StartReconciliationLoop(ctx context.Context)
- Continuously polls for fills (5-second intervals)
- Lists all `submitted` and `partial_filled` executions
- Calls `reconciler.GetFills()` for each active order
- Updates execution with new fill data
- Transitions to `filled` when complete
- Publishes `execution.filled` event when filled

#### ListExecutionsByUser(ctx, userID, status)
- Query method for HTTP handlers

### Submitter
Handles order submission with retry logic:

```go
type Submitter struct {
    logger       *slog.Logger
    adapter      exchange.ExchangeAdapter
    maxRetries   int               // Default: 3
    baseInterval time.Duration      // Default: 1 second
    attempts     map[string]int
}
```

**Retry Strategy**:
- **Exponential Backoff**: 1s, 2s, 4s between attempts
- **Max Attempts**: 3 (configurable)
- **Total Attempts**: 4 (initial + 3 retries)
- **Failure Handling**: On final failure, status set to `failed`

**Submit Flow**:
1. Attempt 1: Try immediately
2. If fails → Wait 1s, Attempt 2
3. If fails → Wait 2s, Attempt 3
4. If fails → Wait 4s, Attempt 4
5. If all fail → Return error

### Reconciler
Handles polling for fills from Binance:

```go
type Reconciler struct {
    logger       *slog.Logger
    adapter      exchange.ExchangeAdapter
    pollInterval time.Duration     // Default: 5 seconds
}
```

**Key Methods**:
- `GetFills(ctx, exchangeOrderID, symbol)` - Query Binance for fills
- Returns list of `FillRecord` objects
- Handles network errors gracefully

---

## Infrastructure Layer

### InMemoryExecutionRepository
MVP implementation (production should use PostgreSQL):

```go
type InMemoryExecutionRepository struct {
    mu           sync.RWMutex
    executions   map[string]*Execution
    byExchangeID map[string]*Execution
}
```

**Features**:
- Thread-safe concurrent access
- Idempotency via `ClientOrderID` deduplication
- Lookups by ID and ExchangeOrderID
- Filtering by user and status

**Methods**:
- `Create(ctx, execution)` - Idempotent create
- `GetByID(ctx, id)` - Lookup by UUID
- `GetByClientOrderID(ctx, clientOrderID)` - Lookup by client ID
- `GetByExchangeOrderID(ctx, exchangeOrderID)` - Lookup by Binance ID
- `Update(ctx, execution)` - Persist updates
- `ListByUser(ctx, userID, status)` - Query by user and optional status
- `ListByStatus(ctx, status)` - Query by status

---

## Transport Layer

### HTTP Handler
RESTful endpoints for execution queries:

#### GET /executions/{id}
Get execution details by ID

```bash
curl http://localhost:8087/executions/exec-uuid-here
```

Response:
```json
{
  "id": "exec-1",
  "order_id": "order-1",
  "client_order_id": "client-1",
  "exchange_order_id": "binance-order-123",
  "user_id": "user1",
  "symbol": "BTCUSDT",
  "side": "BUY",
  "original_quantity": 1.0,
  "executed_quantity": 1.0,
  "average_fill_price": 50000.0,
  "fees": 10.0,
  "status": "filled",
  "attempt_count": 1,
  "created_at": "2026-05-20T10:00:00Z",
  "updated_at": "2026-05-20T10:00:05Z"
}
```

#### GET /executions?user_id=X&status=Y
List executions with optional filters

```bash
curl "http://localhost:8087/executions?user_id=user1&status=submitted"
```

#### GET /health
Service health check

#### GET /ready
Service readiness probe

---

## State Machine Transitions

```
created → submitting
    ↓
submitting → submitted
    ↓
submitted → partial_filled
    ↓
partial_filled → filled
    ↓
filled → closed

Valid alternative paths:
- Any state → failed (on error)
- submitted → canceling → canceled
- partial_filled → canceling → canceled
- canceling → failed
```

---

## Kafka Events

### Consumed
**Topic**: `order.created`

```json
{
  "event_id": "evt-1",
  "order_id": "order-1",
  "client_order_id": "client-1",
  "correlation_id": "corr-1",
  "user_id": "user1",
  "symbol": "BTCUSDT",
  "side": "BUY",
  "quantity": 1.0,
  "created_at": "2026-05-20T10:00:00Z"
}
```

### Published
**Topic**: `execution.submitted`

```json
{
  "event_id": "evt-2",
  "execution_id": "exec-1",
  "order_id": "order-1",
  "exchange_order_id": "binance-order-123",
  "status": "submitted",
  "event_type": "execution.submitted",
  "timestamp": "2026-05-20T10:00:01Z"
}
```

**Topic**: `execution.filled`

```json
{
  "event_id": "evt-3",
  "execution_id": "exec-1",
  "status": "filled",
  "executed_qty": 1.0,
  "average_fill_price": 50000.0,
  "fees": 10.0,
  "event_type": "execution.filled",
  "timestamp": "2026-05-20T10:00:05Z"
}
```

---

## Error Handling & Resilience

### Submission Failures
- Initial attempt fails → Retry with exponential backoff
- After 4 total attempts → Status = `failed`
- Error message stored in `last_attempt_error`
- Operator can manually retry or investigate

### Reconciliation Failures
- Network error fetching fills → Log and skip for now
- Next poll (5s later) will retry
- No data loss: executions persist across polling attempts

### Idempotency
- **Client Order ID** acts as idempotency key
- Creating execution with same `ClientOrderID` returns existing record
- Prevents duplicate orders even with retries
- Supports at-least-once Kafka delivery

---

## Implementation Notes

### Polling Interval
- Default: 5 seconds
- Configurable via `Reconciler.pollInterval`
- Balance between latency (fast fills) and API rate limits

### Retry Configuration
- Default: 3 retries (4 total attempts)
- Configurable in `NewSubmitter()`
- Backoff starts at 1 second, doubles each time
- Total max wait = 1 + 2 + 4 = 7 seconds

### Thread Safety
- All state machines guarded by sync.RWMutex
- Concurrent Kafka consumers and HTTP requests supported
- Repository operations thread-safe

### Production Considerations

#### Database
Replace `InMemoryExecutionRepository` with PostgreSQL:
```go
// config/migration.sql
CREATE TABLE executions (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    client_order_id VARCHAR(255) UNIQUE NOT NULL,
    exchange_order_id VARCHAR(255),
    user_id VARCHAR(255) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL,
    original_quantity DECIMAL(18,8) NOT NULL,
    executed_quantity DECIMAL(18,8) DEFAULT 0,
    average_fill_price DECIMAL(18,8),
    fees DECIMAL(18,8),
    status VARCHAR(50) NOT NULL,
    attempt_count INT DEFAULT 0,
    submitted_at TIMESTAMP,
    first_filled_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    INDEX(user_id),
    INDEX(status),
    INDEX(exchange_order_id)
);
```

#### Monitoring
- Track submission retry rates
- Alert on repeated failures for same order
- Monitor reconciliation lag (current fills vs. latest)
- Gauge active executions by status

#### Disaster Recovery
- Replay `order.created` events from Kafka to rebuild execution state
- Reconciliation loop automatically corrects stale fill data
- Client order IDs prevent resubmission duplicates

---

## Testing

Run tests:
```bash
go test ./tests -run Execution
```

**Test Coverage**:
- ✅ State machine transitions
- ✅ Idempotent creation via ClientOrderID
- ✅ Retry logic with exponential backoff
- ✅ Repository CRUD operations
- ✅ Kafka event consumption and publishing
- ✅ Fill reconciliation

---

## Files Created

```
internal/domain/execution.go
    ├── Execution struct
    ├── ExecutionStatus enum
    ├── FillRecord struct
    └── ExecutionEvent struct

internal/application/execution/
    ├── service.go       (main orchestrator)
    ├── submitter.go     (retry logic)
    └── reconciler.go    (polling fills)

internal/infrastructure/execution_repository.go
    └── InMemoryExecutionRepository

internal/transport/http/execution_handler.go
    └── ExecutionHandler (HTTP routes)

cmd/execution-service/main.go
    └── Service bootstrap

tests/execution_service_test.go
    └── Unit tests
```

---

## Environment Variables

```bash
APP_HTTP_ADDR=:8087
APP_LOG_LEVEL=info
APP_DATABASE_URL=postgres://...
APP_KAFKA_BROKERS=localhost:9092
```

---

## Next Steps

1. **Execution Service Integration** → Run with Kafka + Order/Risk services
2. **End-to-End Testing** → Full pipeline: Strategy → Order → Risk → Execution
3. **Market Data Updates** → Polling for candles and tickers
4. **Portfolio Service** → Track filled orders and balance updates
5. **Notification Service** → Alert traders on order fills

---
