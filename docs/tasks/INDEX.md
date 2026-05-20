# Task Management - Directory Index

## Tổng quan
Các task được tổ chức thành 8 giai đoạn, mỗi giai đoạn có thư mục riêng chứa SRS, PRD, acceptance criteria, deliverables.

## AI Workflow
- [AI Workflow](AI_WORKFLOW.md) - Quy trình làm từng task theo thứ tự, rule chuyển trạng thái, acceptance gate.

## Danh sách Task

### Giai đoạn 1: Hạ tầng cơ bản (Tuần 1-2)

- **[TASK-001: Setup Project Structure](001-setup-project-structure/)**
  - Folder structure, Go modules, Docker Compose, common packages
  - Effort: 4h | Timeline: Ngày 1 sáng

- **[TASK-002: Database Schema](002-database-schema/)**
  - PostgreSQL setup, migrations, indexes, seed data
  - Effort: 4h | Timeline: Ngày 1 chiều

- **[TASK-003: Message Broker](003-message-broker/)**
  - Kafka/NATS setup, topics, producer/consumer
  - Effort: 3h | Timeline: Ngày 2 sáng

- **[TASK-004: API Gateway](004-api-gateway/)**
  - HTTP server, middleware, routing, Swagger
  - Effort: 6h | Timeline: Ngày 2-3

### Giai đoạn 2: Xác thực & Người dùng (Tuần 2-3)

- **[TASK-005: Auth Service](005-auth-service/)**
  - Register, login, JWT, refresh token, RBAC
  - Effort: 6h | Timeline: Ngày 3-4

- **[TASK-006: User Service](006-user-service/)**
  - Profile management, API key encryption, audit log
  - Effort: 5h | Timeline: Ngày 4 chiều

### Giai đoạn 3: Dữ liệu thị trường (Tuần 3-4)

- **[TASK-007: Market Data Service](007-market-data-service/)**
  - Exchange adapter, Binance REST/WebSocket, cache, events
  - Effort: 8h | Timeline: Ngày 5-6

### Giai đoạn 4: Chiến lược (Tuần 4-5)

- **[TASK-008: Strategy Service](008-strategy-service/)**
  - EMA cross, RSI strategies, signal generation
  - Effort: 6h | Timeline: Ngày 6-7

### Giai đoạn 5: Trading Core - Matching & Execution (Tuần 5-6)

- **[TASK-009: Risk Service](009-risk-service/)**
  - Risk policy evaluation, position size/leverage/margin checks
  - Effort: 6h | Timeline: Ngày 8-9

- **[TASK-010: Order Service](010-order-service/)**
  - Order lifecycle management, state machine, idempotency
  - Effort: 7h | Timeline: Ngày 9-10

- **[TASK-011: Execution Service](011-execution-service/)** ⭐ **CRITICAL**
  - Send orders to exchange, retry logic, reconciliation, matching via Binance
  - Effort: 8h | Timeline: Ngày 11-12

- **[TASK-012: Portfolio Service](012-portfolio-service/)**
  - Position tracking, PnL calculation, margin management
  - Effort: 6h | Timeline: Ngày 12-13

### Giai đoạn 6+: Tiếp tục
- [ ] [TASK-020: Futures Trading Service](020-futures-trading-service/)
- Notification Service, Audit Log Service, Exchange Adapters (Bybit, OKX)
- Testing, Integration, Deployment, Monitoring

## Cách sử dụng

### Cho từng engineer
1. Đọc `README.md` trong task folder của bạn.
2. Xem SRS section để hiểu requirements.
3. Xem PRD section để hiểu acceptance criteria.
4. Theo dõi Deliverables checklist.
5. Implement trong folder structure được định sẵn.

### Cho team lead
1. Track progress qua checklist trong mỗi task README.
2. Code review theo deliverable list.
3. Run acceptance test theo PRD.
4. Mark task done khi tất cả ✅ hoàn thành.

## Status Overview

