# Task Breakdown - Chi tiết công việc cần làm

## 📋 Quick Navigation to Task Folders

Mỗi task có thư mục riêng với SRS, PRD, acceptance criteria:

- [AI Workflow](tasks/AI_WORKFLOW.md) - Quy trình thực thi từng task theo thứ tự.
- **[TASK INDEX](tasks/INDEX.md)** - Liên kết tất cả tasks
- [TASK-001: Setup Project Structure](tasks/001-setup-project-structure/README.md)
- [TASK-002: Database Schema](tasks/002-database-schema/README.md)
- [TASK-003: Message Broker](tasks/003-message-broker/README.md)
- [TASK-004: API Gateway](tasks/004-api-gateway/README.md)
- [TASK-005: Auth Service](tasks/005-auth-service/README.md)
- [TASK-006: User Service](tasks/006-user-service/README.md)
- [TASK-007: Market Data Service](tasks/007-market-data-service/README.md)
- [TASK-008: Strategy Service](tasks/008-strategy-service/README.md)

## Phân chia theo Giai đoạn

### Giai đoạn 1: Hạ tầng cơ bản (Tuần 1-2)

#### 1.1 Setup project structure
- [ ] **TASK-001**: Khởi tạo Go module và cấu trúc thư mục.
  - [ ] Tạo cmd/, internal/, migrations/ folders.
  - [ ] Setup .gitignore, go.mod, go.sum.
  - [ ] Tạo Dockerfile, docker-compose.yml cho dev.
  - **Deliverables (Folders)**:
    ```
    cmd/
      ├── api-gateway/
      ├── auth-service/
      ├── user-service/
      ├── market-data-service/
      ├── strategy-service/
      ├── risk-service/
      ├── order-service/
      ├── execution-service/
      ├── exchange-adapter-service/
      ├── portfolio-service/
      ├── notification-service/
      ├── audit-log-service/
      └── migrate/
    internal/
      ├── domain/
      ├── application/
      ├── infrastructure/
      ├── transport/
      ├── config/
      ├── observability/
      └── common/
    migrations/
    proto/
    scripts/
    tests/
    ```
  - Effort: 2h
  - Owner: DevOps/Lead

- [ ] **TASK-002**: Thiết lập common packages.
  - [ ] config package: load từ env, config file.
  - [ ] logger package: structured logging với slog.
  - [ ] database package: connection pool, migration.
  - [ ] observability package: metrics, tracing, logging hooks.
  - Effort: 4h
  - Owner: Backend Lead

#### 1.2 Database & Storage
- [ ] **TASK-003**: Thiết kế và tạo schema PostgreSQL.
  - [ ] Implement migration file cho users, roles, strategies, orders, etc.
  - [ ] Tạo index trên các trường query thường xuyên.
  - [ ] Seed data cho roles: admin, user, trader.
  - Effort: 4h
  - Owner: DBA/Backend

- [ ] **TASK-004**: Redis setup.
  - [ ] Client Redis, connection pool.
  - [ ] Các key naming convention (cache:*, lock:*, rate-limit:*).
  - Effort: 1h
  - Owner: DevOps

#### 1.3 Message Broker
- [ ] **TASK-005**: Setup Kafka/NATS/RabbitMQ.
  - [ ] Docker compose config cho broker.
  - [ ] Topic/queue definition: market.price, strategy.signal, order.*, execution.*, portfolio.*.
  - [ ] Consumer group strategy cho scaling.
  - Effort: 3h
  - Owner: DevOps/Backend Lead

- [ ] **TASK-006**: Event schema definition.
  - [ ] Define Avro/JSON schema cho từng event type.
  - [ ] Version hóa schema (v1, v2).
  - [ ] Swagger/OpenAPI cho event documentation.
  - Effort: 2h
  - Owner: Backend Lead

#### 1.4 API Gateway
- [ ] **TASK-007**: Implement API Gateway.
  - [ ] HTTP server (Echo hoặc Gin).
  - [ ] Middleware: logging, tracing, CORS.
  - [ ] Rate limiter middleware.
  - [ ] JWT validation middleware.
  - Effort: 6h
  - Owner: Backend 1

