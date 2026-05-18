# Phase 2 Implementation Plan - Trading Core (Order Matching & Execution)

## 🎯 Phase 2 Overview

**Objective**: Complete the order matching pipeline from strategy signal → Binance execution → portfolio update  
**Duration**: 27 hours across 4 services  
**Timeline**: Week 5-6 (Days 8-13)  
**Platforms**: Risk Service, Order Service, Execution Service, Portfolio Service

---

## 📋 Detailed Task Sequence

### TASK-009: Risk Service (6 hours) - Days 8-9

**Purpose**: Evaluate risk policies before order creation. Prevent over-leverage, over-exposure, excessive daily losses.

**Key Components**:
1. **Risk Evaluator Engine**
   - Position size check: `position_value <= max_position_size`
   - Leverage check: `position_value / account_equity <= max_leverage`
   - Margin check: `available_balance / total_equity >= 50%`
   - Daily loss check: `realized_pnl_today >= -max_daily_loss`

2. **Decision Tree**
   ```
   Signal arrives
   └─ Check position size
      ├─ FAIL → Reject with reason
      └─ PASS → Check leverage
         ├─ FAIL → Reject
         └─ PASS → Check margin
            ├─ FAIL → Reject
            └─ PASS → Check daily loss
               ├─ FAIL → Reject
               └─ PASS → Approve ✅
   ```

3. **State**
   - Policies: stored in-memory (map[string]RiskPolicy)
   - Portfolio snapshot: cached from Portfolio Service (or DB)
   - Processed signals: deduplicate by signal_id

4. **Events**
   - **Consume**: `strategy.signal.generated`
   - **Publish**: `risk.order.approved` or `risk.order.rejected`

**Deliverables**:
- `cmd/risk-service/main.go`
- `internal/domain/risk_policy.go`, `risk_decision.go`
- `internal/application/risk/service.go`, `evaluator.go`, `policies.go`
- `internal/transport/http/risk_handler.go` (CRUD policies)
- Unit tests (evaluator, policies)

**Port**: 8085

---

### TASK-010: Order Service (7 hours) - Days 9-10

**Purpose**: Manage internal order lifecycle. Create, validate, update, track orders from creation to submission.

**Key Components**:
1. **Order State Machine**
   ```
   created
   └─→ submitted
       ├─→ filled (match complete)
       ├─→ partial_filled (some qty matched)
       ├─→ canceled (user cancel or risk reject)
       └─→ rejected (validation failed)
   ```

2. **Idempotency**
   - `client_order_id` (UUID): prevents duplicate orders
   - Check cache: if same `client_order_id` exists → return cached order
   - Store for 24 hours

3. **Order Fields**
   ```go
   type Order struct {
       OrderID       string  // internal order id
       UserID        string
       StrategyID    string
       Symbol        string
       Side          string  // buy/sell
       OrderType     string  // market/limit
       Quantity      float64
       Price         float64 // null for market
       Status        string  // state machine
       ClientOrderID string  // idempotency key ⭐
       CorrelationID string  // traceability
       CreatedAt     time.Time
       UpdatedAt     time.Time
   }
   ```

4. **Events**
   - **Consume**: `risk.order.approved` (triggers order creation)
   - **Publish**: `order.created`

**Deliverables**:
- `cmd/order-service/main.go`
- `internal/domain/order.go`, `order_status.go`
- `internal/application/order/service.go`, `state_machine.go`, `validator.go`
- `internal/infrastructure/repository/order_repository.go` (in-memory)
- `internal/transport/http/order_handler.go`
- Unit tests (state machine, validator)

**Port**: 8086

---

### TASK-011: Execution Service (8 hours) - Days 11-12 ⭐ **CRITICAL**

**Purpose**: Send orders to Binance, handle retries/timeouts, poll for fills, reconcile execution results.

**This is where actual order MATCHING happens via Binance.**

**Key Components**:

#### 1. Order Submission
```go
// submitter.go
func (s *Submitter) Submit(ctx context.Context, order Order) error {
    // Build Binance request
    binanceReq := &BinanceOrderRequest{
        Symbol:           order.Symbol,
        Side:             order.Side.ToTradingSide(),  // BUY/SELL
        Type:             order.OrderType.String(),     // MARKET/LIMIT
        Quantity:         order.Quantity,
        Price:            order.Price,  // nil for market
        NewClientOrderId: order.ClientOrderID,  // Idempotency ⭐
    }
    
    // Retry loop: 3 attempts, exponential backoff
    for attempt := 1; attempt <= 3; attempt++ {
        resp, err := s.binanceClient.CreateOrder(ctx, binanceReq)
        if err == nil {
            // Success
            return s.handleSubmissionSuccess(ctx, order, resp)
        }
        
        if ctx.Err() != nil {
            return err  // Context canceled
        }
        
        if attempt < 3 {
            backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
            time.Sleep(backoff)
        }
    }
    
    // 3 attempts failed
    return s.handleSubmissionFailure(ctx, order)
}
```

