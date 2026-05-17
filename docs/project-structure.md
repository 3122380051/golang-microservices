# Go Project Structure - Cấu trúc thư mục dự án

## Cấu trúc chính

```
golang-microservices/
├── cmd/                          # Entrypoint của mỗi service
│   ├── api-gateway/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── auth-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── user-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── market-data-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── strategy-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── risk-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── order-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── execution-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── exchange-adapter-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── portfolio-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── notification-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   ├── audit-log-service/
│   │   ├── main.go
│   │   └── Dockerfile
│   └── migrate/
│       └── main.go               # DB migration tool
│
├── internal/                      # Shared internal code
│   ├── domain/                   # Domain models & interfaces
│   │   ├── user.go
│   │   ├── order.go
│   │   ├── execution.go
│   │   ├── portfolio.go
│   │   ├── strategy.go
│   │   ├── signal.go
│   │   ├── market.go
│   │   └── ...
│   │
│   ├── application/              # Application services (use cases)
│   │   ├── user/
│   │   │   ├── service.go
│   │   │   ├── dto.go
│   │   │   └── *_test.go
│   │   ├── auth/
│   │   │   ├── service.go
│   │   │   ├── jwt.go
│   │   │   └── *_test.go
│   │   ├── order/
│   │   │   ├── service.go
│   │   │   ├── state_machine.go
│   │   │   └── *_test.go
│   │   └── ...
│   │
│   ├── infrastructure/           # External dependencies
│   │   ├── repository/           # Database layers
│   │   │   ├── user_repo.go
│   │   │   ├── order_repo.go
│   │   │   ├── execution_repo.go
│   │   │   └── *_test.go
│   │   │
│   │   ├── broker/               # Message broker clients
│   │   │   ├── kafka_client.go
│   │   │   ├── nats_client.go
│   │   │   ├── event_publisher.go
│   │   │   └── event_consumer.go
│   │   │
│   │   ├── exchange/             # Exchange adapters
│   │   │   ├── binance/
│   │   │   │   ├── adapter.go
│   │   │   │   ├── client.go
│   │   │   │   └── *_test.go
│   │   │   ├── bybit/
│   │   │   ├── okx/
│   │   │   └── adapter_factory.go
│   │   │
│   │   ├── cache/                # Redis caching
│   │   │   ├── redis_client.go
│   │   │   ├── market_price_cache.go
│   │   │   └── *_test.go
│   │   │
│   │   └── notification/         # Notification channels
│   │       ├── email.go
│   │       ├── telegram.go
│   │       └── *_test.go
│   │
│   ├── transport/                # HTTP/gRPC handlers
│   │   ├── http/
│   │   │   ├── handler.go
│   │   │   ├── middleware.go
│   │   │   ├── auth_handler.go
│   │   │   ├── user_handler.go
│   │   │   ├── order_handler.go
│   │   │   └── *_test.go
│   │   ├── grpc/
│   │   │   ├── auth.pb.go
│   │   │   ├── order.pb.go
│   │   │   └── ...
│   │   └── websocket/
│   │       ├── market_stream.go
│   │       └── *_test.go
│   │
│   ├── config/                   # Configuration management
│   │   ├── config.go
│   │   ├── env.go
│   │   └── *_test.go
│   │
│   ├── observability/            # Logging, metrics, tracing
│   │   ├── logger.go
│   │   ├── metrics.go
│   │   ├── tracer.go
│   │   └── *_test.go
│   │
│   └── common/                   # Shared utilities
│       ├── errors.go
│       ├── pagination.go
│       ├── correlation_id.go
│       └── *_test.go
│
├── migrations/                    # Database migrations
│   ├── 001_init_users.sql
│   ├── 002_init_roles.sql
│   ├── 003_init_orders.sql
│   ├── 004_init_executions.sql
│   ├── 005_init_portfolios.sql
│   ├── 006_init_strategies.sql
│   ├── 007_init_audit_logs.sql
│   └── migrations.go             # Migration runner
│
├── proto/                         # Protocol Buffer definitions (jika gunakan gRPC)
│   ├── auth.proto
│   ├── order.proto
│   ├── execution.proto
│   └── Makefile                  # Proto compilation
│
├── docs/                          # Documentation (sudah ada)
│   ├── architecture-tong-the.md
│   ├── erd-chi-tiet.md
│   ├── thiet-ke-chi-tiet-thanh-phan-va-patterns.md
│   ├── prd-product-requirements.md
│   ├── srs-software-requirements.md
│   ├── task-breakdown.md
│   └── project-structure.md       # File ini
│
├── scripts/                       # Utility scripts
│   ├── migrate.sh                # Run migrations
│   ├── docker-build.sh           # Build all docker images
│   ├── docker-push.sh            # Push to registry
│   ├── local-setup.sh            # Local dev setup
│   └── seed-data.sql             # Seed test data
│
├── tests/                         # Integration & end-to-end tests
│   ├── integration/
│   │   ├── auth_flow_test.go
│   │   ├── order_flow_test.go
│   │   ├── execution_test.go
│   │   └── fixtures.go
│   ├── e2e/
│   │   ├── order_to_execution_test.go
│   │   └── full_flow_test.go
│   └── load/
│       ├── order_load_test.go
│       └── market_data_load_test.go
│
├── .docker/                       # Docker-related files
│   ├── Dockerfile.base           # Base image cho tất cả service
│   └── Dockerfile.ci             # CI build image
│
├── .github/                       # GitHub workflows
│   └── workflows/
│       ├── ci.yml
│       ├── deploy-dev.yml
│       └── deploy-prod.yml
│
├── docker-compose.yml             # Local dev environment
├── docker-compose.prod.yml        # Production compose
├── Makefile                       # Build targets
├── .env.example                   # Environment variables template
├── .gitignore
├── go.mod
├── go.sum
├── README.md
├── CONTRIBUTING.md
└── LICENSE
```

