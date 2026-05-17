package http

import (
	"encoding/json"
	"net/http"
	"strings"

	userapp "github.com/3122380051/golang-microservices/internal/application/user"
)

// UserHandler provides HTTP endpoints for user profile and api key management.
type UserHandler struct {
	service *userapp.Service
}

func NewUserHandler(service *userapp.Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/users/me", h.me)
	mux.HandleFunc("/users/api-keys", h.apiKeys)
	mux.HandleFunc("/users/api-keys/", h.deleteAPIKey)
}

func (h *UserHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "user-service"})
}

func (h *UserHandler) me(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := h.service.GetProfile(r.Context(), userID)
		if err != nil {
			handleUserError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":                       profile.ID,
			"email":                    profile.Email,
			"full_name":                profile.FullName,
			"status":                   profile.Status,
			"roles":                    profile.Roles,
			"timezone":                 profile.Timezone,
			"language":                 profile.Language,
			"notification_preferences": profile.NotificationPreferences,
		})
	case http.MethodPut:
		var req struct {
			FullName                string         `json:"full_name"`
			Timezone                string         `json:"timezone"`
			Language                string         `json:"language"`
			NotificationPreferences map[string]any `json:"notification_preferences"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		updated, err := h.service.UpdateProfile(
			r.Context(),
			userID,
			req.FullName,
			req.Timezone,
			req.Language,
			req.NotificationPreferences,
		)
		if err != nil {
			handleUserError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":                       updated.ID,
			"email":                    updated.Email,
			"full_name":                updated.FullName,
			"status":                   updated.Status,
			"roles":                    updated.Roles,
			"timezone":                 updated.Timezone,
			"language":                 updated.Language,
			"notification_preferences": updated.NotificationPreferences,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *UserHandler) apiKeys(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req struct {
			Exchange  string `json:"exchange"`
			APIKey    string `json:"api_key"`
			APISecret string `json:"api_secret"`
			Label     string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		created, err := h.service.CreateAPIKey(r.Context(), userID, req.Exchange, req.APIKey, req.APISecret, req.Label)
		if err != nil {
			handleUserError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	case http.MethodGet:
		items, err := h.service.ListAPIKeys(r.Context(), userID)
		if err != nil {
			handleUserError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *UserHandler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := requestUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user id")
		return
	}

	apiKeyID := strings.TrimPrefix(r.URL.Path, "/users/api-keys/")
	if apiKeyID == "" || strings.Contains(apiKeyID, "/") {
		writeError(w, http.StatusBadRequest, "invalid api key id")
		return
	}

	if err := h.service.DeleteAPIKey(r.Context(), userID, apiKeyID); err != nil {
		handleUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func requestUserID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-User-ID"))
}

func handleUserError(w http.ResponseWriter, err error) {
	switch err {
	case userapp.ErrInvalidUserID:
		writeError(w, http.StatusUnauthorized, err.Error())
	case userapp.ErrInvalidInput:
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
