# SRS - Software Requirements Specification
## Hệ thống Trading Crypto Microservices

### 1. Giới thiệu

#### 1.1 Mục đích tài liệu
Định nghĩa yêu cầu phần mềm chi tiết, interface, dữ liệu và ràng buộc kỹ thuật cho hệ thống trading crypto microservices.

#### 1.2 Phạm vi
- 12 microservice chính cộng thêm message broker, database, cache.
- Hỗ trợ Binance, Bybit, OKX qua adapter pattern.
- Event-driven architecture với eventual consistency.

### 2. Tổng quan kiến trúc hệ thống

Xem [docs/architecture-tong-the.md](docs/architecture-tong-the.md) và [docs/thiet-ke-chi-tiet-thanh-phan-va-patterns.md](docs/thiet-ke-chi-tiet-thanh-phan-va-patterns.md) để hiểu kiến trúc, service, pattern.

### 3. Yêu cầu chức năng chi tiết

#### 3.1 Chứng thực và Phân quyền

**Requirement AUTH-001: Đăng ký người dùng**
- Input: email, password.
- Output: user id, JWT token.
- Validation: email hợp lệ, password >= 8 ký tự.
- Action: hash password, lưu DB, gửi confirmation email.
- Error handling: email đã tồn tại -> HTTP 400.

**Requirement AUTH-002: Đăng nhập**
- Input: email, password.
- Output: access token (1h), refresh token (7d).
- Validation: email tồn tại, password match.
- Token: HS256 hoặc RS256, gồm user_id, email, roles.

**Requirement AUTH-003: Refresh token**
- Input: refresh token.
- Output: access token mới.
- Validation: refresh token hợp lệ và chưa hết hạn.

**Requirement AUTH-004: Revoke session**
- Input: user id.
- Output: revoke token.
- Action: thêm token vào blacklist trong Redis.

#### 3.2 Quản lý người dùng

**Requirement USER-001: Lấy thông tin người dùng**
- Input: user id.
- Output: profile (email, full_name, status, preferences).
- Permission: user chỉ xem được profile của chính họ (except admin).

**Requirement USER-002: Cập nhật profile**
- Input: full_name, language, timezone, notification channels.
- Output: updated profile.
- Validation: timezone trong danh sách hợp lệ.

**Requirement USER-003: Quản lý API Key**
- Input: exchange name, api_key, api_secret.
- Output: key_id, masked_key (chỉ show 4 ký tự cuối).
- Action: mã hóa api_secret trước lưu.
- Validation: key hợp lệ với exchange (nên test connect).

#### 3.3 Dữ liệu thị trường

**Requirement MARKET-001: Lấy dữ liệu giá real-time**
- Input: exchange, symbol, interval (1m, 5m, 15m, 1h).
- Output: OHLCV candle mới nhất.
- Source: cache Redis trước, fallback MarketData service.
- Latency SLA: < 100ms.

**Requirement MARKET-002: Stream giá mới**
- Input: symbol list.
- Output: WebSocket stream OHLCV + trade.
- Persistence: TimescaleDB lưu để backtest.

**Requirement MARKET-003: Order Book**
- Input: exchange, symbol.
- Output: bid/ask levels (top 20).
- Latency SLA: < 200ms.

#### 3.4 Chiến lược giao dịch

**Requirement STRATEGY-001: Tạo chiến lược**
- Input: name, type (ema_cross, rsi, bollinger), config_json.
- Output: strategy id.
- Validation: config schema match chiến lược type.
- Config example: `{"fast_ema": 12, "slow_ema": 26, "signal": 9}`.

**Requirement STRATEGY-002: Kích hoạt/Vô hiệu chiến lược**
- Input: strategy_id, status (active/inactive).
- Output: updated strategy.
- Action: nếu active, bắt đầu consume market events.

**Requirement STRATEGY-003: Tạo signal**
- Input: market event (giá, candle, trade).
- Output: signal (buy, sell, hold) + metadata (reason, confidence).
- Constraint: idempotent per market event.

#### 3.5 Quản lý rủi ro

**Requirement RISK-001: Kiểm tra rủi ro**
- Input: signal, user portfolio state, exchange.
- Output: approved/rejected + reason.
- Checks:
  - Position size <= max_position_size.
  - Leverage <= user_max_leverage.
  - Daily loss < max_daily_loss.
  - Account margin > 50%.

**Requirement RISK-002: Cảnh báo rủi ro**
- Input: portfolio state.
- Output: alert nếu margin_ratio < threshold.
- Action: gửi notification cho user.

#### 3.6 Quản lý lệnh

**Requirement ORDER-001: Tạo lệnh**
- Input: signal, user_id, symbol, quantity, order_type.
- Output: order_id, status.
- Fields bắt buộc: client_order_id (UUID), correlation_id (trace).
- Status mới: created.

**Requirement ORDER-002: Lấy lệnh**
- Input: order_id.
- Output: order detail (status, fills, fees).
- Filter: user chỉ xem order của chính họ.

**Requirement ORDER-003: Hủy lệnh**
- Input: order_id.
- Output: updated order (status = canceled).
- Constraint: chỉ có thể hủy nếu status là created hoặc pending.
- Action: gửi cancel request sang sàn qua adapter.

#### 3.7 Thực hiện lệnh

**Requirement EXEC-001: Gửi lệnh lên sàn**
- Input: order (internal).
- Output: exchange_order_id, exchange, status.
- Idempotency: dùng client_order_id.
- Retry: max 3 lần, exponential backoff.
- Timeout: 10s per attempt.