## File mẫu cho mỗi package

### Domain Layer (`internal/domain/`)
```go
// user.go
package domain

type User struct {
    ID    string
    Email string
    Name  string
    // ...
}

type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
    // ...
}
```

### Application Layer (`internal/application/`)
```go
// user/service.go
package user

type Service struct {
    repo domain.UserRepository
}

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // use case logic
}
```

### Infrastructure Layer (`internal/infrastructure/`)
```go
// repository/user_repo.go
package repository

type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    // DB query
}
```

### Transport Layer (`internal/transport/`)
```go
// http/user_handler.go
package http

type UserHandler struct {
    service *application.UserService
}

func (h *UserHandler) GetUser(c echo.Context) error {
    // HTTP handler
}
```

## Quy ước đặt tên

### File
- Lowercase + underscore: `user_repository.go`, `auth_handler.go`.
- Test file: `*_test.go` trong cùng package.
- Interface file riêng nếu cần: `interfaces.go`.

### Package
- Đặt theo domain/functionality.
- Tránh generic name như `utils`, `helper` (có thể dùng `common`).
- Avoid circular import bằng cách đảm bảo dependency flow: domain → application → infrastructure.

### Interface
- Suffix `er` hoặc `or`: `Repository`, `Publisher`, `Adapter`.
- Nhỏ và focused: mỗi interface 1-3 method.

## Build & Run

### Build individual service
```bash
cd cmd/auth-service
go build -o bin/auth-service
```

### Build all services
```bash
make build-all
```

### Run locally
```bash
docker-compose up
```

### Run migrations
```bash
./scripts/migrate.sh
```

## Testing

### Unit test
```bash
go test ./internal/...
```

### Integration test
```bash
go test ./tests/integration/...
```

### Load test
```bash
go test ./tests/load/... -bench .
```

## Service dependencies

Mỗi service sẽ có:
- `cmd/{service}/main.go` - entrypoint.
- `internal/{domain,application,infrastructure}/` - business logic.
- `internal/transport/http` - HTTP handlers.
- Dockerfile.
- _test.go files cho mỗi package.

Ví dụ: **auth-service**
```
cmd/auth-service/
├── main.go
├── Dockerfile
└── ...

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

## Khởi tạo project

```bash
# Init Go module
go mod init github.com/3122380051/golang-microservices

# Create folder structure
mkdir -p cmd/{api-gateway,auth-service,user-service,market-data-service,strategy-service,risk-service,order-service,execution-service,exchange-adapter-service,portfolio-service,notification-service,audit-log-service,migrate}
mkdir -p internal/{domain,application,infrastructure,transport,config,observability,common}
mkdir -p migrations proto scripts tests/{integration,e2e,load} .docker .github/workflows

# Add common Go dependencies
go get github.com/labstack/echo/v4@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/lib/pq@latest
go get github.com/redis/go-redis/v9@latest
go get github.com/confluentinc/confluent-kafka-go@latest
go get go.uber.org/zap@latest
go get github.com/prometheus/client_golang@latest
```

## File hierarchy cho testing

```
tests/
├── integration/
│   ├── auth_flow_test.go          # Test đăng nhập, token refresh
│   ├── order_flow_test.go         # Test tạo order, status update
│   ├── execution_test.go          # Test gửi order lên sàn
│   ├── market_data_test.go        # Test market data fetch & cache
│   └── fixtures.go                # Setup test data & teardown
├── e2e/
│   ├── order_to_execution_test.go # Full flow: signal -> risk -> order -> execution
│   ├── portfolio_update_test.go   # Test portfolio update after execution
│   └── notification_test.go       # Test notification delivery
└── load/
    ├── order_load_test.go         # Load test order creation
    ├── market_data_load_test.go   # Load test market data stream
    └── execution_load_test.go     # Load test order submission
```

## Artifacts per service

Mỗi service khi complete phải deliver:
1. `cmd/{service}/main.go` + Dockerfile.
2. Application service logic + tests.
3. Repository layer + tests.
4. HTTP handlers + tests.
5. Consumer logic (nếu event-driven).
6. Docker image.

---

Sử dụng tài liệu này cùng với [Task Breakdown](task-breakdown.md) để biết folder nào cần tạo cho mỗi task.
