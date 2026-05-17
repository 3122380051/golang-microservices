# Thiết kế chi tiết cho từng thành phần và các pattern sử dụng

## Mục tiêu tài liệu
Tài liệu này mô tả cách thiết kế từng thành phần trong hệ thống trading crypto bằng Go, vai trò của từng service, ranh giới trách nhiệm, luồng dữ liệu, và các pattern nên áp dụng để hệ thống dễ mở rộng, an toàn và dễ vận hành.

## Nguyên tắc chung
- Mỗi service có một trách nhiệm chính, một database riêng và một contract giao tiếp rõ ràng.
- Business logic không phụ thuộc trực tiếp vào SDK của sàn hoặc framework hạ tầng.
- Mọi lệnh giao dịch phải có idempotency key và correlation id.
- Giao tiếp nội bộ ưu tiên bất đồng bộ qua event, chỉ dùng sync call khi thật sự cần phản hồi tức thời.
- Các thao tác nhạy cảm phải có audit, trace và khả năng replay sự kiện.

## 1. API Gateway

### Vai trò
- Là cổng vào duy nhất của hệ thống cho web, mobile và admin.
- Xác thực request, kiểm tra quyền, giới hạn tốc độ và định tuyến tới service phù hợp.

### Trách nhiệm chi tiết
- Xác thực JWT hoặc session token.
- Rate limit theo IP, user và client app.
- Request validation ở mức biên.
- Aggregation cho các API đọc dữ liệu từ nhiều service.
- Bảo vệ hệ thống khỏi traffic bất thường.

### Pattern sử dụng
- **Gateway Pattern**: gom điểm vào, giảm số lượng endpoint public.
- **Rate Limiting**: chống spam và bảo vệ tài nguyên.
- **JWT Authentication**: xác thực stateless.
- **Backpressure/Throttling**: kiểm soát tải đầu vào.
- **BFF Pattern** nếu có nhiều client khác nhau như web, mobile, admin.

### Gợi ý Go implementation
- Router mỏng, không chứa business logic.
- Middleware cho auth, logging, tracing, timeout.
- Handler chỉ làm nhiệm vụ map request sang command/query.

## 2. Auth Service

### Vai trò
- Quản lý đăng nhập, refresh token, phân quyền và trạng thái phiên.

### Trách nhiệm chi tiết
- Đăng ký, đăng nhập, refresh token, logout.
- Cấp role và kiểm soát scope.
- Lưu token metadata để revoke khi cần.
- Ghi audit cho các hành vi nhạy cảm.

### Pattern sử dụng
- **Token-Based Authentication**: giảm state ở server.
- **RBAC**: phân quyền theo vai trò.
- **Session Revocation List**: hỗ trợ vô hiệu hóa token đã cấp.
- **Hashing** cho password bằng bcrypt/argon2.

### Gợi ý Go implementation
- Domain model cho user credential và token.
- Repository tách biệt để truy xuất metadata token.
- Service layer giữ logic đăng nhập và policy auth.

## 3. User Service

### Vai trò
- Quản lý hồ sơ người dùng, cấu hình cá nhân và trạng thái KYC.

### Trách nhiệm chi tiết
- CRUD thông tin profile.
- Cấu hình ngôn ngữ, múi giờ, kênh nhận thông báo.
- Lưu trạng thái xác minh danh tính nếu hệ thống yêu cầu.

### Pattern sử dụng
- **CRUD Service Pattern**: đơn giản hóa quản lý hồ sơ.
- **Validation**: kiểm tra đầu vào ở tầng application.
- **Preference Profile Pattern**: tách cấu hình người dùng khỏi logic giao dịch.

### Gợi ý Go implementation
- Có thể dùng query riêng cho read model nếu UI cần nhiều thông tin tổng hợp.
- Không để service này biết chi tiết đặt lệnh hay chiến lược.

## 4. Market Data Service

