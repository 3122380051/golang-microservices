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

	"log/slog"

	"github.com/3122380051/golang-microservices/internal/application/risk"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/database"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
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

	appLogger.Info("starting risk service", "version", "1.0.0")

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
		consumer = broker.NewKafkaConsumer(brokers, "risk-service", "strategy.signal.generated", broker.DefaultDLQTopic, producer)
		defer consumer.Close()
	}

	riskPolicyRepo := infrastructure.NewInMemoryRiskPolicyRepository()
	portfolioCache := infrastructure.NewPortfolioCache()

	evaluator := risk.NewEvaluator(appLogger)
	riskService := risk.NewService(
		appLogger,
		evaluator,
		riskPolicyRepo,
		portfolioCache,
		producer,
		consumer,
	)

	initializeDefaultPolicies(riskPolicyRepo, appLogger)

	if consumer != nil {
		go riskService.ConsumeStrategySignals(ctx)
	}

	mux := http.NewServeMux()
	riskHandler := httptransport.NewRiskHandler(riskService, appLogger)
	riskHandler.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"risk-service"}`)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","service":"risk-service"}`)
	})

	server := &http.Server{
		Addr:    ":8085",
		Handler: mux,
	}

	go func() {
		appLogger.Info("http server starting", "addr", ":8085")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	appLogger.Info("shutting down risk service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("http server shutdown error", "error", err)
	}
	appLogger.Info("risk service stopped")
}

func initializeDefaultPolicies(repo domain.RiskPolicyRepository, logger *slog.Logger) {
	defaultPolicy := &domain.RiskPolicy{
		ID:              "default-policy",
		UserID:          "default",
		StrategyID:      "*",
		MaxPositionSize: 100000.0,
		MaxLeverage:     5.0,
		MaxDailyLoss:    -1000.0,
		MinMarginRatio:  0.5,
		MaxExposure:     500000.0,
		IsActive:        true,
	}
	if err := repo.Create(context.Background(), defaultPolicy); err != nil {
		logger.Error("failed to create default policy", "error", err)
	}
}

const shutdownTimeout = 30 // seconds

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
