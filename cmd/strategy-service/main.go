package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	strategyapp "github.com/3122380051/golang-microservices/internal/application/strategy"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/domain/event"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/3122380051/golang-microservices/internal/infrastructure/repository"
	"github.com/3122380051/golang-microservices/internal/logger"
	httptransport "github.com/3122380051/golang-microservices/internal/transport/http"
	"github.com/segmentio/kafka-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel)
	appLogger.Info("starting strategy service", "addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := repository.NewStrategyRepository()
	brokers := splitCSV(cfg.KafkaBrokers)
	var publisher broker.Publisher
	var producer *broker.KafkaProducer
	if len(brokers) > 0 {
		producer = broker.NewKafkaProducer(brokers)
		publisher = producer
		defer producer.Close()
	}

	service := strategyapp.NewService(repo, publisher)
	handler := httptransport.NewStrategyHandler(service)

	if len(brokers) > 0 {
		go consumeMarketEvents(ctx, service, producer, brokers)
	}

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

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
		fmt.Fprintf(os.Stderr, "strategy server error: %v\n", err)
		os.Exit(1)
	}
}

func consumeMarketEvents(ctx context.Context, service *strategyapp.Service, producer *broker.KafkaProducer, brokers []string) {
	if len(brokers) == 0 {
		return
	}
	consumer := broker.NewKafkaConsumer(brokers, "strategy-service", event.TopicMarketPriceUpdated, broker.DefaultDLQTopic, producer)
	defer consumer.Close()
	_ = consumer.Consume(ctx, func(ctx context.Context, msg kafka.Message) error {
		var envelope event.Envelope
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			return err
		}
		_, err := service.HandleMarketPriceUpdated(ctx, envelope)
		return err
	})
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
