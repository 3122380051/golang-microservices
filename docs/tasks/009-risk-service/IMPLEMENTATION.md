# Task-009: Risk Service - Implementation Guide

## 📋 Overview

**Risk Service** acts as the guardian of trading risk. It evaluates strategy signals against predefined risk policies before allowing orders to proceed. This service sits between Strategy Service and Order Service in the trading pipeline.

### Core Responsibility
- **Evaluate**: Analyze signals against risk policies (position size, leverage, margin, daily loss)
- **Decide**: Approve or reject each signal with detailed reasoning
- **Publish**: Emit approved/rejected events to Kafka for downstream consumption
- **Cache**: Store decisions for audit and maintain portfolio snapshots

---

## 🏗️ Architecture & Design

### System Position
```
Market Data → Strategy Service → ⭐ RISK SERVICE ⭐ → Order Service → Execution Service
                                     (GATEKEEPER)
                                     
Flow:
1. Strategy generates signal (EMA crossover, RSI, etc.)
2. Risk Service receives it via Kafka: strategy.signal.generated
3. Risk Service evaluates against user's risk policy
4. Publishes: risk.order.approved OR risk.order.rejected
5. Order Service consumes approved events only
6. Execution Service places order on Binance
```

### Design Principles

#### 1. **Decision Tree Architecture**
Instead of a single monolithic check, risk evaluation follows a cascading decision tree:

```
Signal Received
    ↓
[1] Position Size Check
    - order_value = qty × price
    - if order_value > max_position_size → REJECT
    ↓
[2] Leverage Check
    - projected_leverage = (current_exposure + new_order) / equity
    - if projected_leverage > max_leverage → REJECT
    ↓
[3] Margin Check
    - margin_ratio = available_margin / total_balance
    - if margin_ratio < min_margin_ratio → REJECT
    ↓
[4] Daily Loss Check
    - if realized_pnl_today < max_daily_loss → REJECT (stop-out)
    ↓
[5] Exposure Check
    - total_exposure = current + new_order
    - if total_exposure > max_exposure → REJECT
    ↓
APPROVED ✅ → Emit risk.order.approved event
```

**Reasoning**: Early exit on first failure prevents unnecessary checks and provides specific rejection reason.

#### 2. **Layered Architecture**

```
┌─────────────────────────────────────────┐
│  Transport Layer (HTTP)                 │
│  └─ RiskHandler: REST endpoints         │
│     - GET/POST/PUT/DELETE policies      │
│     - GET decision details              │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│  Application Layer (Orchestration)      │
│  └─ Service: Coordinates flow           │
│     - Consumes Kafka signals            │
│     - Calls evaluator                   │
│     - Publishes decisions               │
│     - Manages caches                    │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│  Business Logic Layer (Core)            │
│  └─ Evaluator: Pure risk checks         │
│     - Position size validation          │
│     - Leverage calculation              │
│     - Margin ratio analysis             │
│     - Daily loss tracking               │
│     - Exposure limits                   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│  Infrastructure Layer (Persistence)     │
│  ├─ RiskPolicyRepository               │
│  │  └─ InMemoryRiskPolicyRepository    │
│  ├─ PortfolioCache                     │
│  └─ Kafka Producer/Consumer            │
└─────────────────────────────────────────┘
```

**Reasoning**: Separation of concerns enables testing and reuse; business logic (Evaluator) is pure Go with no I/O dependencies.

---

## 💡 Key Design Decisions & Reasoning

### Decision 1: **Kafka Event-Driven Architecture**

**Decision**: Risk Service consumes `strategy.signal.generated` and publishes `risk.order.approved`/`risk.order.rejected` events.

**Why Not Synchronous REST Calls?**
- ❌ REST coupling: Strategy Service would need to know risk endpoint; tight coupling
- ❌ Latency: HTTP roundtrip adds latency (typical: 50-100ms); Kafka batching is faster
- ❌ Resilience: HTTP timeout → request lost; Kafka has persistence and retry
- ✅ Event-driven: Natural fit for financial events; enables audit trail
- ✅ Scalability: Multiple Risk Services can consume same topic (horizontal scaling)

**Implementation**: 
```go
// Service.ConsumeStrategySignals runs in background goroutine
for {
    msg, _ := consumer.ReadMessage(ctx, timeout)
    signal := parseSignal(msg.Value)
    decision := evaluator.EvaluateSignal(...)
    publisher.PublishMessage(topic, decision.EventID, payload)
}
```

