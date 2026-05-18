# Task-010: Order Service - Implementation Guide

## 📋 Overview

**Order Service** manages the complete order lifecycle from creation through settlement. It acts as the intermediate layer between Risk Service (which approves signals) and Execution Service (which places orders on the exchange).

### Core Responsibility
- **Create**: Convert approved risk decisions into internal orders
- **Track**: Maintain order state through its lifecycle (created → submitted → filled → closed)
- **Validate**: Enforce state machine rules and cancellation policies
- **Correlate**: Link orders to signals and risk decisions for end-to-end traceability
- **Publish**: Emit order events for downstream services

---

## 🏗️ Architecture & Design

### System Position
```
Risk Service → ⭐ ORDER SERVICE ⭐ → Execution Service
              (ORCHESTRATOR)           (Binance)
              
Flow:
1. Risk Service approves signal → publishes risk.order.approved
2. Order Service consumes it → creates internal Order
3. Order Service transitions to "submitted" state
4. Execution Service consumes order.created → submits to Binance
5. Binance fills order → Execution Service reports fill
6. Order Service updates state to "filled"
7. Portfolio Service tracks position
```

### Design Principles

#### 1. **Order State Machine**

Orders follow a strict state machine with no backward transitions:

```
                ┌─→ FILLED ──→ CLOSED
                │
CREATED ─→ SUBMITTED ├─→ PARTIAL_FILLED ──→ FILLED
                │
                ├─→ CANCELED (terminal)
                │
                └─→ REJECTED (terminal)
```

**Key Rules**:
- Created → Submitted: Only when about to send to exchange
- Submitted → Filled: When order matches completely
- Submitted → Partial_Filled: When partial fill received
- Partial_Filled → Filled: When remaining quantity fills
- Any state → Canceled: User cancels before settled (only Created/Submitted)
- Any state → Rejected: System rejects (e.g., invalid params)
- Terminal states (Filled, Canceled, Rejected, Closed) have no outbound transitions

**Reasoning**: Strict state machine prevents invalid order states and race conditions. No backward transitions = order history is immutable.

#### 2. **Idempotency via ClientOrderID**

**Problem**: 
- If Order Service crashes after creating order but before committing to DB:
  - Next restart → reads same Kafka message again
  - Would create duplicate order

**Solution**:
```go
// Request creates order with client_order_id (provided by caller or generated)
// If same client_order_id seen twice within 24h → return existing order
type Order {
    ID              string  // Internal UUID
    ClientOrderID   string  // Idempotency key (stays constant across retries)
    ...
}

// Repository level idempotency:
func (r *InMemoryOrderRepository) Create(order *Order) error {
    if existing := r.ordersByClientID[order.ClientOrderID]; existing != nil {
        // Copy existing order data to new order object (idempotent)
        *order = *existing
        return nil
    }
    r.ordersById[order.ID] = order
    r.ordersByClientID[order.ClientOrderID] = order
    return nil
}
```

**Why This Works**:
- ClientOrderID is deterministic (generated once when order created)
- Multiple create requests with same clientOrderID return same order
- Prevents duplicate orders at source

#### 3. **Correlation ID for Tracing**

**Problem**: How to trace order through the system?

**Solution**: Every order has a CorrelationID:
```
Signal: signal-001 (user's strategy generated)
    ↓
Risk Decision: risk-dec-001 (traced via TraceID)
    ↓
Order: order-001 (CorrelationID = trace-789)
    ↓
Execution: exec-001 (includes CorrelationID)
    ↓
Portfolio: position updated (tagged with CorrelationID)

Later: Query CorrelationID = trace-789 → get full order history
```

**Implementation**: CorrelationID is set once at order creation and propagated to all downstream events.

#### 4. **Layered Architecture**

```
┌──────────────────────────────┐
│  Transport Layer (HTTP)      │
│  └─ OrderHandler             │
│     - POST /orders (create)  │
│     - GET /orders/{id}       │
│     - DELETE /orders/{id}    │
│       (cancel)               │
└──────────────┬───────────────┘
               │
┌──────────────▼───────────────────┐
│  Application Layer (State Mgmt)   │
│  └─ Service                       │
│     - ConsumeRiskDecisions()      │
│     - CreateOrder()               │
│     - CancelOrder()               │
│     - UpdateOrderFill()           │
│     - Publishes order.* events    │
└──────────────┬───────────────────┘
               │
┌──────────────▼──────────────────┐
│  Business Logic Layer            │
│  ├─ StateMachine                 │
│  │  └─ Transition() / Validate() │
│  └─ Validator                    │
│     └─ ValidateOrderCreation()   │
│     └─ ValidateCancellation()    │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│  Infrastructure Layer            │
│  └─ OrderRepository              │
│     └─ InMemoryOrderRepository   │
│     - CRUD operations            │
│     - Idempotency via clientID   │
└──────────────────────────────────┘
```

