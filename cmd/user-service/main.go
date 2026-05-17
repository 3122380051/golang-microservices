package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	userapp "github.com/3122380051/golang-microservices/internal/application/user"
	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/database"
	cryptox "github.com/3122380051/golang-microservices/internal/infrastructure/crypto"
	"github.com/3122380051/golang-microservices/internal/infrastructure/repository"
	"github.com/3122380051/golang-microservices/internal/logger"
	userhttp "github.com/3122380051/golang-microservices/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel)
	appLogger.Info("starting user service", "addr", cfg.HTTPAddr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	encKey := cfg.UserAPIKeyEncryptionKey
	if encKey == "" {
		// Default dev key for local run only.
		encKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	}
	encryptor, err := cryptox.NewEncryptorFromBase64(encKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create encryptor: %v\n", err)
		os.Exit(1)
	}

	repo := repository.NewUserRepository(db)
	service := userapp.NewService(repo, encryptor)
	handler := userhttp.NewUserHandler(service)

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
		fmt.Fprintf(os.Stderr, "user server error: %v\n", err)
		os.Exit(1)
	}
}