### Decision 2: **Idempotency via Signal ID**

**Decision**: Cache processed signal IDs to prevent duplicate evaluations.

**Why This Matters**:
- Kafka guarantees: "at-least-once" delivery (not exactly-once)
- If Risk Service crashes after evaluation but before committing Kafka offset → message re-delivered
- Without idempotency: Same signal evaluated twice → Two decisions published → Order Service confusion

**Implementation**:
```go
processedSignalsMu.Lock()
if processedSignals[signal.ID] {
    return // Skip already-processed signal
}
processedSignals[signal.ID] = true
processedSignalsMu.Unlock()
```

### Decision 3: **Portfolio Snapshot Model**

**Decision**: Fetch portfolio state from cache (populated by Portfolio Service) instead of querying DB directly.

**Why**:
- 🚀 Performance: Cache hit → microseconds vs. DB query → milliseconds
- 📊 Consistency: Portfolio Service is source-of-truth; we read its published cache
- 🔄 Loose coupling: Risk doesn't depend on Portfolio DB schema
- 📝 Phase progression: MVP uses in-memory cache; Phase 2 → Poll Portfolio Service API

**Trade-off**: Cache may be slightly stale (last update time in snapshot); acceptable because:
- Risk policy checks use conservative thresholds (min 50% margin, max 5x leverage)
- 5-10 second staleness << user reaction time (humans trade on hours/minutes)

### Decision 4: **Policy Hierarchy: User+Strategy → Default**

**Decision**: 
```go
1. Try: GetByUserAndStrategy(userID, strategyID)  // Specific
2. Fallback: GetDefaultPolicy()                    // Global
3. Fail: Error                                     // No policy = reject
```

**Why**:
- Flexibility: Power users can define custom policies per strategy
- Safety: Default policy ensures even new strategies get risk checks
- Simplicity: MVP uses in-memory storage; scales to DB in Phase 2

### Decision 5: **Thread-Safe Repository with RWMutex**

**Decision**: Use `sync.RWMutex` for concurrent access to policy cache.

**Why**:
- Multiple Kafka consumers (goroutines) read policies simultaneously
- Occasional writes (admin updates policies)
- RWMutex: Allows concurrent reads, exclusive writes
- Simpler than channels; suitable for in-memory cache

```go
// Many goroutines can do this in parallel:
r.mu.RLock()
defer r.mu.RUnlock()
policy := r.policies[id]

// Admin does exclusive update:
r.mu.Lock()
defer r.mu.Unlock()
r.policies[id] = newPolicy
```

---

## 📐 Data Structures

### RiskPolicy
Defines constraints for a user/strategy:
```go
type RiskPolicy struct {
    ID              string    // UUID
    UserID          string    // Who this policy applies to
    StrategyID      string    // Which strategy ("*" = all)
    MaxPositionSize float64   // USD per order
    MaxLeverage     float64   // e.g., 5.0 = 5x
    MaxDailyLoss    float64   // e.g., -1000 = stop-loss at -$1000
    MinMarginRatio  float64   // e.g., 0.5 = keep 50% available
    MaxExposure     float64   // Total exposure limit (USD)
    IsActive        bool      // Soft delete
}
```

### RiskDecision
Records outcome of evaluation:
```go
type RiskDecision struct {
    ID              string          // UUID
    SignalID        string          // Which signal was evaluated
    IsApproved      bool            // Passed all checks?
    RejectionReason string          // If rejected, why?
    Checks          ChecksDetail    // Detailed breakdown
    TraceID         string          // Correlation ID for logging
}

type ChecksDetail struct {
    PositionSizeCheck CheckResult
    LeverageCheck     CheckResult
    MarginCheck       CheckResult
    DailyLossCheck    CheckResult
    ExposureCheck     CheckResult
}

type CheckResult struct {
    Passed bool   // true/false
    Reason string // "Position OK: $50k <= $100k"
    Value  string // "$50k / $100k"
}
```

**Why This Structure**:
- ✅ Auditability: Every decision is recorded with full reasoning
- ✅ Debuggability: Traders see exactly which check failed
- ✅ Pattern matching: Platform can analyze failure patterns

---

## 🔄 Complete Data Flow