- [ ] **TASK-008**: API documentation.
  - [ ] Setup Swagger/OpenAPI.
  - [ ] Document auth endpoints.
  - Effort: 2h
  - Owner: Backend 1

### Giai đoạn 2: Xác thực & Người dùng (Tuần 2-3)

#### 2.1 Auth Service
- [ ] **TASK-009**: Implement Auth Service structure.
  - [ ] Domain: User, Token, Role entities.
  - [ ] Repository: User and token repository.
  - [ ] Service layer: login, register, refresh, revoke.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/auth-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── user.go
      ├── token.go
      └── role.go
    internal/application/auth/
      ├── service.go
      ├── jwt.go
      ├── dto.go
      └── service_test.go
    internal/infrastructure/repository/
      ├── user_repo.go
      └── user_repo_test.go
    internal/transport/http/
      ├── auth_handler.go
      └── auth_handler_test.go
    ```
  - Effort: 6h
  - Owner: Backend 2

- [ ] **TASK-010**: Password hashing & JWT.
  - [ ] Bcrypt hashing cho password.
  - [ ] JWT token generation (access + refresh).
  - [ ] Token claims structure (user_id, email, roles).
  - Effort: 3h
  - Owner: Backend 2

- [ ] **TASK-011**: Auth endpoints.
  - [ ] POST /auth/register, /auth/login, /auth/refresh, /auth/logout.
  - [ ] Input validation, error handling.
  - [ ] Audit log cho login/logout.
  - Effort: 4h
  - Owner: Backend 2

#### 2.2 User Service
- [ ] **TASK-012**: Implement User Service.
  - [ ] Domain: User profile, preferences.
  - [ ] Repository: CRUD user profile.
  - [ ] Service layer: update profile, manage API keys.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/user-service/
      ├── main.go
      └── Dockerfile
    internal/application/user/
      ├── service.go
      ├── dto.go
      └── service_test.go
    internal/infrastructure/repository/
      └── profile_repo.go
    internal/transport/http/
      ├── user_handler.go
      └── user_handler_test.go
    ```
  - Effort: 5h
  - Owner: Backend 3

- [ ] **TASK-013**: User endpoints.
  - [ ] GET /users/me, POST /users/me (update).
  - [ ] POST /users/api-keys, GET /users/api-keys, DELETE /users/api-keys/{id}.
  - [ ] Encrypt/decrypt API keys.
  - Effort: 4h
  - Owner: Backend 3

- [ ] **TASK-014**: API Key encryption.
  - [ ] AES-256 encryption/decryption.
  - [ ] Key rotation strategy.
  - Effort: 2h
  - Owner: Backend 3

#### 2.3 Tests
- [ ] **TASK-015**: Unit tests cho Auth & User services.
  - [ ] Test password hashing.
  - [ ] Test JWT generation/validation.
  - [ ] Test API key encryption.
  - [ ] Test edge cases (invalid email, weak password).
  - Effort: 6h
  - Owner: QA/Backend Lead

### Giai đoạn 3: Dữ liệu thị trường (Tuần 3-4)

