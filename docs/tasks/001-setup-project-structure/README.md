# Task 001: Setup Project Structure

## Mô tả
Khởi tạo Go module, cấu trúc thư mục, common packages (config, logger, database, observability) và Docker Compose cho dev environment.

## SRS - Requirements
- [x] Go 1.20+ module với go.mod, go.sum.
- [x] Cấu trúc thư mục: cmd/, internal/, migrations/, proto/, scripts/, tests/.
- [x] Config package: load từ env, .env file, config.yaml.
- [x] Logger package: structured logging với slog hoặc zap.
- [x] Database package: PostgreSQL connection pool, migration runner.
- [x] Observability package: metrics, tracing hooks (stub hooks implemented).
- [x] Docker Compose: PostgreSQL, Redis, Kafka/NATS, local dev environment.
- [x] Dockerfile template cho Go service.
- [x] .gitignore, Makefile cho build/test/run.

## PRD - Acceptance Criteria
- [x] Team có thể `make setup` và có full dev environment up.
- [x] Team có thể `go run ./cmd/api-gateway/main.go` không error.
- [x] Config được load từ `.env` hoặc env var.
- [x] Logger output có structured format (JSON).
- [x] Database migration runner hoạt động (up/down/status).
- [x] All services có Dockerfile và Docker Compose config.
- [x] README có quick start instructions.

## Folder Structure
```
.
├── cmd/
│   └── api-gateway/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── logger/
│   │   └── logger.go
│   ├── database/
│   │   ├── db.go
│   │   └── migration.go
│   ├── observability/
│   │   ├── metrics.go
│   │   ├── tracing.go
│   │   └── hooks.go
│   └── transport/
│       └── http.go
├── migrations/
│   └── 000_init.up.sql
├── scripts/
│   ├── setup.sh
│   ├── migrate.sh
│   └── seed.sh
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .gitignore
├── Makefile
└── README.md
```

## Deliverables
- [x] ✅ go.mod (github.com/3122380051/golang-microservices)
- [x] ✅ config.go (viper integration + .env support)
- [x] ✅ logger.go (slog JSON setup)
- [x] ✅ db.go (pgxpool + migrate integration)
- [x] ✅ Dockerfile, docker-compose.yml
- [x] ✅ Makefile (setup, run, test, lint, build)
- [x] ✅ .env.example
- [x] ✅ go.sum

## Implementation Notes
- `cmd/api-gateway/main.go` là entrypoint tối thiểu để validate project setup.
- `cmd/migrate/main.go` dùng để chạy migration `up|down|status`.
- `internal/transport/http.go` cung cấp `/health`, `/ready`, `/metrics`.
- `migrations/000_init.up.sql` là migration bootstrap tối thiểu cho Task 001.
- `scripts/setup.sh` dựng dev environment bằng Docker Compose.

## Effort
4h (Backend Lead)

## Timeline
Ngày 1 sáng

## Status
✅ **COMPLETED** - All deliverables implemented and tested
- Go 1.22.0 project structure initialized
- Config, logger, database, observability packages functional
- Docker Compose environment ready (postgres, redis, kafka)
- Makefile with setup/run/test/lint commands
- All services have Dockerfile with multi-stage build
