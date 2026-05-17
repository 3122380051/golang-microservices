# Task 006: User Service

## Mô tả
Implement User Service: quản lý profile, preferences, API key management (CRUD, encryption), audit log integration.

## SRS - Requirements
- [ ] GET /users/me: return user profile.
- [ ] PUT /users/me: update full_name, timezone, language, notification preferences.
- [ ] POST /users/api-keys: add exchange API key (encrypt api_secret).
- [ ] GET /users/api-keys: list user's API keys (masked).
- [ ] DELETE /users/api-keys/{id}: remove API key.
- [ ] Test connection: verify API key valid with exchange (Binance test).
- [ ] Encryption: AES-256 GCM cho api_secret.
- [ ] Audit: log create/delete/update API key.

## PRD - Acceptance Criteria
- [ ] Can create, read, delete API keys.
- [ ] API secret encrypted in DB, decrypted on use.
- [ ] Invalid API key -> error message, retry allowed.
- [ ] Audit log show user action + timestamp.
- [ ] User see only own API keys (auth check).

## Deliverables
- [ ] ✅ cmd/user-service/main.go
- [ ] ✅ internal/domain/user.go, api_key.go
- [ ] ✅ internal/application/user/service.go
- [ ] ✅ internal/infrastructure/repository/user_repository.go
- [ ] ✅ internal/infrastructure/crypto/encrypt.go
- [ ] ✅ internal/transport/http/user_handler.go
- [ ] ✅ tests/user_service_test.go

## Effort
5h (Backend 3)

## Timeline
Ngày 4 chiều
