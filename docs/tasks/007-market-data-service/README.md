# Task 007: Market Data Service

## Mô tả
Implement Market Data Service: kết nối sàn (Binance), lấy dữ liệu real-time (ticker, candle, order book), normalize, cache, phát event.

## SRS - Requirements
- [ ] Exchange adapter interface: GetTicker, GetCandles, GetOrderBook.
- [ ] Binance REST client: replicateTicker, candles.
- [ ] Binance WebSocket client: stream ticker, trades.
- [ ] Data normalization: convert Binance format -> internal model.
- [ ] Market symbols: store list (BTCUSDT, ETHUSDT, v.v.).
- [ ] Cache: price in Redis (TTL 10s), candle in TimescaleDB.
- [ ] Event publisher: market.price.updated, market.candle.created.
- [ ] Error handling: retry, circuit breaker, timeout.

## PRD - Acceptance Criteria
- [ ] GET /market/price?symbol=BTCUSDT -> {price, bid, ask, ts}.
- [ ] GET /market/candles?symbol=BTCUSDT&interval=1h&limit=100 -> list candles.
- [ ] GET /market/order-book?symbol=BTCUSDT -> {bids, asks}.
- [ ] WebSocket stream /ws/market -> tick by tick price.
- [ ] Cache hit rate > 80% cho price query.
- [ ] Latency P99 < 100ms.

## Deliverables
- [ ] ✅ cmd/market-data-service/main.go
- [ ] ✅ internal/domain/market.go, candle.go
- [ ] ✅ internal/application/market/service.go
- [ ] ✅ internal/infrastructure/exchange/adapter.go
- [ ] ✅ internal/infrastructure/exchange/binance.go
- [ ] ✅ internal/infrastructure/cache/market_cache.go
- [ ] ✅ internal/transport/http/market_handler.go
- [ ] ✅ internal/transport/ws/market_stream.go
- [ ] ✅ tests/market_service_test.go, exchange_adapter_test.go

## Effort
8h (Backend 4)

## Timeline
Ngày 5-6