### Vai trò
- Thu thập và chuẩn hóa dữ liệu thị trường từ các sàn.

### Trách nhiệm chi tiết
- Kết nối websocket/REST tới từng exchange.
- Chuẩn hóa ticker, candle, order book, trade stream.
- Phát dữ liệu vào broker để các service khác tiêu thụ.
- Ghi cache giá gần nhất cho truy cập nhanh.

### Pattern sử dụng
- **Adapter Pattern**: ẩn khác biệt API của từng sàn.
- **Publisher-Subscriber**: phát stream dữ liệu tới nhiều consumer.
- **Cache-Aside**: cache giá gần nhất trong Redis.
- **Circuit Breaker**: ngắt kết nối tạm thời nếu sàn lỗi liên tiếp.
- **Retry with Jitter**: giảm bão retry khi mất kết nối.

### Gợi ý Go implementation
- Mỗi sàn một adapter nhỏ, có interface chung.
- Một pipeline ingest -> normalize -> publish.
- Tách luồng realtime và luồng snapshot định kỳ.

## 5. Strategy Service

### Vai trò
- Chạy chiến lược giao dịch và sinh tín hiệu.

### Trách nhiệm chi tiết
- Đọc market data và tính chỉ báo.
- Tạo signal như buy, sell, hold.
- Hỗ trợ backtest hoặc paper trading.
- Xuất tín hiệu có lý do, confidence và ngữ cảnh.

### Pattern sử dụng
- **Strategy Pattern**: mỗi chiến lược là một implementation riêng.
- **Event-Driven Processing**: phản ứng theo dữ liệu phát ra.
- **Rules Engine nhẹ** nếu muốn cấu hình chiến lược bằng rule.
- **Stateless Processor** nếu trạng thái được lưu bên ngoài.

### Gợi ý Go implementation
- Một interface chung như Evaluate(ctx, marketEvent) -> signal.
- Plugin hóa chiến lược để thêm mới mà không sửa core.
- Nên có chế độ dry-run để kiểm tra logic không đặt lệnh thật.

## 6. Risk Service

### Vai trò
- Kiểm tra rủi ro trước khi lệnh được chấp nhận.

### Trách nhiệm chi tiết
- Kiểm tra position size, exposure, leverage, max loss.
- Chặn lệnh nếu vượt hạn mức.
- Tính budget còn lại cho từng user/strategy.

### Pattern sử dụng
- **Policy Engine**: mỗi policy là một luật rủi ro độc lập.
- **Fail Fast**: từ chối sớm khi không đạt điều kiện.
- **Guard Clause**: mã nguồn rõ ràng, dễ audit.
- **Idempotent Decision**: cùng một signal phải ra cùng một quyết định trong cùng ngữ cảnh.

### Gợi ý Go implementation
- Mỗi policy là một struct riêng có interface Check(...).
- Kết quả nên trả về reason, severity và suggested max size.

## 7. Order Service

### Vai trò
- Quản lý vòng đời lệnh nội bộ của hệ thống.

### Trách nhiệm chi tiết
- Tạo order từ signal đã được duyệt.
- Theo dõi trạng thái: created, pending, submitted, partially_filled, filled, canceled, rejected.
- Lưu correlation id và idempotency key.

### Pattern sử dụng
- **State Machine**: mô hình hóa vòng đời order.
- **Command Pattern**: create/cancel/amend order.
- **Saga Orchestration**: điều phối các bước liên quan tới execution và portfolio.
- **Idempotency Key**: chống tạo lệnh trùng.

### Gợi ý Go implementation
- Dùng state transition rõ ràng, không cho phép nhảy trạng thái tùy tiện.
- Command handler chỉ cập nhật state qua domain service.

## 8. Execution Service

### Vai trò
- Gửi lệnh thật ra sàn và theo dõi kết quả thực thi.

