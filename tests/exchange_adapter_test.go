package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
)

func TestBinanceAdapterNormalization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/ticker/bookTicker", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","bidPrice":"100.10","askPrice":"100.30"}`))
	})
	mux.HandleFunc("/api/v3/klines", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[[1700000000000,"100","110","90","105","1000",1700003600000]]`))
	})
	mux.HandleFunc("/api/v3/depth", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"bids":[["100.1","1.2"]],"asks":[["100.3","1.1"]]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := exchange.NewBinanceClientWithBaseURL(server.URL, 2*time.Second)
	ctx := context.Background()

	price, err := client.GetTicker(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("GetTicker: %v", err)
	}
	if price.Symbol != "BTCUSDT" || price.Price <= 0 {
		t.Fatalf("unexpected ticker normalization: %+v", price)
	}

	candles, err := client.GetCandles(ctx, "BTCUSDT", "1h", 1)
	if err != nil {
		t.Fatalf("GetCandles: %v", err)
	}
	if len(candles) != 1 || candles[0].Close != 105 {
		t.Fatalf("unexpected candle normalization: %+v", candles)
	}

	orderBook, err := client.GetOrderBook(ctx, "BTCUSDT", 5)
	if err != nil {
		t.Fatalf("GetOrderBook: %v", err)
	}
	if len(orderBook.Bids) != 1 || len(orderBook.Asks) != 1 {
		t.Fatalf("unexpected orderbook normalization: %+v", orderBook)
	}
}
