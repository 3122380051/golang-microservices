package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/3122380051/golang-microservices/internal/application/risk"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
)

// RiskHandler handles HTTP requests for risk service endpoints
type RiskHandler struct {
	riskService *risk.Service
	logger      infrastructure.Logger
}

// NewRiskHandler creates a new risk handler
func NewRiskHandler(riskService *risk.Service, logger infrastructure.Logger) *RiskHandler {
	return &RiskHandler{
		riskService: riskService,
		logger:      logger,
	}
}

// RegisterRoutes registers all risk service routes
func (h *RiskHandler) RegisterRoutes(server *http.ServeMux) {
	server.HandleFunc("GET /risk/policies", h.listPolicies)
	server.HandleFunc("GET /risk/policies/{id}", h.getPolicy)
	server.HandleFunc("POST /risk/policies", h.createPolicy)
	server.HandleFunc("PUT /risk/policies/{id}", h.updatePolicy)
	server.HandleFunc("DELETE /risk/policies/{id}", h.deletePolicy)
	server.HandleFunc("GET /risk/decisions/{id}", h.getDecision)
}

// listPolicies GET /risk/policies?user_id=...
func (h *RiskHandler) listPolicies(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	policies, err := h.riskService.ListUserPolicies(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"policies": policies,
		"count":    len(policies),
	})
}

// getPolicy GET /risk/policies/{id}
func (h *RiskHandler) getPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "policy id is required")
		return
	}

	policy, err := h.riskService.GetPolicy(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, policy)
}

// createPolicy POST /risk/policies
func (h *RiskHandler) createPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID          string  `json:"user_id"`
		StrategyID      string  `json:"strategy_id"`
		MaxPositionSize float64 `json:"max_position_size"`
		MaxLeverage     float64 `json:"max_leverage"`
		MaxDailyLoss    float64 `json:"max_daily_loss"`
		MinMarginRatio  float64 `json:"min_margin_ratio"`
		MaxExposure     float64 `json:"max_exposure"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	policy := &domain.RiskPolicy{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		StrategyID:      req.StrategyID,
		MaxPositionSize: req.MaxPositionSize,
		MaxLeverage:     req.MaxLeverage,
		MaxDailyLoss:    req.MaxDailyLoss,
		MinMarginRatio:  req.MinMarginRatio,
		MaxExposure:     req.MaxExposure,
		IsActive:        true,
	}

	if err := h.riskService.CreatePolicy(r.Context(), policy); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, policy)
}

// updatePolicy PUT /risk/policies/{id}
func (h *RiskHandler) updatePolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "policy id is required")
		return
	}

	// Get existing policy
	policy, err := h.riskService.GetPolicy(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		MaxPositionSize *float64 `json:"max_position_size"`
		MaxLeverage     *float64 `json:"max_leverage"`
		MaxDailyLoss    *float64 `json:"max_daily_loss"`
		MinMarginRatio  *float64 `json:"min_margin_ratio"`
		MaxExposure     *float64 `json:"max_exposure"`
		IsActive        *bool    `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields if provided
	if req.MaxPositionSize != nil {
		policy.MaxPositionSize = *req.MaxPositionSize
	}
	if req.MaxLeverage != nil {
		policy.MaxLeverage = *req.MaxLeverage
	}
	if req.MaxDailyLoss != nil {
		policy.MaxDailyLoss = *req.MaxDailyLoss
	}
	if req.MinMarginRatio != nil {
		policy.MinMarginRatio = *req.MinMarginRatio
	}
	if req.MaxExposure != nil {
		policy.MaxExposure = *req.MaxExposure
	}
	if req.IsActive != nil {
		policy.IsActive = *req.IsActive
	}

	if err := h.riskService.UpdatePolicy(r.Context(), policy); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, policy)
}

// deletePolicy DELETE /risk/policies/{id}
func (h *RiskHandler) deletePolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "policy id is required")
		return
	}

	if err := h.riskService.DeletePolicy(r.Context(), id); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// getDecision GET /risk/decisions/{id}
func (h *RiskHandler) getDecision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "decision id is required")
		return
	}

	decision, err := h.riskService.GetDecision(id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, decision)
}

// Helper functions

func (h *RiskHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *RiskHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  status,
	})
}
