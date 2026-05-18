# Task 003: Message Broker Setup (Kafka/NATS)

## Mô tả
Setup Kafka hoặc NATS trong Docker Compose, define topic/subject names, create producer/consumer clients, setup consumer groups, dead-letter queue handling.

## SRS - Requirements
- [ ] Kafka 2.8+ hoặc NATS 2.0+ trong Docker Compose.
- [ ] Topics/Subjects: market.*, strategy.*, risk.*, order.*, execution.*, portfolio.*, notification.*.
- [ ] Version hóa schema event (v1, v2).
- [ ] Producer: KafkaProducer hoặc NatsConn abstraction.
- [ ] Consumer: consumer group, offset tracking, retry/DLQ.
- [ ] Partitioning: theo user_id hoặc symbol để scale.
- [ ] At-least-once delivery guarantee.

## PRD - Acceptance Criteria
- [ ] Chạy `docker-compose up` broker lên without error.
- [ ] Produce message tới topic -> consume được message.
- [ ] Consumer group tracking offset -> restart consumer không duplicate.
- [ ] Dead-letter queue (DLQ) catch invalid message.
- [ ] Multi-partition topic: producer distribute balanced.

## Deliverables
- [x] ✅ Docker Compose config cho Kafka/NATS.
- [x] ✅ internal/infrastructure/broker/producer.go
- [x] ✅ internal/infrastructure/broker/consumer.go
- [x] ✅ internal/domain/event/market_event.go (event types)
- [x] ✅ scripts/kafka-topics.sh hoặc NATS config
- [x] ✅ tests/broker_test.go (integration test)

## Implementation Notes
- Dùng Kafka để thống nhất với compose hiện tại.
- Producer/consumer dùng `segmentio/kafka-go`.
- Consumer có DLQ publish khi handler trả lỗi và vẫn commit offset để tránh loop message xấu.
- Event schema được bọc trong envelope version hóa.

## Effort
3h (DevOps/Backend Lead)

## Timeline
Ngày 2 sáng

## Status
✅ **COMPLETED** - Kafka broker fully operational
- Kafka 3.x configured in Docker Compose
- 8 topics created (market.price.updated, strategy.signal.generated, risk.order.approved, order.created, execution.submitted, portfolio.updated, notification.send.requested, dead-letter)
- Producer/consumer clients implemented with segmentio/kafka-go
- Event envelope with version hóa schema (v1)
- Consumer group offset tracking and DLQ handling
- Integration tests passing
