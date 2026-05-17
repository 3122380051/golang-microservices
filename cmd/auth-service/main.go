package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authapp "github.com/3122380051/golang-microservices/internal/application/auth"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/database"
	"github.com/3122380051/golang-microservices/internal/infrastructure/repository"
	"github.com/3122380051/golang-microservices/internal/logger"
	authhttp "github.com/3122380051/golang-microservices/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appLogger := logger.New(cfg.LogLevel)
	appLogger.Info("starting auth service", "addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewUserRepository(db)
	service := authapp.NewService(
		repo,
		cfg.AuthJWTSecret,
		time.Duration(cfg.AuthAccessTokenTTLMinutes)*time.Minute,
		time.Duration(cfg.AuthRefreshTokenTTLHours)*time.Hour,
	)
	handler := authhttp.NewAuthHandler(service)

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
		fmt.Fprintf(os.Stderr, "auth server error: %v\n", err)
		os.Exit(1)
	}
}
