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

	"github.com/3122380051/golang-microservices/internal/application/execution"
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

	appLogger.Info("starting execution service", "version", "1.0.0")

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
		consumer = broker.NewKafkaConsumer(brokers, "execution-service", event.TopicOrderCreated, broker.DefaultDLQTopic, producer)
		defer consumer.Close()
	}

	// Initialize Binance exchange adapter
	exchangeAdapter := exchange.NewBinanceClient()

	// Initialize repository
	executionRepository := infrastructure.NewInMemoryExecutionRepository()

	// Initialize execution service
	executionService := execution.NewService(appLogger, executionRepository, producer, consumer, exchangeAdapter)

	// Start Kafka consumer loop for order.created events
	if consumer != nil {
		go executionService.ConsumeOrderCreated(ctx)
	}

	// Start reconciliation loop (polling for fills)
	go executionService.StartReconciliationLoop(ctx)

	mux := http.NewServeMux()
	executionHandler := httptransport.NewExecutionHandler(executionService, appLogger)
	executionHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"execution-service"}`)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"execution-service"}`)
	})

	server := &http.Server{
		Addr:    ":8087",
		Handler: mux,
	}

	go func() {
		appLogger.Info("http server starting", "addr", ":8087")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	appLogger.Info("shutting down execution service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("http server shutdown error", "error", err)
	}
	appLogger.Info("execution service stopped")
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
