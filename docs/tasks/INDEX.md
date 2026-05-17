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

### Giai đoạn 5-8: Tiếp tục
- Risk Service, Order Service, Execution Service, Exchange Adapter
- Portfolio Service, Notification Service, Audit Log Service
- Testing, Integration, Deployment

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

| Task | Status | Owner | Effort | Timeline |
|------|--------|-------|--------|----------|
| 001 - Setup | ✅ Done | Copilot | 4h | Day 1 |
| 002 - Database | ✅ Done | Copilot | 4h | Day 1 |
| 003 - Broker | ⏳ To Do | - | 3h | Day 2 |
| 004 - Gateway | ⏳ To Do | - | 6h | Day 2-3 |
| 005 - Auth | ⏳ To Do | - | 6h | Day 3-4 |
| 006 - User | ⏳ To Do | - | 5h | Day 4 |
| 007 - Market Data | ⏳ To Do | - | 8h | Day 5-6 |
| 008 - Strategy | ⏳ To Do | - | 6h | Day 6-7 |

## Quick Links
- [Architecture Overview](../architecture-tong-the.md)
- [Full Task Breakdown](../task-breakdown.md)
- [Project Structure Guide](../project-structure.md)
- [SRS - Software Requirements](../srs-software-requirements.md)
- [PRD - Product Requirements](../prd-product-requirements.md)