### Trách nhiệm chi tiết
- Map order nội bộ sang payload exchange.
- Gửi lệnh qua adapter của từng sàn.
- Retry an toàn, timeout, reconcile trạng thái.
- Ghi nhận exchange_order_id và trade fill.

### Pattern sử dụng
- **Adapter Pattern**: gọi nhiều sàn qua interface thống nhất.
- **Retry + Timeout**: xử lý lỗi mạng và độ trễ.
- **Circuit Breaker**: tránh dồn lỗi sang sàn đang không ổn định.
- **Outbox Pattern**: bảo đảm event gửi ra broker không mất khi commit DB.
- **Saga Step**: một bước trong chuỗi giao dịch phân tán.

### Gợi ý Go implementation
- Phân tách command submission và status reconciliation.
- Nên có worker nền để poll trạng thái khi exchange không push event đầy đủ.

## 9. Exchange Adapter Service

### Vai trò
- Chuẩn hóa tích hợp từng sàn như Binance, Bybit, OKX.

### Trách nhiệm chi tiết
- Chuyển đổi request/response về contract nội bộ.
- Quản lý đặc thù mỗi sàn như precision, lot size, rate limit.
- Tách biệt logic thực thi khỏi business logic.

### Pattern sử dụng
- **Adapter Pattern**: cốt lõi của service này.
- **Factory Pattern**: tạo adapter theo exchange.
- **Interface Segregation**: tách interface cho spot, futures, account, market data.

### Gợi ý Go implementation
- Interface rõ ràng cho từng nhóm chức năng.
- Bộ normalize chung cho symbol, quantity, fee, status.

## 10. Portfolio Service

### Vai trò
- Giữ trạng thái tài sản, vị thế và PnL của người dùng.

### Trách nhiệm chi tiết
- Cập nhật balance, margin, realized/unrealized PnL.
- Duy trì positions theo từng symbol.
- Là nguồn đọc chính cho dashboard tài chính.

### Pattern sử dụng
- **Projection / Read Model**: tổng hợp dữ liệu từ order, execution và market data.
- **Event Sourcing nhẹ** nếu muốn tái dựng trạng thái từ event.
- **Eventual Consistency**: chấp nhận độ trễ nhỏ sau khi khớp lệnh.
- **Optimistic Locking**: tránh cập nhật chồng trạng thái vị thế.

### Gợi ý Go implementation
- Consumer event từ execution và market updates.
- Projection builder cập nhật bảng portfolio/positions.

## 11. Notification Service

### Vai trò
- Gửi thông báo cho người dùng qua nhiều kênh.

### Trách nhiệm chi tiết
- Xử lý email, Telegram, Slack, webhook.
- Template hóa nội dung theo loại event.
- Theo dõi trạng thái gửi thành công hay thất bại.

### Pattern sử dụng
- **Observer Pattern**: phản ứng theo event phát ra.
- **Template Method**: khung xử lý chung, mỗi kênh cài phần riêng.
- **Retry Queue**: gửi lại khi kênh ngoài lỗi tạm thời.
- **Fan-out**: một event có thể gửi qua nhiều kênh.

### Gợi ý Go implementation
- Channel adapter riêng cho từng provider.
- Có dead-letter queue cho thông báo lỗi nhiều lần.

## 12. Audit Log Service

### Vai trò
- Lưu dấu vết thao tác và phục vụ kiểm toán, truy vết sự cố.

### Trách nhiệm chi tiết
- Ghi login, logout, create order, cancel order, thay đổi cấu hình.
- Lưu trace id, metadata và user context.
- Hỗ trợ truy vấn theo thời gian, entity và user.

### Pattern sử dụng
- **Append-Only Log**: không sửa, chỉ ghi thêm.
- **Audit Trail**: phục vụ kiểm toán và forensics.
- **Immutable Record**: tính toàn vẹn dữ liệu cao.

### Gợi ý Go implementation
- Ghi log bất đồng bộ để không ảnh hưởng request chính.
- Metadata nên là JSON để linh hoạt.