**Reasoning**: 
- StateMachine is pure logic (testable, no I/O)
- Validator is pure logic (testable, no I/O)  
- Service orchestrates: consumes Kafka, calls logic, publishes events
- Repository abstracts persistence (easily swap in-memory for DB)

#### 5. **Idempotency at Multiple Levels**

**Level 1: Kafka Consumer**
```go
// Track processed risk decisions
processedRiskDecisions: map[string]bool
if processedRiskDecisions[riskDecisionID] {
    return // Skip duplicate
}
```

**Level 2: Repository**
```go
// Handle duplicate CreateOrder calls
if existing := repo.ordersByClientID[clientOrderID]; existing != nil {
    return existing // Idempotent
}
```

**Why Two Levels?**:
- Kafka-level prevents duplicate processing
- Repository-level provides failover idempotency (if Kafka tracking lost)

---

## 💡 Key Design Decisions & Reasoning

### Decision 1: **State Machine over Direct State Updates**

**Decision**: Use StateMachine.Transition() instead of directly setting `order.Status = newStatus`

**Why**:
- ❌ Direct update: Too easy to create invalid states (e.g., Filled → Created)
- ✅ StateMachine: Validates transition before applying
- ✅ Centralized logic: All state rules in one place
- ✅ Testable: Can test state transitions independently

```go
// ❌ Bad:
order.Status = domain.OrderStatusFilled

// ✅ Good:
sm.Transition(order, domain.OrderStatusFilled, "")
```

### Decision 2: **Separate Validator Module**

**Decision**: Extract validation logic into Validator type instead of inline in Service

**Why**:
- ✅ Testability: Can test validation rules without Service
- ✅ Reusability: Can use Validator in multiple services
- ✅ Clarity: Validation logic is explicit and documented
- ✅ Maintenance: Changes to rules only in one place

```go
// Validator methods:
- ValidateOrderCreation()
- ValidateCancellation()
- ValidateFillUpdate()
- ValidateOrderState()
```

### Decision 3: **Kafka Event-Driven instead of Direct API Calls**

**Decision**: Order Service consumes risk.order.approved/rejected via Kafka instead of REST endpoints

**Why**:
- ❌ REST coupling: Risk Service URL hardcoded; tight coupling
- ❌ Latency: HTTP roundtrips add latency
- ❌ No persistence: Request lost if network fails
- ✅ Kafka: Persistent message queue; can replay
- ✅ Decoupling: Risk Service and Order Service don't need to know each other
- ✅ Scalability: Can run multiple Order Services consuming same topic

### Decision 4: **In-Memory Repository for MVP**

**Decision**: Use InMemoryOrderRepository for Phase 1; migrate to PostgreSQL in Phase 2

**Why MVP**:
- 🚀 Fast iteration: No DB schema management
- 🧪 Easy testing: Mock data easily
- 🔧 Simpler debugging: In-memory state visible in debugger

**Phase 2 Migration**:
- Add PostgreSQL migration file
- Create PostgresOrderRepository
- Swap implementation via dependency injection
- No logic layer changes needed

### Decision 5: **CorrelationID Set at Order Creation**

**Decision**: Generate CorrelationID once in CreateOrder(); never changes

**Why**:
- ✅ Immutable tracing: Same ID through entire lifecycle
- ✅ Predictable: Logs/events always reference same ID
- ✅ Debugging: Can grep logs for CorrelationID
- ✅ Analytics: Can aggregate events by CorrelationID

---

## 📐 Data Structures

### Order State Machine Diagram
```
            Create Order
                 ↓
         [CREATED Status]
           /    |    \
    Submit /     |     \ Cancel / Reject
         /       |       \
    [SUBMITTED] | 
    /    |  \   |
Fill /    |   \ Partial / Reject
   /     |     \
[FILLED] | [PARTIAL_FILLED]
         |    /
        Close or Reject
         ↓
    [Terminal]
```

### Key Fields

**Order.ID**: Internal UUID (never shown to user)
- Generated at order creation
- Immutable
- Used for database lookups

**Order.ClientOrderID**: Idempotency key
- Can be provided by caller or generated by service
- Unique per user
- Stays same across retries
- Used for duplicate detection

**Order.CorrelationID**: Tracing identifier
- Generated at order creation
- Immutable
- Propagated to all downstream events
- Used for end-to-end logging