#### 2. Retry Policy
| Attempt | Action | Backoff | Timeout |
|---------|--------|---------|---------|
| 1 | Send to Binance | - | 10s |
| 2 (if failed) | Resend (same client_order_id) | 1s | 10s |
| 3 (if failed) | Final retry (same client_order_id) | 2s | 10s |
| After 3 | Mark FAILED | - | - |

Binance idempotency: same `newClientOrderId` → returns previous result (no duplicate)

#### 3. Fill Reconciliation (Polling)
```go
// reconciler.go
func (r *Reconciler) ReconcileLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)  // Poll every 5-10s
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            r.reconcileAllPendingOrders(ctx)
        }
    }
}

func (r *Reconciler) reconcileAllPendingOrders(ctx context.Context) {
    pending := r.pendingOrdersCache.GetAll()
    
    for _, order := range pending {
        status, err := r.binanceClient.GetOrderStatus(ctx, order.Symbol, order.ExchangeOrderID)
        if err != nil {
            continue  // retry next cycle
        }
        
        if status.Status == "FILLED" || status.Status == "PARTIALLY_FILLED" {
            r.handleNewFills(ctx, order, status.Fills)
        } else if status.Status == "CANCELED" || status.Status == "EXPIRED" {
            r.handleOrderCanceled(ctx, order)
        }
    }
}
```

#### 4. Execution Records
```go
// EXECUTIONS table schema (future PostgreSQL)
type Execution struct {
    ExecutionID      string    // execution_id
    OrderID          string    // internal order_id (FK)
    ExchangeOrderID  string    // Binance order id (for reconciliation)
    Exchange         string    // "binance"
    FillPrice        float64
    FillQty          float64
    Fee              float64   // in BTC/ETH/USDT (asset depends)
    FeeAsset         string    // e.g., "BTC"
    ExecutedAt       time.Time
}
```

#### 5. Event Flow
```
order.created (consumed)
└─→ Submit to Binance
    ├─ Success → store exchange_order_id → Publish execution.submitted
    └─ Failure (3 retries) → Publish execution.failed

[Reconciliation Loop every 5-10s]
└─→ Poll Binance for fills
    ├─ New fills → Record execution → Update order status → Publish execution.filled
    └─ No new fills → Wait for next cycle
```

#### 6. Idempotency in Action
```
Scenario: Network timeout on first submission

Time 1: Send order
  ├─ Binance receives → creates order → fills immediately
  ├─ Response timeout (10s exceeded)
  └─ We retry

Time 2: Resend with same client_order_id
  ├─ Binance recognizes same client_order_id
  ├─ Returns previous result (already filled)
  ├─ We record execution
  └─ SUCCESS (no duplicate order created) ✅
```

**Deliverables**:
- `cmd/execution-service/main.go`
- `internal/domain/execution.go`, `fill.go`
- `internal/application/execution/service.go` (main orchestrator)
- `internal/application/execution/submitter.go` (send orders)
- `internal/application/execution/reconciler.go` (poll fills)
- `internal/application/execution/retry_policy.go` (backoff logic)
- `internal/infrastructure/exchange/executor.go` (interface)
- `internal/infrastructure/exchange/binance_executor.go` (Binance impl)
- `internal/infrastructure/cache/pending_orders_cache.go`
- Unit tests (submitter, reconciler, retry_policy)

**Port**: 8087

**Critical Success Factors**:
1. Idempotency works correctly (no duplicate orders)
2. Retry logic handles timeouts properly
3. Reconciliation detects all fills without missing any
4. Concurrent order handling (multiple symbols simultaneously)

---

### TASK-012: Portfolio Service (6 hours) - Days 12-13

**Purpose**: Track positions and calculate P&L in real-time.

**Key Components**:

#### 1. Position Tracking
```go
type Position struct {
    Symbol         string    // BTCUSDT, ETHUSDT
    Side           string    // long/short (MVP: long only)
    Quantity       float64   // qty held
    EntryPrice     float64   // weighted average entry price
    MarkPrice      float64   // current market price
    CumulativeFees float64   // total fees paid (in quote asset)
    OpenedAt       time.Time
    UpdatedAt      time.Time
}
```

#### 2. PnL Calculation
```
Unrealized P&L = (MarkPrice - EntryPrice) × Quantity - CumulativeFees

Example:
  ├─ Entry: buy 0.5 BTC @ 66,000 → fee 0.0001 BTC
  ├─ Mark price: 66,100
  └─ Unrealized P&L = (66,100 - 66,000) × 0.5 - 0.0001
                     = 100 × 0.5 - 0.0001
                     = 50 - 0.0001 BTC
                     ≈ 50 BTC or ~$3,305 USD (at 66k/BTC)
```

#### 3. Event Consumers

**a) execution.filled event** (update positions)
```
Execution arrives:
  ├─ symbol: BTCUSDT, side: buy, fill_qty: 0.5, fill_price: 66000.50, fee: 0.0001
  └─ Update Position:
     ├─ If new: create position with qty=0.5, entry_price=66000.50
     ├─ If existing: recalculate weighted entry_price
     │   new_entry = (old_qty × old_price + new_qty × new_price) / (old_qty + new_qty)
     └─ Add fee to cumulative_fees
```

**b) market.price.updated event** (recalculate unrealized P&L)
```
Price update arrives:
  ├─ symbol: BTCUSDT, price: 66100
  └─ For each position with BTCUSDT:
     └─ Update mark_price: 66100
        └─ Recalculate unrealized_pnl
```

#### 4. Portfolio Summary
```go
type Portfolio struct {
    UserID              string     // portfolio owner
    TotalEquity         float64    // balance + unrealized_pnl
    AvailableBalance    float64    // free to trade
    UsedMargin          float64    // locked in positions
    UnrealizedPnL       float64    // Σ(position.unrealized_pnl)
    RealizedPnL         float64    // from closed positions
    MarginRatio         float64    // available_balance / total_equity
    UpdatedAt           time.Time
}

Calculation:
  ├─ TotalEquity = InitialBalance + UnrealizedPnL + RealizedPnL
  ├─ UsedMargin = Σ(position.qty × position.mark_price)
  ├─ AvailableBalance = TotalEquity - UsedMargin
  └─ MarginRatio = AvailableBalance / TotalEquity
```

#### 5. Margin Alert
```
If MarginRatio < 50%:
  ├─ Alert: "Low margin! Only 45% available"
  └─ Publish: margin.alert event (for Notification Service)
```

**Deliverables**:
- `cmd/portfolio-service/main.go`
- `internal/domain/portfolio.go`, `position.go`, `pnl.go`
- `internal/application/portfolio/service.go` (event consumers)
- `internal/application/portfolio/position_manager.go` (update positions)
- `internal/application/portfolio/pnl_calculator.go` (PnL logic)
- `internal/infrastructure/repository/portfolio_repository.go`, `position_repository.go`
- `internal/transport/http/portfolio_handler.go` (GET portfolio, positions)
- Unit tests (pnl_calculator, position_manager)

**Port**: 8088

---

## 🔗 Data Flow (Complete Pipeline)

