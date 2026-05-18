package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/3122380051/golang-microservices/internal/application/order"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/3122380051/golang-microservices/internal/transport"
)

func main() {
	// Initialize config
	cfg := infrastructure.LoadConfig()

	// Initialize logger
	logger := infrastructure.NewLogger(cfg)
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
		}
	}()

	logger.Info("starting order service", "version", "1.0.0")

	// Initialize database
	db, err := infrastructure.NewDatabase(cfg)
	if err != nil {
		logger.Fatal("failed to initialize database", "error", err)
	}
	defer db.Close()

	// Initialize Kafka producer & consumer
	producer, err := broker.NewProducer(cfg)
	if err != nil {
		logger.Fatal("failed to initialize kafka producer", "error", err)
	}
	defer producer.Close()

	consumer, err := broker.NewConsumer(cfg, "risk.order.approved", "risk.order.rejected")
	if err != nil {
		logger.Fatal("failed to initialize kafka consumer", "error", err)
	}
	defer consumer.Close()

	// Initialize repository
	orderRepository := infrastructure.NewInMemoryOrderRepository()

	// Initialize order service
	orderService := order.NewService(
		logger,
		orderRepository,
		producer,
		consumer,
	)

	// Start Kafka consumer loop in background
	go orderService.ConsumeRiskDecisions(context.Background())

	// Initialize HTTP transport
	httpServer := transport.NewHTTPServer(cfg, logger)

	// Register order endpoints
	orderHandler := transport.NewOrderHandler(orderService, logger)
	orderHandler.RegisterRoutes(httpServer)

	// Health check endpoints
	httpServer.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"order-service"}`)
	})

	httpServer.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"order-service"}`)
	})

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.OrderServicePort)
		logger.Info("http server starting", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", "error", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down order service")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	logger.Info("order service stopped")
}

const shutdownTimeout = 30 // seconds