#### 3.1 Market Data Service
- [ ] **TASK-016**: Implement Market Data Service structure.
  - [ ] Domain: Market symbol, price, candle.
  - [ ] Repository: symbol management.
  - [ ] Service: fetch từ exchange, normalize data.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/market-data-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── market.go
      ├── candle.go
      └── symbol.go
    internal/application/market/
      ├── service.go
      └── service_test.go
    internal/infrastructure/repository/
      ├── symbol_repo.go
      └── candle_repo.go
    internal/transport/
      ├── http/market_handler.go
      └── websocket/market_stream.go
    ```
  - Effort: 5h
  - Owner: Backend 4

- [ ] **TASK-017**: Exchange adapter for Binance.
  - [ ] REST client cho Binance API (ticker, candles, order book).
  - [ ] WebSocket client cho stream giá.
  - [ ] Normalize Binance response -> internal model.
  - [ ] Error handling, retry, circuit breaker.
  - Effort: 8h
  - Owner: Backend 4

- [ ] **TASK-018**: Market data caching.
  - [ ] Cache giá trong Redis (TTL 10s).
  - [ ] Cache candle trong TimescaleDB.
  - [ ] Cache invalidation strategy.
  - Effort: 4h
  - Owner: Backend 4

- [ ] **TASK-019**: Market data events.
  - [ ] Publish market.price.updated event.
  - [ ] Publish market.candle.created event.
  - [ ] Event batching nếu cần optimize.
  - Effort: 3h
  - Owner: Backend 4

- [ ] **TASK-020**: Market data API.
  - [ ] GET /market/price?exchange=binance&symbol=BTCUSDT.
  - [ ] GET /market/candles?symbol=BTCUSDT&interval=1h&limit=100.
  - [ ] GET /market/order-book?symbol=BTCUSDT.
  - [ ] WebSocket endpoint /ws/market/stream.
  - Effort: 5h
  - Owner: Backend 4

### Giai đoạn 4: Chiến lược & Tín hiệu (Tuần 4-5)

#### 4.1 Strategy Service
- [ ] **TASK-021**: Strategy Service structure.
  - [ ] Domain: Strategy, signal entities.
  - [ ] Repository: strategy CRUD, signal history.
  - [ ] Service: evaluate strategy, generate signal.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/strategy-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── strategy.go
      └── signal.go
    internal/application/strategy/
      ├── service.go
      ├── evaluator.go
      └── service_test.go
    internal/infrastructure/repository/
      ├── strategy_repo.go
      └── signal_repo.go
    internal/transport/http/
      ├── strategy_handler.go
      └── strategy_handler_test.go
    ```
  - Effort: 5h
  - Owner: Backend 5

- [ ] **TASK-022**: Implement EMA Cross strategy.
  - [ ] Calculate EMA fast, slow.
  - [ ] Generate buy/sell signal khi cross.
  - [ ] Confidence calculation.
  - [ ] Unit tests.
  - Effort: 6h
  - Owner: Backend 5

- [ ] **TASK-023**: Implement RSI strategy (optional for MVP).
  - [ ] Calculate RSI.
  - [ ] Overbought/oversold logic.
  - [ ] Unit tests.
  - Effort: 4h
  - Owner: Backend 5 (optional)

- [ ] **TASK-024**: Consumer market data events.
  - [ ] Subscribe market.price.updated.
  - [ ] Evaluate strategy mỗi event.
  - [ ] Publish strategy.signal.generated khi có signal.
  - Effort: 4h
  - Owner: Backend 5

- [ ] **TASK-025**: Strategy API.
  - [ ] POST /strategies (create).
  - [ ] GET /strategies, GET /strategies/{id}.
  - [ ] POST /strategies/{id}/activate, /deactivate.
  - [ ] GET /strategies/{id}/signals (history).
  - Effort: 4h
  - Owner: Backend 5

### Giai đoạn 5: Rủi ro & Lệnh (Tuần 5-6)

#### 5.1 Risk Service
- [ ] **TASK-026**: Risk Service structure.
  - [ ] Domain: RiskPolicy, check result.
  - [ ] Repository: risk config per user.
  - [ ] Service: evaluate risk policies.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/risk-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── risk.go
      └── policy.go
    internal/application/risk/
      ├── service.go
      ├── policies.go
      └── service_test.go
    internal/infrastructure/repository/
      └── policy_repo.go
    internal/transport/http/
      ├── risk_handler.go
      └── risk_handler_test.go
    ```
  - Effort: 4h
  - Owner: Backend 6

- [ ] **TASK-027**: Implement risk policies.
  - [ ] Position size check.
  - [ ] Leverage check.
  - [ ] Daily loss check.
  - [ ] Margin check.
  - [ ] Unit tests.
  - Effort: 6h
  - Owner: Backend 6

- [ ] **TASK-028**: Consumer strategy signals.
  - [ ] Subscribe strategy.signal.generated.
  - [ ] Evaluate risk policies.
  - [ ] Publish risk.order.approved hoặc risk.order.rejected.
  - Effort: 3h
  - Owner: Backend 6

- [ ] **TASK-029**: Risk API.
  - [ ] POST /risk/policies (manage user policies).
  - [ ] GET /risk/status (check current risk state).
  - Effort: 2h
  - Owner: Backend 6

#### 5.2 Order Service
- [ ] **TASK-030**: Order Service structure.
  - [ ] Domain: Order, state machine.
  - [ ] Repository: order CRUD.
  - [ ] Service: create, update, cancel order.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/order-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── order.go
      └── state_machine.go
    internal/application/order/
      ├── service.go
      ├── commands.go
      └── service_test.go
    internal/infrastructure/repository/
      └── order_repo.go
    internal/transport/http/
      ├── order_handler.go
      └── order_handler_test.go
    ```
  - Effort: 5h
  - Owner: Backend 7

