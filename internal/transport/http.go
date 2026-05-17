package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
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
	rateLimiter *localRateLimiter
	jwtToken string
	timeout time.Duration
}

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyUserID ctxKey = "user_id"
)

type rateBucket struct {
	windowStart time.Time
	count       int
}

type localRateLimiter struct {
	mu      sync.Mutex
	byIP    map[string]rateBucket
	byUser  map[string]rateBucket
}

func newLocalRateLimiter() *localRateLimiter {
	return &localRateLimiter{
		byIP:   make(map[string]rateBucket),
		byUser: make(map[string]rateBucket),
	}
}

func (l *localRateLimiter) allow(ip, user string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !allowBucket(l.byIP, ip, now, 100) {
		return false
	}

	if user != "" && !allowBucket(l.byUser, user, now, 1000) {
		return false
	}

	return true
}

func allowBucket(store map[string]rateBucket, key string, now time.Time, limit int) bool {
	if key == "" {
		return true
	}

	bucket, ok := store[key]
	if !ok || now.Sub(bucket.windowStart) >= time.Minute {
		store[key] = rateBucket{windowStart: now, count: 1}
		return true
	}

	if bucket.count >= limit {
		return false
	}

	bucket.count++
	store[key] = bucket
	return true
}

// NewServer creates a new HTTP server with health endpoints.
func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	server := &Server{
		addr:   cfg.HTTPAddr,
		logger: logger,
		mux:    mux,
		rateLimiter: newLocalRateLimiter(),
		jwtToken: cfg.GatewayJWTToken,
		timeout: time.Duration(cfg.GatewayTimeoutSeconds) * time.Second,
	}

	// Health & readiness
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)

	// API v1 stubs
	mux.HandleFunc("/api/v1/ping", server.pingHandler)
	mux.Handle("/api/v1/profile", server.authMiddleware(http.HandlerFunc(server.profileHandler)))

	// OpenAPI / Docs stubs
	mux.HandleFunc("/openapi.json", server.openapiHandler)
	mux.HandleFunc("/docs", server.docsHandler)

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
	h = s.timeoutMiddleware(h)
	h = s.corsMiddleware(h)
	h = s.loggingMiddleware(h)
	h = s.recoveryMiddleware(h)
	h = s.requestIDMiddleware(h)
	return h
}

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := fmt.Sprintf("%d", time.Now().UnixNano())
		w.Header().Set("X-Request-ID", reqID)
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, reqID))
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
		reqID, _ := r.Context().Value(ctxKeyRequestID).(string)
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
		ip := clientIP(r.RemoteAddr)
		user := userFromAuthHeader(r.Header.Get("Authorization"))
		if !s.rateLimiter.allow(ip, user, time.Now()) {
			observability.RequestErrors.Add(1)
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeout := s.timeout
		if timeout <= 0 {
			timeout = 8 * time.Second
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			observability.RequestErrors.Add(1)
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token != s.jwtToken {
			observability.RequestErrors.Add(1)
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, "trader")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) pingHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"pong": "ok"})
}

func (s *Server) profileHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	writeJSON(w, http.StatusOK, map[string]string{
		"user": userID,
		"role": "trader",
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
	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) openapiHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"openapi": "3.0.3",
		"info": map[string]string{
			"title":   "Crypto Trading API Gateway",
			"version": "0.1.0",
		},
		"paths": map[string]any{
			"/health": map[string]any{"get": map[string]string{"summary": "health check"}},
			"/ready": map[string]any{"get": map[string]string{"summary": "readiness check"}},
			"/api/v1/ping": map[string]any{"get": map[string]string{"summary": "ping endpoint"}},
			"/api/v1/profile": map[string]any{"get": map[string]string{"summary": "authenticated profile"}},
		},
	})
}

func (s *Server) docsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"docs": "OpenAPI available at /openapi.json",
	})
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func userFromAuthHeader(auth string) string {
	parts := strings.SplitN(strings.TrimSpace(auth), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