**Order.Status**: Lifecycle state
- Follows state machine rules
- Only valid transitions allowed
- Terminal states have no outbound transitions

### Event Structure

```go
type OrderEvent {
    EventID        string   // UUID
    EventType      string   // "order.created", "order.canceled", etc.
    CorrelationID  string   // For tracing
    OrderID        string
    ClientOrderID  string
    UserID         string
    Symbol         string
    Status         string
    ExecutedQty    float64
    Fees           float64
    Timestamp      time.Time
}

// Kafka Topics:
- "order.created"    → Execution Service consumes
- "order.canceled"   → Audit/notifications
- "order.updated"    → Portfolio Service updates
```

---

## 🔄 Complete Data Flow

### Scenario: User's EMA strategy generates BUY signal

**Timeline**:

```
T0: Strategy Service detects EMA crossover
    ↓
    Publishes: strategy.signal.generated {id: sig-001, symbol: BTCUSDT, qty: 0.5}
    
T1: Risk Service receives signal (Kafka consumer)
    ↓
    Evaluates: Position size? ✅ Leverage? ✅ Margin? ✅ Daily loss? ✅
    ↓
    Publishes: risk.order.approved {event_id: risk-dec-001, signal_id: sig-001, user_id: user-123}

T2: Order Service receives risk decision (Kafka consumer)
    ↓
    Check idempotency: processedRiskDecisions[risk-dec-001]? No → proceed
    ↓
    CreateOrder():
        - ID: "order-uuid-789"
        - ClientOrderID: "client-uuid-abc" (idempotency key)
        - CorrelationID: "trace-xyz" (for tracing)
        - Status: CREATED
        - SignalID: sig-001
        - RiskDecisionID: risk-dec-001
    ↓
    Repository.Create(order):
        - Store by ID: ordersById["order-uuid-789"] = order
        - Store by ClientID: ordersByClientID["client-uuid-abc"] = order
    ↓
    Publishes: order.created {order_id: order-uuid-789, correlation_id: trace-xyz, status: CREATED}

T3: Execution Service receives order.created (Kafka consumer)
    ↓
    Submits to Binance:
        symbol: BTCUSDT
        side: BUY
        quantity: 0.5
        newClientOrderId: client-uuid-abc (passes our clientOrderID to Binance)
    ↓
    Binance returns: {orderId: 12345, clientOrderId: client-uuid-abc, status: FILLED}
    ↓
    Publishes: execution.filled {order_id: order-uuid-789, binance_order_id: 12345, executed_qty: 0.5, avg_price: 66000}

T4: Order Service receives execution fill (from Execution Service)
    ↓
    UpdateOrderFill():
        - ExecutedQuantity: 0.5
        - AverageFillPrice: 66000
        - Status transition: SUBMITTED → FILLED
    ↓
    Repository.Update(order)
    ↓
    Publishes: order.updated {order_id: order-uuid-789, status: FILLED, executed_qty: 0.5}

T5: Portfolio Service receives order.updated
    ↓
    Creates position: BTCUSDT, qty=0.5, entry_price=66000
    ↓
    Publishes: portfolio.updated {user_id: user-123, symbol: BTCUSDT, position: 0.5}

T6: All services in sync
    - Strategy: Signal executed ✅
    - Risk: Decision recorded ✅
    - Order: Filled ✅
    - Execution: Binance order complete ✅
    - Portfolio: Position tracked ✅
    - Trace: All tagged with correlation_id=trace-xyz ✅
```

---

## 🧪 Testing Strategy

### Test Coverage: 20+ tests

**StateMachine Tests** (5 tests):
- Valid transition: Created → Submitted
- Valid transition: Submitted → Filled
- Invalid transition: Filled → Submitted (rejected)
- AllowedTransitions() for each state
- DescribeTransition() human-readable descriptions

**Validator Tests** (8 tests):
- Valid order creation
- Missing required fields (userID, symbol, etc.)
- Invalid quantity (zero)
- Limit order requires price
- Can cancel: Created/Submitted
- Cannot cancel: Filled/Canceled/Rejected
- Fill update validation (qty, price, fees)

**Repository Tests** (6 tests):
- Create with idempotency (same clientID → existing order)
- GetByID, GetByClientOrderID
- ListByUser (with optional status filter)
- Update order state
- Delete order
- Get deleted order → error

**Order Domain Tests** (2 tests):
- CanTransition() for all valid/invalid paths
- IsFinal() for terminal vs. non-terminal states

