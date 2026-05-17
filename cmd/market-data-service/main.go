package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	marketapp "github.com/3122380051/golang-microservices/internal/application/market"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/3122380051/golang-microservices/internal/infrastructure/cache"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
	"github.com/3122380051/golang-microservices/internal/logger"
	httptransport "github.com/3122380051/golang-microservices/internal/transport/http"
	wstransport "github.com/3122380051/golang-microservices/internal/transport/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel)
	appLogger.Info("starting market data service", "addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	adapter := exchange.NewBinanceClient()
	cacheStore := cache.NewMarketCache()

	var publisher broker.Publisher
	brokers := splitCSV(cfg.KafkaBrokers)
	if len(brokers) > 0 {
		publisher = broker.NewKafkaProducer(brokers)
		defer publisher.Close()
	}

	service := marketapp.NewService(adapter, cacheStore, publisher)

	// Polling loop for websocket pushes.
	go service.StartPolling(ctx, "BTCUSDT", time.Second)

	handler := httptransport.NewMarketHandler(service)
	wsHandler := wstransport.NewMarketStreamHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	mux.Handle("/ws/market", wsHandler)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "market data server error: %v\n", err)
		os.Exit(1)
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
