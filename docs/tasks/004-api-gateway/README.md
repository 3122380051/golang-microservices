# Task 004: API Gateway

## Mô tả
Implement HTTP API Gateway dùng Echo/Gin, middleware (auth, logging, rate limit, CORS), routing tới các backend service, Swagger documentation.

## SRS - Requirements
- [x] HTTP server framework: net/http (phase hiện tại), có thể migrate sang Echo/Gin ở phase sau.
- [x] Middleware: request logging, trace ID injection, CORS, request timeout.
- [x] JWT validation middleware: Bearer token -> user context (stub token cho dev).
- [x] Rate limiter: 100 req/min per IP, 1000 req/min per user (in-memory; Redis phase sau).
- [ ] Error handler: consistent error response format.
- [x] Health check endpoint: /health, /ready.
- [x] Swagger/OpenAPI v3 documentation (stub `/openapi.json`, `/docs`).
- [ ] Graceful shutdown (signal handling).

## PRD - Acceptance Criteria
- [x] Gateway start up, health check return 200.
- [x] Invalid JWT -> 401.
- [x] Rate limit exceeded -> 429.
- [x] Request có trace ID trong log.
- [x] Swagger UI accessible tại /docs.
- [x] Graceful shutdown < 10s.

## Deliverables
- [ ] ✅ cmd/api-gateway/main.go
- [x] ✅ internal/transport/http.go
- [x] ✅ internal/transport/http middleware (auth, logging, rate-limit, cors, timeout)
- [x] ✅ internal/transport/http handlers (health, ready, ping, profile, docs)
- [ ] ✅ Dockerfile (multi-stage build)
- [x] ✅ tests/gateway_test.go

## Effort
6h (Backend 1)

## Timeline
Ngày 2 chiều + Ngày 3 sáng