```
1. Market Data Service (Polling every 1s)
   └─ Fetch BTCUSDT price → publish market.price.updated
      Event: { symbol: "BTCUSDT", price: 66100 }

2. Strategy Service (Kafka consumer)
   ├─ Consume market.price.updated
   ├─ Evaluate EMA strategy
   └─ Generate signal → publish strategy.signal.generated
      Event: { symbol: "BTCUSDT", action: "buy", confidence: 0.87 }

3. Risk Service (Kafka consumer)
   ├─ Consume strategy.signal.generated
   ├─ Evaluate policies
   │  ├─ Check position size ✅
   │  ├─ Check leverage ✅
   │  ├─ Check margin ✅
   │  └─ Check daily loss ✅
   └─ Approve → publish risk.order.approved
      Event: { signal_id: "sig-123", approved: true }

4. Order Service (Kafka consumer)
   ├─ Consume risk.order.approved
   ├─ Create internal order
   │  {
   │    order_id: "ord-456",
   │    symbol: "BTCUSDT",
   │    side: "buy",
   │    qty: 0.5,
   │    client_order_id: "cli-uuid-001"
   │  }
   └─ Publish order.created
      Event: { order_id: "ord-456", status: "created" }

5. Execution Service (Kafka consumer)
   ├─ Consume order.created
   ├─ Submit to Binance (with retries)
   ├─ Binance MATCHING ENGINE (Binance's internal matching)
   │  └─ Match against orderbook
   │     └─ Fill: 0.5 BTC @ 66100, fee: 0.0001 BTC
   ├─ Receive response → store exchange_order_id
   ├─ Publish execution.submitted
   │  Event: { order_id: "ord-456", exchange_order_id: "12345678" }
   └─ [5-10s polling loop]
      ├─ Check fill status
      └─ Publish execution.filled
         Event: { order_id: "ord-456", fill_price: 66100, fill_qty: 0.5 }

6. Portfolio Service (Kafka consumer)
   ├─ Consume execution.filled
   ├─ Update position:
   │  {
   │    symbol: "BTCUSDT",
   │    qty: 0.5,
   │    entry_price: 66100,
   │    mark_price: 66100
   │  }
   ├─ Consume market.price.updated (continuous)
   │  └─ Update mark_price: 66150
   │     └─ Unrealized P&L: (66150 - 66100) × 0.5 = 25 BTC
   └─ Publish portfolio.updated
      Event: { symbol: "BTCUSDT", unrealized_pnl: 25 }
```

---

## ✅ Acceptance Criteria (All Tasks)

### TASK-009: Risk Service
- [ ] Position size check prevents over-exposure
- [ ] Leverage check prevents over-leverage
- [ ] Margin check ensures 50% minimum
- [ ] Daily loss check prevents excessive losses
- [ ] Multiple signals processed concurrently

### TASK-010: Order Service
- [ ] Order created with unique client_order_id
- [ ] State transitions validated (no invalid jumps)
- [ ] Duplicate client_order_id → return cached order (idempotent)
- [ ] Order fetched by id, user_id, status

### TASK-011: Execution Service
- [ ] Order submitted to Binance (retry up to 3 times)
- [ ] Timeout handled with exponential backoff
- [ ] Fills reconciled via polling (5-10s intervals)
- [ ] Partial fills tracked correctly
- [ ] execution.filled event published for all fills

### TASK-012: Portfolio Service
- [ ] Position created/updated on execution.filled
- [ ] Entry price weighted average on partial fills
- [ ] Unrealized PnL recalculated on market.price.updated
- [ ] Margin ratio calculated correctly
- [ ] Alert published when margin < 50%

---

## 📊 Architecture Summary (Phase 2)

```
┌─────────────────────────────────────────────────────────────┐
│ COMPLETE TRADING PIPELINE (Phase 1 + Phase 2)              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ Market Data Service (port 8083) ────────┐                  │
│         ↓ (market.price.updated)        │                  │
│ Strategy Service (port 8084) ────────┐  │                  │
│         ↓ (strategy.signal.generated) │  │                  │
│ Risk Service (port 8085) ────────┐   │  │                  │
│         ↓ (risk.order.approved)   │   │  │                  │
│ Order Service (port 8086) ────────┤───┤──┤                  │
│         ↓ (order.created)         │   │  │                  │
│ Execution Service (port 8087) ────┤───┤──┤ Binance API    │
│         ↓ (execution.filled)      │   │  │                  │
│ Portfolio Service (port 8088) ◄───┼───┼──┘                  │
│                                   │   │                     │
│ Event Broker (Kafka) ◄───────────┴───┴──────────────────────│
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚀 Implementation Order

**Recommended sequence** (dependencies matter):
1. **TASK-009 (Risk)** - Independent, can start immediately
2. **TASK-010 (Order)** - Depends on Risk Service (via events)
3. **TASK-011 (Execution)** ⭐ **CRITICAL** - Depends on Order (via events), interacts with Binance
4. **TASK-012 (Portfolio)** - Depends on Execution (via events)

**Parallel work**: TASK-009 and TASK-010 can be done in parallel if resources available.

---

## 🎯 Success Metrics

By end of Phase 2:
- ✅ Complete order lifecycle: Signal → Approval → Execution → Fill → Portfolio update
- ✅ Idempotency ensures no duplicate orders ever created
- ✅ Retry logic handles network issues gracefully
- ✅ Concurrent order processing works reliably
- ✅ Real-time P&L tracking and margin monitoring
- ✅ All 4 services deployed and tested together (integration)

**Critical Path**: Task-011 (Execution Service) - must work correctly for order matching to function.
