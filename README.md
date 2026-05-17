# golang-microservices

Hệ thống trading crypto bằng Go microservices.

## Tài liệu chính

### Thiết kế & Kiến trúc
- [Kiến trúc tổng thể](docs/architecture-tong-the.md) - Sơ đồ hệ thống, các lớp chính, luồng xử lý.
- [Thiết kế chi tiết thành phần & Pattern](docs/thiet-ke-chi-tiet-thanh-phan-va-patterns.md) - Chi tiết từng service, trách nhiệm, pattern sử dụng.
- [ERD Database](docs/erd-chi-tiet.md) - Sơ đồ quan hệ dữ liệu, schema chi tiết.
- [Project Structure](docs/project-structure.md) - Cấu trúc thư mục Go, package organization, file conventions.

### Yêu cầu & Kế hoạch
- [PRD - Product Requirements](docs/prd-product-requirements.md) - Mục tiêu kinh doanh, tính năng, scope MVP.
- [SRS - Software Requirements](docs/srs-software-requirements.md) - Yêu cầu chức năng chi tiết, interface, API, dữ liệu.
- [Task Breakdown](docs/task-breakdown.md) - Chi tiết công việc từng tuần, effort, dependencies, prioritas, folder deliverables.

### Tasks theo Giai đoạn
- [**TASK INDEX**](docs/tasks/INDEX.md) - Liên kết tất cả task folder, từng task có SRS/PRD/Deliverables riêng
  - [AI Workflow](docs/tasks/AI_WORKFLOW.md) - Quy trình làm từng task theo thứ tự
  - [TASK-001: Setup Project Structure](docs/tasks/001-setup-project-structure/README.md)
  - [TASK-002: Database Schema](docs/tasks/002-database-schema/README.md)
  - [TASK-003: Message Broker](docs/tasks/003-message-broker/README.md)
  - [TASK-004: API Gateway](docs/tasks/004-api-gateway/README.md)
  - [TASK-005: Auth Service](docs/tasks/005-auth-service/README.md)
  - [TASK-006: User Service](docs/tasks/006-user-service/README.md)
  - [TASK-007: Market Data Service](docs/tasks/007-market-data-service/README.md)
  - [TASK-008: Strategy Service](docs/tasks/008-strategy-service/README.md)

## Quick Start

```bash
# Setup dev environment
make setup

# Run database migrations
make migrate-up

# Start the gateway
make run
```

## Cấu trúc dự án
```
golang-microservices/
├── cmd/                          # Mỗi service có folder riêng
│   ├── api-gateway/
│   ├── auth-service/
│   ├── market-data-service/
│   └── ...
├── internal/                      # Shared internal code
│   ├── domain/                    # Domain models
│   ├── application/               # Business logic
│   ├── infrastructure/            # DB, external services
│   ├── transport/                 # HTTP handlers
│   ├── config/
│   ├── observability/
│   └── common/
├── migrations/                    # Database migrations
├── proto/                         # Protocol Buffer definitions
├── scripts/                       # Utility scripts
├── tests/                         # Integration & e2e tests
├── docs/                          # Documentation
├── docker-compose.yml             # Local dev
└── README.md
```

Xem [docs/project-structure.md](docs/project-structure.md) để hiểu chi tiết cấu trúc thư mục.

## License
MIT