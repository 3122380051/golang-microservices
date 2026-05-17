# AI Workflow - Execute Tasks Sequentially

## Mục tiêu
Tài liệu này mô tả quy trình để AI hoặc team lead thực thi các task theo đúng thứ tự, không bỏ qua dependency, và luôn cập nhật trạng thái sau mỗi bước.

## Nguyên tắc vận hành
- Chỉ làm **1 task tại một thời điểm**.
- Không bắt đầu task tiếp theo nếu task hiện tại chưa đạt acceptance criteria.
- Mọi thay đổi phải bám theo folder task tương ứng.
- Mọi task phải có: SRS, PRD, deliverables, checklist, và cập nhật trạng thái.
- Khi task hoàn thành, cập nhật `INDEX.md`, `task-breakdown.md`, và task folder README.

## Trạng thái task
Mỗi task chỉ đi qua 5 trạng thái:
- `todo`
- `in-progress`
- `blocked`
- `review`
- `done`

## Quy trình chuẩn cho mỗi task

### Step 1: Load task context
Đọc các tài liệu sau:
- `docs/tasks/INDEX.md`
- `docs/task-breakdown.md`
- `docs/tasks/{task-id}/README.md`
- `docs/architecture-tong-the.md`
- `docs/thiet-ke-chi-tiet-thanh-phan-va-patterns.md` nếu task liên quan service design
- `docs/erd-chi-tiet.md` nếu task liên quan database

### Step 2: Validate dependency
- Kiểm tra task trước đó đã xong chưa.
- Kiểm tra task hiện tại có phụ thuộc vào data model, broker, gateway hay service khác không.
- Nếu thiếu dependency, dừng lại ở trạng thái `blocked` và ghi rõ lý do.

### Step 3: Define deliverables
- Liệt kê folder/files cần tạo.
- Liệt kê API, entity, service, repository, tests.
- Liệt kê các artifact tài liệu nếu có.

### Step 4: Implement smallest vertical slice
- Tạo skeleton trước.
- Tạo domain model trước.
- Tạo application/service layer.
- Tạo infrastructure/repository/adapter sau.
- Tạo transport/API sau cùng.
- Viết test cho logic cốt lõi ngay sau khi code xong.

### Step 5: Validate
- Chạy test liên quan tới task.
- Kiểm tra lint/typecheck nếu có.
- Kiểm tra error output và sửa nếu task bị fail.
- Không chuyển task nếu validation chưa pass.

### Step 6: Update status
- Update checklist trong task README.
- Update `docs/tasks/INDEX.md` status nếu cần.
- Update `docs/task-breakdown.md` nếu thay đổi scope hoặc folder.
- Ghi ngắn gọn những gì đã làm và những gì còn lại.

## Quy trình điều phối toàn dự án

### Phase A: Foundation
1. TASK-001: Setup Project Structure
2. TASK-002: Database Schema
3. TASK-003: Message Broker
4. TASK-004: API Gateway

### Phase B: Identity
5. TASK-005: Auth Service
6. TASK-006: User Service

### Phase C: Market & Signals
7. TASK-007: Market Data Service
8. TASK-008: Strategy Service

### Phase D: Risk & Orders
9. TASK-009: Risk Service
10. TASK-010: Order Service
11. TASK-011: Execution Service
12. TASK-012: Exchange Adapter Service

### Phase E: Portfolio & Notification
13. TASK-013: Portfolio Service
14. TASK-014: Notification Service
15. TASK-015: Audit Log Service

### Phase F: Hardening
16. TASK-016: Integration Tests
17. TASK-017: Load Tests
18. TASK-018: Security Review
19. TASK-019: Deployment

## Luật ưu tiên
- Nếu task hiện tại cần ERD hoặc migration, hoàn thành TASK-002 trước.
- Nếu task hiện tại cần broker event contract, hoàn thành TASK-003 trước.
- Nếu task hiện tại cần auth hoặc user context, hoàn thành TASK-005 và TASK-006 trước.
- Nếu task hiện tại cần market data stream, hoàn thành TASK-007 trước.

## Checklist hoàn thành task
Một task được xem là hoàn thành khi:
- [ ] Tất cả file/folder deliverables đã được tạo.
- [ ] Checklist SRS/PRD đạt 100%.
- [ ] Test quan trọng pass.
- [ ] Không còn blocker mở.
- [ ] Task folder README thể hiện trạng thái `done`.

## Cách AI nên làm việc
- Luôn bắt đầu từ task đầu tiên chưa xong.
- Làm task theo chiều dọc: code -> test -> update docs.
- Nếu gặp chỗ chưa rõ, ưu tiên tạo skeleton và ghi `blocked` thay vì đoán bừa.
- Khi task hoàn tất, chuyển sang task kế tiếp trong danh sách.

## Gợi ý trạng thái tổng quan
- `TASK-001` và `TASK-002`: nền móng, làm trước.
- `TASK-003` và `TASK-004`: hạ tầng giao tiếp.
- `TASK-005` và `TASK-006`: bảo mật và user context.
- `TASK-007` và `TASK-008`: logic giao dịch.

## Ghi chú thực thi
- Không xử lý song song nhiều task nếu chúng chia sẻ schema hoặc contract event.
- Không đổi naming convention giữa các task.
- Không phá vỡ backward compatibility nếu đã có consumer.
- Mọi task liên quan giao dịch tiền thật phải được review trước khi merge.
