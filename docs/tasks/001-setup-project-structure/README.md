# Task 001: Setup Project Structure

## Mô tả
Khởi tạo Go module, cấu trúc thư mục, common packages (config, logger, database, observability) và Docker Compose cho dev environment.

## SRS - Requirements
- [ ] Go 1.20+ module với go.mod, go.sum.
- [ ] Cấu trúc thư mục: cmd/, internal/, migrations/, proto/, scripts/, tests/.
- [ ] Config package: load từ env, .env file, config.yaml.
- [ ] Logger package: structured logging với slog hoặc zap.
- [ ] Database package: PostgreSQL connection pool, migration runner.
- [ ] Observability package: metrics, tracing hooks.
- [ ] Docker Compose: PostgreSQL, Redis, Kafka/NATS, local dev environment.
- [ ] Dockerfile template cho Go service.
- [ ] .gitignore, Makefile cho build/test/run.

## PRD - Acceptance Criteria
- [ ] Team có thể `make setup` và có full dev environment up.
- [ ] Team có thể `go run ./cmd/api-gateway/main.go` không error.
- [ ] Config được load từ `.env` hoặc env var.
- [ ] Logger output có structured format (JSON).
- [ ] Database migration runner hoạt động (up/down/status).
- [ ] All services có Dockerfile và Docker Compose config.
- [ ] README có quick start instructions.

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
done