- [ ] **TASK-031**: Order state machine.
  - [ ] States: created, pending, submitted, partially_filled, filled, canceled, rejected.
  - [ ] Transitions logic.
  - [ ] Guard checks.
  - [ ] Unit tests.
  - Effort: 5h
  - Owner: Backend 7

- [ ] **TASK-032**: Consumer risk approval events.
  - [ ] Subscribe risk.order.approved.
  - [ ] Create order nội bộ.
  - [ ] Publish order.created event.
  - Effort: 3h
  - Owner: Backend 7

- [ ] **TASK-033**: Order API.
  - [ ] GET /orders, GET /orders/{id}.
  - [ ] POST /orders (manual create, admin only).
  - [ ] DELETE /orders/{id} (cancel).
  - Effort: 3h
  - Owner: Backend 7

### Giai đoạn 6: Thực hiện & Danh mục (Tuần 6-7)

#### 6.1 Execution Service
- [ ] **TASK-034**: Execution Service structure.
  - [ ] Domain: Execution entity.
  - [ ] Repository: execution tracking.
  - [ ] Service: submit, reconcile, retry logic.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/execution-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      └── execution.go
    internal/application/execution/
      ├── service.go
      ├── submitter.go
      ├── reconciler.go
      └── service_test.go
    internal/infrastructure/repository/
      └── execution_repo.go
    internal/transport/http/
      ├── execution_handler.go
      └── execution_handler_test.go
    ```
  - Effort: 5h
  - Owner: Backend 8

- [ ] **TASK-035**: Consumer order created events.
  - [ ] Subscribe order.created.
  - [ ] Call exchange adapter.
  - [ ] Handle idempotency (client_order_id).
  - [ ] Publish execution.submitted event.
  - Effort: 4h
  - Owner: Backend 8

- [ ] **TASK-036**: Execution retry & timeout.
  - [ ] Exponential backoff.
  - [ ] Max retry (3-5 times).
  - [ ] Timeout per attempt (10-30s).
  - [ ] Circuit breaker.
  - Effort: 4h
  - Owner: Backend 8

- [ ] **TASK-037**: Order status reconciliation.
  - [ ] Poll exchange mỗi 5-10s.
  - [ ] Match fills với internal order.
  - [ ] Update order status, fees.
  - [ ] Publish execution.filled event.
  - Effort: 4h
  - Owner: Backend 8

#### 6.2 Exchange Adapter Service
- [ ] **TASK-038**: Binance adapter chi tiết.
  - [ ] POST order endpoint.
  - [ ] GET order status endpoint.
  - [ ] Cancel order endpoint.
  - [ ] Error mapping -> internal error type.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/exchange-adapter-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      └── adapter.go
    internal/infrastructure/exchange/
      ├── binance/
      │   ├── client.go
      │   ├── adapter.go
      │   ├── normalizer.go
      │   └── *_test.go
      ├── bybit/
      ├── okx/
      ├── adapter_factory.go
      └── adapter_interface.go
    internal/transport/http/
      ├── exchange_handler.go
      └── exchange_handler_test.go
    ```
  - Effort: 6h
  - Owner: Backend 9

- [ ] **TASK-039**: Exchange adapter abstraction.
  - [ ] Interface chung cho mỗi exchange.
  - [ ] Factory pattern.
  - [ ] Multi-exchange support (Bybit, OKX).
  - Effort: 6h
  - Owner: Backend 9

