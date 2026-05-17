package transport

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Health & readiness
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)

	// API v1 stubs
	mux.HandleFunc("/api/v1/ping", server.pingHandler)

	// Fallback
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
	// Chain: request-id -> recovery -> logging -> rate-limit -> handler
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observability.RequestCount.Add(1)
		next.ServeHTTP(w, r)
	})
	h = s.rateLimitMiddleware(h)
	h = s.loggingMiddleware(h)
	h = s.recoveryMiddleware(h)
	h = s.requestIDMiddleware(h)
	return h
}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := fmt.Sprintf("%d", time.Now().UnixNano())
		w.Header().Set("X-Request-ID", reqID)
		r = r.WithContext(context.WithValue(r.Context(), "request_id", reqID))
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "error", rec)
				observability.RequestErrors.Add(1)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		dur := time.Since(start)
		reqID, _ := r.Context().Value("request_id").(string)
		s.logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"request_id", reqID,
			"duration_ms", dur.Milliseconds(),
		)
	})
}

// rateLimitMiddleware is a simple stub for rate limiting; it currently allows all
// requests but records a metric. Replace with a real limiter per requirements.
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: implement per-client token buckets or use external gateway
		next.ServeHTTP(w, r)
	})
}

func (s *Server) pingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"pong": "ok"})
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