**Pattern**: AAA (Arrange-Act-Assert)
```go
func TestStateMachine_ValidTransition_CreatedToSubmitted(t *testing.T) {
    // Arrange
    sm := order.NewStateMachine()
    ord := &domain.Order{ID: "order-1", Status: domain.OrderStatusCreated}
    
    // Act
    err := sm.Transition(ord, domain.OrderStatusSubmitted, "")
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, domain.OrderStatusSubmitted, ord.Status)
}
```

---

## 📋 Integration Points

### Kafka Topics

| Topic | Direction | Producer | Consumer | Purpose |
|-------|-----------|----------|----------|---------|
| `risk.order.approved` | ← | Risk Service | Order Service | Create internal order |
| `risk.order.rejected` | ← | Risk Service | Order Service | Log rejection, don't create |
| `order.created` | → | Order Service | Execution Service | Submit to exchange |
| `order.updated` | → | Order Service | Portfolio Service | Update positions |
| `order.canceled` | → | Order Service | Audit/Notifications | Log cancellation |

### Service Dependencies

| Service | Interaction | When |
|---------|-------------|------|
| Risk Service | Producer of risk decisions | On signal evaluation |
| Execution Service | Consumer of order.created | Real-time order submission |
| Portfolio Service | Consumer of order.updated | Position tracking |

### HTTP Endpoints

```
POST /orders
    Body: {user_id, strategy_id, symbol, side, quantity, signal_id, risk_decision_id}
    Response: {id, client_order_id, correlation_id, status, created_at}
    
GET /orders/{order_id}
    Response: Full order detail + fills + timestamps
    
GET /orders?user_id=X&status=Y
    Response: List of orders with filters
    
DELETE /orders/{order_id}?reason=X
    Response: {id, status, reason}
    Error 400: If order in terminal state (cannot cancel filled)
    
PUT /orders/{order_id}/fill
    Body: {executed_quantity, average_fill_price, fees, status}
    Response: Updated order with new fill data
```

---

## 🚀 Integration with Trading Pipeline

```
Task-007: Market Data Service
         ↓
Task-008: Strategy Service
    (generates: strategy.signal.generated)
         ↓
Task-009: Risk Service
    (generates: risk.order.approved/rejected)
         ↓
★ Task-010: Order Service ★ (YOU ARE HERE)
    (generates: order.created/updated/canceled)
         ↓
Task-011: Execution Service (NEXT)
    (submits to Binance, generates: execution.filled)
         ↓
Task-012: Portfolio Service
    (updates positions, calculates PnL)
```

---

## 📊 State Transition Validation

All transitions are validated before applying:

```go
func CanTransition(current, requested OrderStatus) bool {
    switch current {
    case CREATED:
        return requested ∈ {SUBMITTED, CANCELED, REJECTED}
    case SUBMITTED:
        return requested ∈ {FILLED, PARTIAL_FILLED, CANCELED, REJECTED}
    case PARTIAL_FILLED:
        return requested ∈ {FILLED, CANCELED}
    case FILLED, CANCELED, REJECTED, CLOSED:
        return false // Terminal states
    }
}
```

---

## 🔮 Phase 2 Enhancements

1. **PostgreSQL Persistence**: Migrate from in-memory to durable storage
2. **Order Amendment**: Allow price/quantity changes for unsubmitted orders
3. **Batch Orders**: Support creating multiple orders in single request
4. **Order History**: Track all state transitions with timestamps
5. **Advanced Queries**: Filter by date range, symbol, status, PnL
6. **Webhook Notifications**: Alert users on order status changes
7. **Audit Trail**: Log who made what changes and when

---

## ✅ Checklist

- [x] Domain models (Order, OrderStatus, OrderSide, etc.)
- [x] State Machine (transitions + validation)
- [x] Validator (creation, cancellation, fill updates)
- [x] Service layer (Kafka consumer + orchestration)
- [x] In-memory repository (CRUD + idempotency)
- [x] HTTP handlers (endpoints)
- [x] Unit tests (20+ tests, all passing)
- [x] Main entry point (service bootstrap)
- [x] Event publishing (order.created, order.updated, order.canceled)
- [x] IMPLEMENTATION.md (this document)

---

## 📝 Summary

**Order Service** provides the foundational order management layer, sitting between Risk Service (policy enforcement) and Execution Service (exchange interaction). By implementing a strict state machine and idempotency via ClientOrderID, it ensures orders are never duplicated and their lifecycle is always valid.

**Key Strengths**:
- ✅ Deterministic state transitions
- ✅ Idempotent: Same requests always produce same result
- ✅ Traceable: CorrelationID enables end-to-end logging
- ✅ Safe: Terminal states prevent invalid operations
- ✅ Scalable: In-memory MVP easily migrates to PostgreSQL
- ✅ Well-tested: 20+ comprehensive unit tests
