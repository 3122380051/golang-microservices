# Task 004: API Gateway

## Mô tả
Implement HTTP API Gateway dùng Echo/Gin, middleware (auth, logging, rate limit, CORS), routing tới các backend service, Swagger documentation.

## SRS - Requirements
- [ ] HTTP server framework: Echo hoặc Gin.
- [ ] Middleware: request logging, trace ID injection, CORS, request timeout.
- [ ] JWT validation middleware: Bearer token -> user context.
- [ ] Rate limiter: 100 req/min per IP, 1000 req/min per user (Redis backed).
- [ ] Error handler: consistent error response format.
- [ ] Health check endpoint: /health, /ready.
- [ ] Swagger/OpenAPI v3 documentation.
- [ ] Graceful shutdown (signal handling).

## PRD - Acceptance Criteria
- [ ] Gateway start up, health check return 200.
- [ ] Invalid JWT -> 401.
- [ ] Rate limit exceeded -> 429.
- [ ] Request có trace ID trong log.
- [ ] Swagger UI accessible tại /docs.
- [ ] Graceful shutdown < 10s.

## Deliverables
- [ ] ✅ cmd/api-gateway/main.go
- [ ] ✅ internal/transport/http/server.go
- [ ] ✅ internal/transport/http/middleware/ (auth, logging, rate-limit)
- [ ] ✅ internal/transport/http/handler/ (placeholder handlers)
- [ ] ✅ Dockerfile (multi-stage build)
- [ ] ✅ tests/gateway_test.go

## Effort
6h (Backend 1)

## Timeline
Ngày 2 chiều + Ngày 3 sáng