#### 6.3 Portfolio Service
- [ ] **TASK-040**: Portfolio Service structure.
  - [ ] Domain: Portfolio, Position entities.
  - [ ] Repository: portfolio, position CRUD.
  - [ ] Service: update balance, PnL, position.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/portfolio-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      ├── portfolio.go
      └── position.go
    internal/application/portfolio/
      ├── service.go
      ├── calculator.go
      └── service_test.go
    internal/infrastructure/repository/
      ├── portfolio_repo.go
      └── position_repo.go
    internal/transport/http/
      ├── portfolio_handler.go
      └── portfolio_handler_test.go
    ```
  - Effort: 5h
  - Owner: Backend 10

- [ ] **TASK-041**: Consumer execution filled events.
  - [ ] Subscribe execution.filled.
  - [ ] Update position (qty, avg_price).
  - [ ] Calculate unrealized PnL.
  - [ ] Publish portfolio.updated event.
  - Effort: 4h
  - Owner: Backend 10

- [ ] **TASK-042**: Market price consumer.
  - [ ] Subscribe market.price.updated.
  - [ ] Update mark_price cho positions.
  - [ ] Recalculate unrealized PnL.
  - Effort: 3h
  - Owner: Backend 10

- [ ] **TASK-043**: Portfolio API.
  - [ ] GET /portfolio (balance, PnL).
  - [ ] GET /portfolio/positions.
  - [ ] GET /portfolio/history (trade history).
  - Effort: 3h
  - Owner: Backend 10

### Giai đoạn 7: Thông báo & Ghi nhận (Tuần 7)

#### 7.1 Notification Service
- [ ] **TASK-044**: Notification Service structure.
  - [ ] Domain: Notification, channel.
  - [ ] Repository: notification history.
  - [ ] Service: send notification multi-channel.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/notification-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      └── notification.go
    internal/application/notification/
      ├── service.go
      ├── template_engine.go
      └── service_test.go
    internal/infrastructure/
      ├── repository/notification_repo.go
      └── notification/
          ├── email.go
          ├── telegram.go
          ├── slack.go
          └── *_test.go
    internal/transport/http/
      ├── notification_handler.go
      └── notification_handler_test.go
    ```
  - Effort: 4h
  - Owner: Backend 11

- [ ] **TASK-045**: Email channel.
  - [ ] Email template.
  - [ ] SMTP client.
  - [ ] Retry logic.
  - Effort: 3h
  - Owner: Backend 11

- [ ] **TASK-046**: Telegram channel.
  - [ ] Telegram bot API client.
  - [ ] Message template.
  - [ ] Error handling.
  - Effort: 2h
  - Owner: Backend 11

- [ ] **TASK-047**: Notification consumer.
  - [ ] Subscribe relevant events (order.*, execution.*, portfolio.*).
  - [ ] Transform event -> notification.
  - [ ] Send notification.
  - Effort: 3h
  - Owner: Backend 11

#### 7.2 Audit Log Service
- [ ] **TASK-048**: Audit Log Service structure.
  - [ ] Domain: AuditLog entity (immutable).
  - [ ] Repository: append-only log.
  - [ ] Service: log action.
  - **Deliverables (Folders & Files)**:
    ```
    cmd/audit-log-service/
      ├── main.go
      └── Dockerfile
    internal/domain/
      └── audit_log.go
    internal/application/audit/
      ├── service.go
      └── service_test.go
    internal/infrastructure/repository/
      └── audit_log_repo.go
    internal/transport/http/
      ├── audit_handler.go
      └── audit_handler_test.go
    ```
  - Effort: 3h
  - Owner: Backend 12

- [ ] **TASK-049**: Audit log consumer.
  - [ ] Subscribe key events (auth.*, order.*, execution.*).
  - [ ] Append to audit log.
  - [ ] Ensure immutability.
  - Effort: 2h
  - Owner: Backend 12

- [ ] **TASK-050**: Audit API.
  - [ ] GET /audit/logs?user_id=...&action=...&from=...&to=...
  - [ ] Filtering, pagination.
  - Effort: 2h
  - Owner: Backend 12

### Giai đoạn 8: Testing & Deployment (Tuần 8)

#### 8.1 Integration Tests
- [ ] **TASK-051**: End-to-end order flow test.
  - [ ] Setup test data (user, strategy, market data).
  - [ ] Trigger signal -> risk check -> order create -> execution -> portfolio.
  - [ ] Assert results ở mỗi bước.
  - Effort: 8h
  - Owner: QA/Backend Lead

- [ ] **TASK-052**: Failure scenarios tests.
  - [ ] Exchange connection failure.
  - [ ] Risk rejection.
  - [ ] Execution timeout & retry.
  - [ ] Order partial fill.
  - Effort: 6h
  - Owner: QA/Backend Lead

