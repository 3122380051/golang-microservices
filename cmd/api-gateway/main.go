package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/logger"
	"github.com/3122380051/golang-microservices/internal/transport"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appLogger := logger.New(cfg.LogLevel)
	appLogger.Info("configuration loaded", "app_name", cfg.AppName, "env", cfg.AppEnv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := transport.NewServer(cfg, appLogger)
	if err := server.Run(ctx); err != nil && err.Error() != "http: Server closed" {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