## 13. Message Broker

### Vai trò
- Là xương sống event-driven của hệ thống.

### Trách nhiệm chi tiết
- Phân phối event giữa market data, strategy, risk, order, execution, portfolio và notification.
- Hỗ trợ decouple giữa producer và consumer.

### Pattern sử dụng
- **Publish-Subscribe**.
- **Competing Consumers** cho worker scale ngang.
- **Dead Letter Queue** cho message lỗi.
- **At-Least-Once Delivery** kết hợp idempotency ở consumer.

### Gợi ý Go implementation
- Định nghĩa topic rõ ràng theo domain.
- Version hóa schema event ngay từ đầu.

## 14. PostgreSQL

### Vai trò
- Lưu dữ liệu nghiệp vụ chính.

### Trách nhiệm chi tiết
- Lưu user, role, strategy, order, execution, portfolio, audit log.
- Hỗ trợ transaction mạnh cho nghiệp vụ cần tính nhất quán.

### Pattern sử dụng
- **Repository Pattern**: tách domain khỏi persistence.
- **Unit of Work** nếu một use case cần nhiều thay đổi trong cùng transaction.
- **Optimistic Concurrency Control** với version hoặc updated_at.

## 15. Redis

### Vai trò
- Cache, lock, session ngắn hạn và rate limiting.

### Pattern sử dụng
- **Cache-Aside** cho giá và dữ liệu đọc nhiều.
- **Distributed Lock** cho job hoặc critical section.
- **Rate Limiter** cho gateway và execution.
- **Short-Lived Session Store**.

## Pattern kiến trúc cấp hệ thống

### CQRS
- Tách luồng ghi và luồng đọc ở những khu vực cần tối ưu khác nhau.
- Phù hợp cho portfolio, order history và dashboard.

### Saga
- Dùng cho chuỗi nghiệp vụ nhiều bước như signal -> risk -> order -> execution -> portfolio.
- Giúp hệ thống chịu lỗi tốt hơn so với transaction phân tán.

### Outbox
- Ghi event ra bảng outbox cùng transaction với DB rồi worker mới publish.
- Tránh mất event khi service đã commit dữ liệu nhưng chưa kịp gửi message.

### Event Sourcing nhẹ
- Không nhất thiết áp dụng cho toàn hệ thống, chỉ nên dùng nơi cần trace đầy đủ như order lifecycle hoặc audit.

### Hexagonal / Clean Architecture
- Domain nằm trung tâm, hạ tầng ở ngoài.
- Giúp thay thế database, broker, exchange adapter mà ít ảnh hưởng business logic.

## Gợi ý cấu trúc code Go
```text
cmd/
  api-gateway/
  auth-service/
  user-service/
  market-data-service/
  strategy-service/
  risk-service/
  order-service/
  execution-service/
  exchange-adapter-service/
  portfolio-service/
  notification-service/
  audit-log-service/
internal/
  domain/
  application/
  infrastructure/
  transport/
  config/
  observability/
```

## Luồng xử lý khuyến nghị
1. Market Data Service nhận dữ liệu từ exchange adapter.
2. Strategy Service tạo signal.
3. Risk Service kiểm tra và phê duyệt.
4. Order Service tạo order nội bộ.
5. Execution Service gửi order sang sàn.
6. Portfolio Service cập nhật vị thế.
7. Notification Service thông báo cho người dùng.
8. Audit Log Service ghi lại dấu vết cho toàn bộ bước quan trọng.

## Kết luận
Thiết kế này ưu tiên tính tách biệt trách nhiệm, khả năng mở rộng, khả năng quan sát và độ an toàn của lệnh giao dịch. Nếu triển khai đúng các pattern trên, hệ thống sẽ dễ thay đổi sàn giao dịch, dễ thêm chiến lược mới và ít phụ thuộc vào hạ tầng cụ thể.
