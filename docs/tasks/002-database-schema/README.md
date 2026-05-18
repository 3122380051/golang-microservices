# Task 002: Database Schema & PostgreSQL Setup

## Mô tả
Thiết kế và tạo PostgreSQL schema với các bảng: users, roles, strategies, orders, executions, portfolios, positions, audit_logs, notifications. Tạo migration files, indexes, seed data.

## SRS - Requirements
- [ ] PostgreSQL 13+ setup trong Docker Compose.
- [ ] Migration files (.up.sql, .down.sql) cho tất cả tables.
- [ ] Primary keys: UUID v4 cho mọi bảng.
- [ ] Foreign keys: enforced, cascade delete nếu cần.
- [ ] Indexes: ON user_id, symbol_id, status, created_at.
- [ ] Seed data: roles (admin, user, trader).
- [ ] Connection pooling: max_open_conns=25, max_idle_conns=5.
- [ ] Backup strategy: pg_dump script.

## PRD - Acceptance Criteria
- [ ] Chạy `make migrate up` tạo toàn bộ schema không error.
- [ ] Chạy `make migrate down` xóa schema không error.
- [ ] Select từ users, roles bảng seed data hiểu thấy.
- [ ] Foreign key constraint check hoạt động (insert invalid id -> error).
- [ ] Query với index chạy < 100ms.

## Deliverables
- [x] ✅ 001_init_users.up/down.sql
- [x] ✅ 002_init_roles.up/down.sql
- [x] ✅ 003_init_strategies.up/down.sql
- [x] ✅ 004_init_orders_executions.up/down.sql
- [x] ✅ 005_init_portfolio_positions.up/down.sql
- [x] ✅ 006_init_audit_logs.up/down.sql
- [x] ✅ 007_init_notifications.up/down.sql
- [x] ✅ scripts/seed.sql
- [x] ✅ scripts/backup.sh

## Implementation Notes
- `000_init` bật extension `pgcrypto` để dùng `gen_random_uuid()` cho UUID PK.
- Migrations được chia theo thứ tự phụ thuộc: users -> roles -> strategies -> orders/executions -> portfolios/positions -> audit_logs -> notifications.
- `scripts/seed.sql` nạp dữ liệu roles mặc định.
- `scripts/backup.sh` dùng `pg_dump` và lưu file vào thư mục `backups/`.

## Effort
4h (DBA/Backend)

## Timeline
Ngày 1 chiều

## Status
✅ **COMPLETED** - PostgreSQL schema fully implemented
- 8 migration files created (users, roles, strategies, orders, executions, portfolios, positions, audit_logs, notifications)
- UUID v4 primary keys and foreign key constraints enforced
- Indexes on user_id, symbol_id, status, created_at
- Seed data script for default roles
- Connection pooling configured (max_open=25, max_idle=5)
- Backup script with pg_dump
