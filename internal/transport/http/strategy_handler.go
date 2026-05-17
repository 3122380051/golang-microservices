package http

import (
	"encoding/json"
	"net/http"
	"strings"

	strategyapp "github.com/3122380051/golang-microservices/internal/application/strategy"
	"github.com/3122380051/golang-microservices/internal/domain"
)

// StrategyHandler exposes CRUD endpoints for strategies.
type StrategyHandler struct {
	service *strategyapp.Service
}

func NewStrategyHandler(service *strategyapp.Service) *StrategyHandler {
	return &StrategyHandler{service: service}
}

func (h *StrategyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/strategies", h.strategies)
	mux.HandleFunc("/strategies/", h.strategyByID)
}

func (h *StrategyHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "strategy-service"})
}

func (h *StrategyHandler) strategies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name   string              `json:"name"`
			Symbol string              `json:"symbol"`
			Type   domain.StrategyType `json:"type"`
			Active bool                `json:"active"`
			Config json.RawMessage     `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		created, err := h.service.CreateStrategy(r.Context(), domain.Strategy{
			Name:   strings.TrimSpace(req.Name),
			Symbol: strings.TrimSpace(req.Symbol),
			Type:   req.Type,
			Active: req.Active,
			Config: req.Config,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, created)
	case http.MethodGet:
		items, err := h.service.ListStrategies(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *StrategyHandler) strategyByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/strategies/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "invalid strategy id")
		return
	}
	strategyID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		item, err := h.service.GetStrategy(r.Context(), strategyID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	if len(parts) == 2 && parts[1] == "activate" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		item, err := h.service.Activate(r.Context(), strategyID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	if len(parts) == 2 && parts[1] == "deactivate" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		item, err := h.service.Deactivate(r.Context(), strategyID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, item)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}
