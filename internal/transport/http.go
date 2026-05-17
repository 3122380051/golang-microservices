package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/observability"
	"log/slog"
)

// Server is a minimal HTTP server used by the gateway.
type Server struct {
	addr   string
	logger *slog.Logger
	mux    *http.ServeMux
	server *http.Server
}

// NewServer creates a new HTTP server with health endpoints.
func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	server := &Server{
		addr:   cfg.HTTPAddr,
		logger: logger,
		mux:    mux,
	}

	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)
	mux.HandleFunc("/", server.notFoundHandler)

	server.server = &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      server.withMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting http server", "addr", s.addr)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()
	return s.server.ListenAndServe()
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observability.RequestCount.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(observability.RequestCount.String()))
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		writeJSON(w, http.StatusOK, map[string]string{"service": "api-gateway"})
		return
	}
	observability.RequestErrors.Add(1)
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