| Task | Status | Owner | Effort | Timeline | Port |
|------|--------|-------|--------|----------|------|
| [001 - Setup](001-setup-project-structure/) | ✅ COMPLETED | Copilot | 4h | Day 1 | - |
| [002 - Database](002-database-schema/) | ✅ COMPLETED | Copilot | 4h | Day 1 | - |
| [003 - Broker](003-message-broker/) | ✅ COMPLETED | Copilot | 3h | Day 2 | - |
| [004 - Gateway](004-api-gateway/) | ✅ COMPLETED | Copilot | 6h | Day 2-3 | 8080 |
| [005 - Auth](005-auth-service/) | ✅ COMPLETED | Copilot | 6h | Day 3-4 | 8081 |
| [006 - User](006-user-service/) | ✅ COMPLETED | Copilot | 5h | Day 4 | 8082 |
| [007 - Market Data](007-market-data-service/) | ✅ COMPLETED | Copilot | 8h | Day 5-6 | 8083 |
| [008 - Strategy](008-strategy-service/) | ✅ COMPLETED | Copilot | 6h | Day 6-7 | 8084 |
| [009 - Risk](009-risk-service/) | ✅ COMPLETED | Backend 6 | 6h | Day 8-9 | 8085 |
| [010 - Order](010-order-service/) | ✅ COMPLETED | Backend 7 | 7h | Day 9-10 | 8086 |
| [011 - Execution](011-execution-service/) | ✅ COMPLETED | Backend 8 | 8h | Day 11-12 | 8087 |
| [012 - Portfolio](012-portfolio-service/) | ✅ COMPLETED | Backend 9 | 6h | Day 12-13 | 8088 |

## 📊 Completion Summary

**Phase 1 (Infrastructure + Market/Strategy): 8/8 ✅ COMPLETE** (42 hours)  
**Phase 2 (Risk + Order + Execution + Portfolio): 4/4 ✅ COMPLETE** (27 hours)  
**Phase 3+: Additional Services** (TBD)

### Phase 1 Status: ✅ COMPLETE

#### Infrastructure Layer ✅
- [x] Go 1.22.0 project with modular structure
- [x] PostgreSQL 16 with 8 migration files
- [x] Kafka 3.x with 8 topics configured
- [x] API Gateway (port 8080) with middleware stack

#### Authentication & User Management ✅
- [x] Auth Service (port 8081): JWT, refresh tokens, RBAC
- [x] User Service (port 8082): Profile, API key encryption, audit logs

#### Market Data & Trading Core ✅
- [x] Market Data Service (port 8083): Real-time Binance integration, WebSocket streaming
- [x] Strategy Service (port 8084): EMA cross, RSI engines, signal generation

#### Testing & Deployment ✅
- [x] Unit tests for all services
- [x] Integration tests for broker, market, strategy
- [x] Docker Compose with all 8 services
- [x] Multi-stage Dockerfile for each service

### Phase 2 Status: ✅ COMPLETE

#### Trading Core - Order Matching ⏳
- [x] Risk Service (port 8085): Risk policy evaluation, approval/rejection
- [x] Order Service (port 8086): Order lifecycle, state machine, idempotency
- [x] **Execution Service (port 8087)**: **Order submission, retry logic, Binance matching, reconciliation** ⭐ **CRITICAL**
- [x] Portfolio Service (port 8088): Position tracking, PnL calculation, margin management

#### Event Flow (Phase 2 complete)
```
Market Data → Strategy Signal → Risk Check → Order Create → Execution → Binance Match → Portfolio Update
```

## Next Phase Tasks (Planned)
- **TASK-009**: Risk Service (6h) ✅ Complete
- **TASK-010**: Order Service (7h) ✅ Complete
- **TASK-011**: Execution Service (8h) ⭐ Critical for order matching, complete
- **TASK-012**: Portfolio Service (6h) ✅ Complete

## Quick Links
- [Architecture Overview](../architecture-tong-the.md)
- [Full Task Breakdown](../task-breakdown.md)
- [Project Structure Guide](../project-structure.md)
- [SRS - Software Requirements](../srs-software-requirements.md)
- [PRD - Product Requirements](../prd-product-requirements.md)
