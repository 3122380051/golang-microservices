# Task 005: Auth Service

## Mô tả
Implement Auth Service: đăng ký, đăng nhập, JWT token generation, refresh token, logout, phân quyền RBAC, token revocation.

## SRS - Requirements
- [ ] User registration: email validation, password hashing (bcrypt), duplicate check.
- [ ] User login: credential validation, JWT + refresh token generation.
- [ ] JWT access token: 1h expiry, HS256 signed.
- [ ] Refresh token: 7d expiry, rotate on use (optional).
- [ ] Token claims: user_id, email, roles (JSON).
- [ ] Logout: add token to Redis blacklist.
- [ ] Password reset: send email link (optional for MVP).
- [ ] RBAC: store user_roles, check permission in API Gateway.

## PRD - Acceptance Criteria
- [ ] POST /auth/register {email, password} -> {user_id, access_token, refresh_token}.
- [ ] POST /auth/login {email, password} -> tokens.
- [ ] POST /auth/refresh {refresh_token} -> new access_token.
- [ ] Invalid credential -> 401.
- [ ] Duplicate email -> 409.
- [ ] Weak password (< 8 char) -> 400.
- [ ] Logout -> subsequent token rejected.

## Deliverables
- [ ] ✅ cmd/auth-service/main.go
- [ ] ✅ internal/domain/user.go, token.go
- [ ] ✅ internal/application/auth/service.go
- [ ] ✅ internal/infrastructure/repository/user_repository.go
- [ ] ✅ internal/transport/http/auth_handler.go
- [ ] ✅ tests/auth_service_test.go
- [ ] ✅ Dockerfile, docker-compose update

## Effort
6h (Backend 2)

## Timeline
Ngày 3 chiều + Ngày 4 sáng
