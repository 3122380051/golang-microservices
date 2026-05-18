# Task 006: User Service

## Mô tả
Implement User Service: quản lý profile, preferences, API key management (CRUD, encryption), audit log integration.

## SRS - Requirements
- [x] GET /users/me: return user profile.
- [x] PUT /users/me: update full_name, timezone, language, notification preferences.
- [x] POST /users/api-keys: add exchange API key (encrypt api_secret).
- [x] GET /users/api-keys: list user's API keys (masked).
- [x] DELETE /users/api-keys/{id}: remove API key.
- [x] Test connection: verify API key valid with exchange (stub exchange validation).
- [x] Encryption: AES-256 GCM cho api_secret.
- [x] Audit: log create/delete/update API key.

## PRD - Acceptance Criteria
- [x] Can create, read, delete API keys.
- [x] API secret encrypted in DB, decrypted on use.
- [x] Invalid API key -> error message, retry allowed.
- [x] Audit log show user action + timestamp.
- [x] User see only own API keys (auth check via `X-User-ID`).

## Deliverables
- [x] ✅ cmd/user-service/main.go
- [x] ✅ internal/domain/user.go, api_key.go
- [x] ✅ internal/application/user/service.go
- [x] ✅ internal/infrastructure/repository/user_repository.go
- [x] ✅ internal/infrastructure/crypto/encrypt.go
- [x] ✅ internal/transport/http/user_handler.go
- [x] ✅ tests/user_service_test.go

## Effort
5h (Backend 3)

## Timeline
Ngày 4 chiều

## Status
✅ **COMPLETED** - User Service fully operational
- Profile management: GET/PUT /users/me
- Preferences: timezone, language, notification settings
- API key management: POST (create), GET (list with masking), DELETE
- AES-256-GCM encryption for api_secret storage
- Exchange API key validation (stub implementation)
- Audit logging for all key operations
- User isolation: only see own keys
- Integration tests covering all scenarios
- Port: 8082