#### 8.2 Load Testing
- [ ] **TASK-053**: Load test setup.
  - [ ] Tool: k6, JMeter hoặc Gatling.
  - [ ] Scenario: 100 concurrent users, 1000 requests/min.
  - [ ] Identify bottlenecks.
  - Effort: 4h
  - Owner: DevOps/QA

- [ ] **TASK-054**: Performance optimization.
  - [ ] Database query optimization.
  - [ ] Cache strategy review.
  - [ ] Batch processing nếu cần.
  - Effort: 4h (tùy kết quả load test)
  - Owner: Backend Lead

#### 8.3 Security Review
- [ ] **TASK-055**: Code security scan.
  - [ ] SAST tool: Sonarqube, GoSec.
  - [ ] Fix critical & high severity issues.
  - Effort: 3h
  - Owner: Security/Backend Lead

- [ ] **TASK-056**: Dependency audit.
  - [ ] Check vulnerabilities: npm audit, go list -u.
  - [ ] Update vulnerable dependencies.
  - Effort: 2h
  - Owner: DevOps

#### 8.4 Documentation
- [ ] **TASK-057**: API documentation update.
  - [ ] Swagger/OpenAPI complete.
  - [ ] Example requests/responses.
  - Effort: 3h
  - Owner: Technical Writer/Backend Lead

- [ ] **TASK-058**: Deployment guide.
  - [ ] Docker build & push.
  - [ ] Docker Compose for dev.
  - [ ] Kubernetes config (optional).
  - [ ] Monitoring setup.
  - Effort: 4h
  - Owner: DevOps

#### 8.5 Deployment
- [ ] **TASK-059**: Dev environment deployment.
  - [ ] Deploy to dev server.
  - [ ] Smoke test.
  - Effort: 2h
  - Owner: DevOps

- [ ] **TASK-060**: Staging & Production preparation.
  - [ ] Setup staging environment.
  - [ ] Database backup strategy.
  - [ ] Rollback plan.
  - Effort: 3h
  - Owner: DevOps

## Prioritas & Dependencies

### Critical Path
1. TASK-001 to TASK-008: Hạ tầng (không thể bỏ).
2. TASK-009 to TASK-015: Auth & User (phải trước mọi API).
3. TASK-016 to TASK-020: Market Data (phụ thuộc của Strategy).
4. TASK-021 to TASK-025: Strategy (phụ thuộc của Risk & Order).
5. TASK-026 to TASK-037: Risk, Order, Execution (cốt lõi MVP).
6. TASK-040 to TASK-043: Portfolio (phụ thuộc Execution).
7. TASK-044 to TASK-050: Notification & Audit (có thể song song).
8. TASK-051 to TASK-060: Testing & Deployment (cuối).

### Optional cho MVP
- TASK-023: RSI strategy (có thể implement sau).
- TASK-039: Multi-exchange (tuy nhiên Binance bắt buộc).
- TASK-051 đến TASK-055: Nếu muốn MVP nhanh, có thể defer một phần.

## Effort Estimate
- Tổng effort (8 tuần): ~270 giờ.
- Trung bình ~ 34 giờ/tuần.
- Team size tối thiểu: 10-12 engineer.

## Resource Allocation
- 1x Backend Lead: Architecture, code review, mentoring.
- 2-3x Backend Engineer per giai đoạn.
- 1x DevOps: Infrastructure, Docker, Kubernetes.
- 1-2x QA: Unit test, integration test, load test.
- 1x Tech Lead/Architect: Design review, decision.

## Success Criteria per Giai đoạn
- Giai đoạn 1: Toàn bộ infra up, local dev working.
- Giai đoạn 2: Auth flow đầy đủ, user management hoạt động.
- Giai đoạn 3: Market data realtime, cache hoạt động tốt.
- Giai đoạn 4: Strategy signals chính xác, backtest (optional).
- Giai đoạn 5: Risk checks chuẩn, orders tạo được.
- Giai đoạn 6: Orders execute trên sàn thực, portfolio tracking.
- Giai đoạn 7: Notifications gửi được, audit log đầy đủ.
- Giai đoạn 8: Toàn bộ MVP deployable, test coverage > 70%.