### Scenario: User buys 0.5 BTC when position size already at $50k

**Input State**:
```
Strategy Signal:  BUY 0.5 BTC at BTCUSDT (estimated $66k)
User Portfolio:
  - Balance: $100k
  - Current Exposure: $50k
  - Available Margin: $40k
User Policy:
  - MaxPositionSize: $100k
  - MaxLeverage: 5.0
  - MinMarginRatio: 50%
  - MaxDailyLoss: -$1000
  - MaxExposure: $300k
```

**Execution Flow**:

```
1. Kafka Consumer Thread
   ├─ Receives: strategy.signal.generated
   │  └─ { id: "sig-123", signal_id: "sig-123", symbol: "BTCUSDT", 
   │       side: "BUY", quantity: 0.5, ... }
   │
   ├─ Check Idempotency
   │  └─ processedSignals["sig-123"] == false ✅ (first time)
   │  └─ Mark as processed: processedSignals["sig-123"] = true
   │
   └─ Process Signal
      ├─ Get Policy
      │  └─ GetByUserAndStrategy("user-123", "ema-cross")
      │  └─ Success: Returns MaxPositionSize=$100k, MaxLeverage=5.0, ...
      │
      ├─ Get Portfolio
      │  └─ portfolioCache.GetSnapshot("user-123")
      │  └─ Success: Current Exposure=$50k, Available=$40k, ...
      │
      ├─ Evaluate Signal
      │  │
      │  ├─[1] Position Size Check
      │  │   └─ orderValue = 0.5 × $66k = $33k
      │  │   └─ $33k <= $100k ✅ PASS
      │  │
      │  ├─[2] Leverage Check
      │  │   └─ newExposure = $50k + $33k = $83k
      │  │   └─ projectedLeverage = $83k / $100k = 0.83x
      │  │   └─ 0.83x <= 5.0x ✅ PASS
      │  │
      │  ├─[3] Margin Check
      │  │   └─ marginRatio = $40k / $100k = 40%
      │  │   └─ 40% < 50% ❌ FAIL
      │  │   └─ Reason: "margin ratio 40% below minimum 50%"
      │  │
      │  ├─[4] Daily Loss Check (skipped - already failed)
      │  ├─[5] Exposure Check (skipped - already failed)
      │  │
      │  └─ Result: REJECTED ❌
      │
      ├─ Create RiskDecision
      │  ├─ ID: "risk-dec-456"
      │  ├─ IsApproved: false
      │  ├─ RejectionReason: "margin ratio 40% below minimum 50%"
      │  ├─ TraceID: "trace-789"
      │  └─ Checks: [✅, ✅, ❌, -, -]
      │
      ├─ Cache Decision
      │  └─ decisionsCache["risk-dec-456"] = decision
      │
      ├─ Publish Event
      │  ├─ Topic: "risk.order.rejected" (auto-selected)
      │  ├─ Key: "risk-dec-456"
      │  ├─ Payload: 
      │  │  {
      │  │    event_id: "risk-dec-456",
      │  │    signal_id: "sig-123",
      │  │    user_id: "user-123",
      │  │    symbol: "BTCUSDT",
      │  │    is_approved: false,
      │  │    rejection_reason: "margin ratio 40% below minimum 50%",
      │  │    trace_id: "trace-789",
      │  │    timestamp: "2024-05-18T10:15:30Z"
      │  │  }
      │  └─ Success ✅
      │
      └─ Log Decision
         └─ Logger.Info("risk decision published", 
                decision_id="risk-dec-456",
                approved=false,
                reason="margin ratio 40% below minimum 50%")

2. Order Service (Consumer)
   └─ Reads: risk.order.rejected
      └─ Discards (doesn't create order)

3. Result
   └─ User sees: "Order rejected: Insufficient margin (40% available < 50% required)"
```

---

## 🧪 Testing Strategy

### Unit Tests
Located: `tests/risk_service_test.go`

#### Test Coverage:
1. **PositionSizeCheck**: Valid and over-limit orders
2. **LeverageCheck**: Projected leverage calculation
3. **MarginCheck**: Margin ratio validation
4. **DailyLossCheck**: Daily loss limit enforcement
5. **RepositoryCRUD**: Policy creation, retrieval, update, delete
6. **CacheOperations**: Snapshot caching and invalidation

