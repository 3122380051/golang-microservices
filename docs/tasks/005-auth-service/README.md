# Task 005: Auth Service

## Mô tả
Implement Auth Service: đăng ký, đăng nhập, JWT token generation, refresh token, logout, phân quyền RBAC, token revocation.

## SRS - Requirements
- [x] User registration: email validation, password hashing (bcrypt), duplicate check.
- [x] User login: credential validation, JWT + refresh token generation.
- [x] JWT access token: 1h expiry, HS256 signed.
- [x] Refresh token: 7d expiry, rotate on use.
- [x] Token claims: user_id, email, roles (JSON).
- [x] Logout: token blacklist (in-memory for current phase).
- [ ] Password reset: send email link (optional for MVP).
- [x] RBAC: store user_roles (default role assignment in auth-service).

## PRD - Acceptance Criteria
- [x] POST /auth/register {email, password} -> {user_id, access_token, refresh_token}.
- [x] POST /auth/login {email, password} -> tokens.
- [x] POST /auth/refresh {refresh_token} -> new access_token.
- [x] Invalid credential -> 401.
- [x] Duplicate email -> 409.
- [x] Weak password (< 8 char) -> 400.
- [x] Logout -> subsequent token rejected.

## Deliverables
- [x] ✅ cmd/auth-service/main.go
- [x] ✅ internal/domain/user.go, token.go
- [x] ✅ internal/application/auth/service.go
- [x] ✅ internal/infrastructure/repository/user_repository.go
- [x] ✅ internal/transport/http/auth_handler.go
- [x] ✅ tests/auth_service_test.go
- [x] ✅ Dockerfile, docker-compose update

## Effort
6h (Backend 2)

## Timeline
Ngày 3 chiều + Ngày 4 sáng

## Status
✅ **COMPLETED** - Auth Service fully operational
- User registration with email validation, bcrypt password hashing, duplicate email check
- Login endpoint with JWT + refresh token generation
- Access token: 60-minute expiry, HS256 signed
- Refresh token: 168-hour (7d) expiry with rotation on use
- Token claims: user_id, email, roles (JSON)
- Logout with token blacklist (in-memory for MVP)
- RBAC support with default role assignment
- All endpoints tested and validated
- Port: 8081
