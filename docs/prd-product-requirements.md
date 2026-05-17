# PRD - Product Requirements Document
## Hệ thống Trading Crypto Microservices

### 1. Tổng quan sản phẩm

#### 1.1 Định nghĩa
Xây dựng nền tảng trading crypto dùng chiến lược tự động, cho phép người dùng kết nối tài khoản sàn, cấu hình chiến lược giao dịch và tự động thực hiện lệnh.

#### 1.2 Mục tiêu kinh doanh
- Cung cấp nền tảng trading chuyên nghiệp với tính năng tự động và quản lý rủi ro.
- Hỗ trợ nhiều sàn (Binance, Bybit, OKX, v.v.) thông qua adapter thống nhất.
- Cộng đồng có thể chia sẻ và mua bán chiến lược.
- Tạo dòng thu từ commission trên chi phí giao dịch.

#### 1.3 Phạm vi MVP (6-8 tuần)
- Xác thực người dùng.
- Quản lý kết nối API key sàn.
- Dữ liệu thị trường từ 1-2 sàn lớn.
- Chiến lược giao dịch đơn giản (EMA cross, RSI, v.v.).
- Đặt lệnh và theo dõi tự động.
- Dashboard cơ bản xem lệnh và PnL.
- Ghi nhận mỗi lệnh để audit.

### 2. Người dùng và use case

#### 2.1 Loại người dùng
- **Trader cá nhân**: muốn tự động hóa chiến lược của họ.
- **Quản lý quỹ**: muốn cấu hình chiến lược cho nhiều tài khoản.
- **Admin nền tảng**: quản lý người dùng, monitor hệ thống.

#### 2.2 Use case chính
1. **Đăng nhập và bảo mật**
   - Người dùng tạo tài khoản dùng email.
   - Xác thực 2FA (nếu MVP có thời gian).
   - Lưu API key sàn được mã hóa.

2. **Quản lý chiến lược**
   - Tạo chiến lược mới với tham số tùy chỉnh.
   - Kích hoạt/vô hiệu hóa chiến lược.
   - Xem lịch sử signal được tạo.

3. **Giao dịch tự động**
   - Chiến lược tạo signal khi điều kiện thỏa mãn.
   - Hệ thống kiểm tra rủi ro.
   - Tự động đặt lệnh nếu hợp lệ.
   - Theo dõi trạng thái lệnh.

4. **Giám sát danh mục**
   - Xem PnL realized/unrealized.
   - Xem vị thế đang mở.
   - Lịch sử giao dịch.

### 3. Tính năng chính

#### 3.1 Tính năng giai đoạn 1 (MVP)
- **Authentication**: đăng nhập, đăng xuất, reset password.
- **API Key Management**: thêm, xóa, kiểm tra quyền API key.
- **Market Data**: theo dõi giá real-time từ Binance (ít nhất).
- **Strategy Engine**: tạo signal dùng EMA cross, RSI, Bollinger Bands.
- **Risk Management**: kiểm tra margin, position size, max loss.
- **Order Management**: tạo, theo dõi, hủy lệnh.
- **Portfolio Tracking**: xem PnL, vị thế, balance.
- **Notification**: thông báo lệnh được khớp qua email/Telegram.
- **Audit**: ghi nhận mỗi thao tác quan trọng.

#### 3.2 Tính năng giai đoạn 2 (Post-MVP)
- Backtest chiến lược.
- Paper trading.
- Chia sẻ/ mua bán chiến lược.
- Quản lý nhiều tài khoản.
- Advanced charting.
- Advanced risk scenarios.

### 4. Yêu cầu phi chức năng

#### 4.1 Hiệu suất
- API response < 200ms cho 99% request.
- Lệnh được gửi trong < 500ms từ khi signal tạo.
- Xử lý 1000 lệnh/ngày ban đầu.

#### 4.2 Độ tin cậy
- 99.5% uptime.
- Không mất dữ liệu giao dịch.
- Khả năng recover sau failure.

#### 4.3 Bảo mật
- Mã hóa API key, private key.
- HTTPS/TLS cho mọi giao tiếp.
- Rate limit để chống brute force.
- Audit trail đầy đủ.

#### 4.4 Khả năng mở rộng
- Kiến trúc microservices để dễ scale từng service.
- Message broker để decouple.
- Database read replica cho scaling read.

### 5. Ràng buộc và giả định

#### 5.1 Ràng buộc
- Dùng Go cho backend để hiệu suất cao.
- PostgreSQL cho lưu trữ chính.
- Kafka/NATS/RabbitMQ cho event.
- Redis cho cache và lock.

#### 5.2 Giả định
- Người dùng cung cấp API key hợp lệ từ sàn.
- Kết nối mạng tới sàn là ổn định (hệ thống xử lý timeout).
- Sàn hỗ trợ REST API và WebSocket.

### 6. Tiêu chí chấp nhận

#### 6.1 Đối với tính năng chính
- [ ] Người dùng có thể đăng nhập và quản lý tài khoản.
- [ ] Hệ thống lấy dữ liệu giá từ sàn trong < 2s.
- [ ] Chiến lược tạo signal khi điều kiện thỏa mãn.
- [ ] Lệnh được đặt < 1s từ khi approved.
- [ ] Dashboard hiển thị PnL chính xác.
- [ ] Tất cả lệnh được audit và có thể trace.

#### 6.2 Đối với hiệu năng
- [ ] Load test 100 user đồng thời không error.
- [ ] Market data latency < 100ms từ exchange.
- [ ] Order submission latency < 500ms.

#### 6.3 Đối với bảo mật
- [ ] Penetration test cơ bản pass.
- [ ] Không có hardcoded key trong code.
- [ ] API key mã hóa trong database.

### 7. Timeline và milestone

- **Tuần 1-2**: Setup hạ tầng, database schema, API gateway.
- **Tuần 3-4**: Auth, User, Market Data service.
- **Tuần 5-6**: Strategy, Risk, Order, Execution service.
- **Tuần 7**: Portfolio, Notification, Audit log.
- **Tuần 8**: Testing, deployment, documentation.

### 8. Success metrics

- Số người dùng đăng ký.
- Số lệnh được thực hiện.
- Tỷ lệ uptime.
- Latency trung bình.
- User retention.