#### Test Pattern:
```go
func TestRiskEvaluator_PositionSizeCheck(t *testing.T) {
    // Setup
    logger := infrastructure.NewLogger(&infrastructure.Config{})
    evaluator := risk.NewEvaluator(logger)
    policy := createTestPolicy()
    signal := createTestSignal()
    portfolio := createTestPortfolio()
    
    // Act
    decision := evaluator.EvaluateSignal(policy, portfolio, signal, 66100.0)
    
    // Assert
    assert.True(t, decision.Checks.PositionSizeCheck.Passed)
    assert.True(t, decision.IsApproved)
}
```

### Integration Testing (Phase 2)
- Mock Kafka broker
- Test Kafka consumer flow
- Test event publishing

### Manual Testing Commands
```bash
# 1. Start Risk Service
go run cmd/risk-service/main.go

# 2. Create a risk policy
curl -X POST http://localhost:8085/risk/policies \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user1",
    "strategy_id": "ema-cross",
    "max_position_size": 100000,
    "max_leverage": 5,
    "max_daily_loss": -1000,
    "min_margin_ratio": 0.5,
    "max_exposure": 500000
  }'

# 3. List policies
curl http://localhost:8085/risk/policies?user_id=user1

# 4. Get decision details
curl http://localhost:8085/risk/decisions/{decision_id}

# 5. Run tests
go test ./tests -v
```

---

## 🚀 Integration Points

### 1. **Kafka Topics**
| Topic | Direction | Purpose |
|-------|-----------|---------|
| `strategy.signal.generated` | ← Consume | Receive strategy signals |
| `risk.order.approved` | → Publish | Signal passed all checks |
| `risk.order.rejected` | → Publish | Signal failed a check |

### 2. **Service Dependencies**
| Service | Interaction | When |
|---------|-------------|------|
| Strategy Service | Producer of signals | Real-time |
| Portfolio Service | Source of portfolio state | On-demand (cache) |
| Order Service | Consumer of decisions | Real-time |

### 3. **Database Schema** (Phase 2)
```sql
CREATE TABLE risk_policies (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    strategy_id TEXT NOT NULL,
    max_position_size DECIMAL(18,2),
    max_leverage DECIMAL(10,2),
    max_daily_loss DECIMAL(18,2),
    min_margin_ratio DECIMAL(5,4),
    max_exposure DECIMAL(18,2),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, strategy_id)
);

CREATE TABLE risk_decisions (
    id UUID PRIMARY KEY,
    signal_id UUID NOT NULL,
    user_id UUID NOT NULL,
    is_approved BOOLEAN NOT NULL,
    rejection_reason TEXT,
    checks_json JSONB,
    trace_id TEXT,
    decided_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 📋 Checklist

- [x] Domain models (RiskPolicy, RiskDecision, PortfolioSnapshot)
- [x] Evaluator with 5 risk checks
- [x] Service with Kafka consumer loop
- [x] In-memory repository
- [x] Portfolio cache
- [x] HTTP handlers (CRUD endpoints)
- [x] Unit tests
- [x] Main entry point with graceful shutdown
- [x] Error handling and logging
- [x] Idempotency mechanism

---

## 🔮 Phase 2 Enhancements

1. **Database Persistence**: Move policies from in-memory to PostgreSQL
2. **Real Portfolio API**: Poll Portfolio Service API instead of local cache
3. **Performance Optimization**: Add Redis for distributed caching
4. **Advanced Checks**: ML-based anomaly detection for unusual orders
5. **Webhook Notifications**: Alert users immediately on rejection
6. **Audit Events**: Track all policy changes with who/when/why
7. **Dynamic Thresholds**: Adjust limits based on market volatility

---

## 📝 Summary

**Risk Service** implements a multi-stage evaluation pipeline that protects users from overleveraging, position-size violations, and other trading risks. By using Kafka events and a decision tree architecture, it integrates seamlessly into the event-driven pipeline while remaining horizontally scalable and maintainable.

**Key Strengths**:
- ✅ Deterministic: Same inputs always produce same decision
- ✅ Observable: Detailed logging and trace IDs for debugging
- ✅ Scalable: Stateless design; can run multiple instances
- ✅ Safe: Fails closed (rejects on error) following defense-in-depth
- ✅ Auditable: Every decision is recorded with full reasoning