**Requirement EXEC-002: Reconcile trạng thái lệnh**
- Input: exchange_order_id, exchange.
- Output: fills, fees, status.
- Schedule: mỗi 5-10s hoặc khi nhận webhook từ exchange.
- Action: update order table với fill detail.

#### 3.8 Danh mục

**Requirement PORTFOLIO-001: Lấy danh mục**
- Input: user_id.
- Output: {total_equity, available_balance, used_margin, realized_pnl, unrealized_pnl}.
- Real-time: update trong < 1s sau khớp lệnh.

**Requirement PORTFOLIO-002: Lấy vị thế**
- Input: user_id, symbol (optional).
- Output: list positions {symbol, qty, entry_price, mark_price, unrealized_pnl, side}.
- Mark price: giá mới nhất từ market data cache.

#### 3.9 Thông báo

**Requirement NOTIF-001: Gửi thông báo**
- Input: user_id, type (order_filled, position_closed, alert), channel (email/telegram/slack).
- Output: notification_id, sent_at.
- Template: sử dụng template riêng cho từng type.
- Retry: max 3 lần nếu fail.

#### 3.10 Ghi nhận kiểm toán

**Requirement AUDIT-001: Ghi log hành động**
- Input: user_id, action, entity_type, entity_id, metadata, trace_id.
- Output: audit_log_id.
- Immutable: append-only, không sửa sau khi ghi.
- Retention: giữ 2 năm.

### 4. Yêu cầu về dữ liệu

#### 4.1 Database
Xem [docs/erd-chi-tiet.md](docs/erd-chi-tiet.md) để xem schema chi tiết.

#### 4.2 Event Schema
- Mỗi event phải có: event_id, trace_id, timestamp, source.
- Version hóa schema (v1, v2, v3) để tương thích ngược.

#### 4.3 Cache
- Market price: TTL 10s.
- Session token: TTL = token expiry.
- Rate limit counter: TTL 1 phút.

### 5. Interface & API

#### 5.1 REST API Gateway

**POST /api/v1/auth/register**
```
Body: {email, password}
Response: {user_id, access_token, refresh_token}
```

**POST /api/v1/auth/login**
```
Body: {email, password}
Response: {access_token, refresh_token}
```

**POST /api/v1/strategies**
```
Header: Authorization: Bearer {token}
Body: {name, type, config}
Response: {strategy_id, created_at}
```

**GET /api/v1/orders**
```
Header: Authorization
Query: limit, offset, status, symbol
Response: [{order_id, symbol, side, qty, status, created_at}, ...]
```

**POST /api/v1/market/price**
```
Query: exchange, symbol
Response: {price, bid, ask, ts}
```

**GET /api/v1/portfolio**
```
Header: Authorization
Response: {total_equity, available_balance, unrealized_pnl, positions}
```

#### 5.2 Internal Service-to-Service
- gRPC cho high-frequency calls (market data, order submission).
- Event stream qua message broker cho async calls.

### 6. Yêu cầu phi chức năng

#### 6.1 Hiệu suất
- API P99 latency: < 200ms.
- Order submission P99 latency: < 500ms.
- Throughput: 1000 order/day ban đầu, scale tới 100k/day.

#### 6.2 Tính sẵn sàng
- Target: 99.5% uptime.
- RTO: < 15 phút.
- RPO: < 5 phút.

#### 6.3 Bảo mật
- OWASP Top 10 compliance.
- API key mã hóa AES-256.
- Password hash bcrypt với salt.
- Rate limit: 100 req/min per IP, 1000 req/min per user.

#### 6.4 Khả năng mở rộng
- Horizontal scaling cho stateless services.
- Message broker partition theo user_id hoặc symbol.
- Read replica cho PostgreSQL khi cần.

### 7. Yêu cầu về vận hành

#### 7.1 Logging
- Mỗi request/response phải có correlation_id.
- Log level: DEBUG, INFO, WARN, ERROR.
- Centralized logging (ELK hoặc CloudWatch).

#### 7.2 Monitoring & Alerting
- Metrics: request rate, latency, error rate, database query time.
- Alert: error rate > 1%, latency P99 > 1s.
- Dashboard: Grafana hoặc DataDog.

#### 7.3 Tracing
- Distributed tracing với Jaeger hoặc Datadog.
- Trace mỗi request qua toàn bộ service stack.

### 8. Ràng buộc kỹ thuật

- Language: Go 1.20+.
- Database: PostgreSQL 13+.
- Cache: Redis 6.0+.
- Message Broker: Kafka 2.8 hoặc NATS 2.0+.
- Container: Docker + Docker Compose.
- Orchestration: Kubernetes (optional for MVP).

### 9. Glossary

- **Signal**: tín hiệu giao dịch (buy/sell/hold) tạo bởi Strategy Service.
- **Order**: lệnh giao dịch nội bộ trong hệ thống.
- **Execution**: kết quả gửi lệnh lên exchange.
- **Position**: vị thế đang giữ của trader cho một symbol.
- **PnL**: Profit and Loss, lợi nhuận/lỗ.
- **Correlation ID**: ID theo dõi request xuyên suốt các service.
- **Idempotency**: thao tác lặp lại không tạo kết quả khác.

### 10. Tham chiếu

- [Architecture Overview](docs/architecture-tong-the.md)
- [Component Design & Patterns](docs/thiet-ke-chi-tiet-thanh-phan-va-patterns.md)
- [ERD Database](docs/erd-chi-tiet.md)
- [PRD Requirements](docs/prd-product-requirements.md)
