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

	"github.com/3122380051/golang-microservices/internal/application/portfolio"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/database"
	"github.com/3122380051/golang-microservices/internal/domain/event"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/3122380051/golang-microservices/internal/infrastructure/exchange"
	"github.com/3122380051/golang-microservices/internal/logger"
	httptransport "github.com/3122380051/golang-microservices/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger.Info("starting portfolio service", "version", "1.0.0")

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	brokers := splitCSV(cfg.KafkaBrokers)

	var producer *broker.KafkaProducer
	if len(brokers) > 0 {
		producer = broker.NewKafkaProducer(brokers)
		defer producer.Close()
	}

	var consumer *broker.KafkaConsumer
	if len(brokers) > 0 && producer != nil {
		consumer = broker.NewKafkaConsumer(brokers, "portfolio-service", event.TopicExecutionFilled, broker.DefaultDLQTopic, producer)
		defer consumer.Close()
	}

	// Initialize repositories
	portfolioRepository := infrastructure.NewInMemoryPortfolioRepository()
	tradeRepository := infrastructure.NewInMemoryTradeResultRepository()

	// Initialize calculator
	calculator := portfolio.NewPnLCalculator(0.1) // 10% maintenance margin

	// Initialize price fetcher (use Binance client for now)
	priceFetcher := exchange.NewBinanceClient()

	// Initialize portfolio service
	portfolioService := portfolio.NewService(
		appLogger,
		portfolioRepository,
		tradeRepository,
		producer,
		consumer,
		calculator,
		priceFetcher,
	)

	// Start Kafka consumer loop for execution.filled events
	if consumer != nil {
		go portfolioService.ConsumeExecutionFilled(ctx)
	}

	mux := http.NewServeMux()
	portfolioHandler := httptransport.NewPortfolioHandler(portfolioService, appLogger)
	portfolioHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"portfolio-service"}`)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"portfolio-service"}`)
	})

	server := &http.Server{
		Addr:    ":8088",
		Handler: mux,
	}

	go func() {
		appLogger.Info("http server starting", "addr", ":8088")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	appLogger.Info("shutting down portfolio service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("http server shutdown error", "error", err)
	}
	appLogger.Info("portfolio service stopped")
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
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
