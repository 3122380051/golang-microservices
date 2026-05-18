package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/3122380051/golang-microservices/internal/application/risk"
	"github.com/3122380051/golang-microservices/internal/domain"
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

	logger.Info("starting risk service", "version", "1.0.0")

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

	consumer, err := broker.NewConsumer(cfg, "strategy.signal.generated")
	if err != nil {
		logger.Fatal("failed to initialize kafka consumer", "error", err)
	}
	defer consumer.Close()

	// Initialize repositories and caches
	riskPolicyRepo := infrastructure.NewInMemoryRiskPolicyRepository()
	portfolioCache := infrastructure.NewPortfolioCache()

	// Initialize risk service
	evaluator := risk.NewEvaluator(logger)
	riskService := risk.NewService(
		logger,
		evaluator,
		riskPolicyRepo,
		portfolioCache,
		producer,
		consumer,
	)

	// Initialize default policies
	initializeDefaultPolicies(riskPolicyRepo, logger)

	// Start Kafka consumer loop in background
	go riskService.ConsumeStrategySignals(context.Background())

	// Initialize HTTP transport
	httpServer := transport.NewHTTPServer(cfg, logger)

	// Register risk endpoints
	riskHandler := transport.NewRiskHandler(riskService, logger)
	riskHandler.RegisterRoutes(httpServer)

	// Health check endpoints
	httpServer.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"risk-service"}`)
	})

	httpServer.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"risk-service"}`)
	})

	// Start HTTP server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.RiskServicePort)
		logger.Info("http server starting", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", "error", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down risk service")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	logger.Info("risk service stopped")
}

func initializeDefaultPolicies(repo domain.RiskPolicyRepository, logger infrastructure.Logger) {
	defaultPolicy := &domain.RiskPolicy{
		ID:               "default-policy",
		UserID:           "default",
		StrategyID:       "*",
		MaxPositionSize:  100000.0,  // USD
		MaxLeverage:      5.0,
		MaxDailyLoss:     -1000.0,   // USD
		MinMarginRatio:   0.5,        // 50%
		MaxExposure:      500000.0,   // USD
		IsActive:         true,
	}
	if err := repo.Create(context.Background(), defaultPolicy); err != nil {
		logger.Error("failed to create default policy", "error", err)
	}
}

const shutdownTimeout = 30 // seconds
