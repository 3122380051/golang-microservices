package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	authapp "github.com/3122380051/golang-microservices/internal/application/auth"
)

// AuthService defines the methods needed by HTTP handlers.
type AuthService interface {
	Register(ctx any, email, password string) (string, any, error)
}

// Handler wraps auth service HTTP endpoints.
type Handler struct {
	service *authapp.Service
}

func NewAuthHandler(service *authapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/auth/register", h.register)
	mux.HandleFunc("/auth/login", h.login)
	mux.HandleFunc("/auth/refresh", h.refresh)
	mux.HandleFunc("/auth/logout", h.logout)
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "auth-service"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	userID, tokens, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":       userID,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	userID, tokens, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	tokens, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	auth := r.Header.Get("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := h.service.Logout(r.Context(), strings.TrimSpace(parts[1])); err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, authapp.ErrInvalidEmail), errors.Is(err, authapp.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, authapp.ErrEmailAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, authapp.ErrInvalidCredential), errors.Is(err, authapp.ErrInvalidToken), errors.Is(err, authapp.ErrTokenRevoked):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, authapp.ErrTokenExpired):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
