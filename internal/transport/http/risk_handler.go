package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// RiskService defines the methods needed by HTTP handlers for risk management.
type RiskService interface {
	GetDecision(id string) (*domain.RiskDecision, error)
	ListDecisions(ctx context.Context, userID string) ([]*domain.RiskDecision, error)
	CreatePolicy(ctx context.Context, policy *domain.RiskPolicy) error
	GetPolicy(ctx context.Context, policyID string) (*domain.RiskPolicy, error)
	UpdatePolicy(ctx context.Context, policy *domain.RiskPolicy) error
	DeletePolicy(ctx context.Context, policyID string) error
	ListPolicies(ctx context.Context, userID string) ([]*domain.RiskPolicy, error)
}

// RiskHandler wraps risk service HTTP endpoints.
type RiskHandler struct {
	service RiskService
	logger  *slog.Logger
}

// NewRiskHandler creates a new risk handler
func NewRiskHandler(service RiskService, logger *slog.Logger) *RiskHandler {
	return &RiskHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers risk endpoints
func (h *RiskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /risk/decisions", h.listDecisions)
	mux.HandleFunc("GET /risk/decisions/{id}", h.getDecision)
	mux.HandleFunc("GET /risk/policies", h.listPolicies)
	mux.HandleFunc("POST /risk/policies", h.createPolicy)
	mux.HandleFunc("GET /risk/policies/{id}", h.getPolicy)
	mux.HandleFunc("PUT /risk/policies/{id}", h.updatePolicy)
	mux.HandleFunc("DELETE /risk/policies/{id}", h.deletePolicy)
}

type riskPolicyRequest struct {
	UserID          string  `json:"user_id"`
	StrategyID      string  `json:"strategy_id"`
	MaxPositionSize float64 `json:"max_position_size"`
	MaxLeverage     float64 `json:"max_leverage"`
	MinMarginRatio  float64 `json:"min_margin_ratio"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxExposure     float64 `json:"max_exposure"`
}

type riskPolicyResponse struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	StrategyID      string  `json:"strategy_id"`
	MaxPositionSize float64 `json:"max_position_size"`
	MaxLeverage     float64 `json:"max_leverage"`
	MinMarginRatio  float64 `json:"min_margin_ratio"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxExposure     float64 `json:"max_exposure"`
	IsActive        bool    `json:"is_active"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type riskDecisionResponse struct {
	ID              string                 `json:"id"`
	SignalID        string                 `json:"signal_id"`
	UserID          string                 `json:"user_id"`
	StrategyID      string                 `json:"strategy_id"`
	Symbol          string                 `json:"symbol"`
	Side            string                 `json:"side"`
	Quantity        float64                `json:"quantity"`
	EstimatedPrice  float64                `json:"estimated_price"`
	IsApproved      bool                   `json:"is_approved"`
	RejectionReason string                 `json:"rejection_reason"`
	Checks          map[string]interface{} `json:"checks"`
	TraceID         string                 `json:"trace_id"`
	DecidedAt       string                 `json:"decided_at"`
	CreatedAt       string                 `json:"created_at"`
}

func (h *RiskHandler) createPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req riskPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	policy := &domain.RiskPolicy{
		UserID:          req.UserID,
		StrategyID:      req.StrategyID,
		MaxPositionSize: req.MaxPositionSize,
		MaxLeverage:     req.MaxLeverage,
		MinMarginRatio:  req.MinMarginRatio,
		MaxDailyLoss:    req.MaxDailyLoss,
		MaxExposure:     req.MaxExposure,
		IsActive:        true,
	}

	if err := h.service.CreatePolicy(r.Context(), policy); err != nil {
		h.logger.Error("failed to create policy", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create policy")
		return
	}

	resp := riskPolicyResponse{
		ID:              policy.ID,
		UserID:          policy.UserID,
		StrategyID:      policy.StrategyID,
		MaxPositionSize: policy.MaxPositionSize,
		MaxLeverage:     policy.MaxLeverage,
		MinMarginRatio:  policy.MinMarginRatio,
		MaxDailyLoss:    policy.MaxDailyLoss,
		MaxExposure:     policy.MaxExposure,
		IsActive:        policy.IsActive,
		CreatedAt:       policy.CreatedAt.String(),
		UpdatedAt:       policy.UpdatedAt.String(),
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *RiskHandler) getPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := r.PathValue("id")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	policy, err := h.service.GetPolicy(r.Context(), policyID)
	if err != nil {
		h.logger.Error("failed to get policy", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get policy")
		return
	}

	if policy == nil {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}

	resp := riskPolicyResponse{
		ID:              policy.ID,
		UserID:          policy.UserID,
		StrategyID:      policy.StrategyID,
		MaxPositionSize: policy.MaxPositionSize,
		MaxLeverage:     policy.MaxLeverage,
		MinMarginRatio:  policy.MinMarginRatio,
		MaxDailyLoss:    policy.MaxDailyLoss,
		MaxExposure:     policy.MaxExposure,
		IsActive:        policy.IsActive,
		CreatedAt:       policy.CreatedAt.String(),
		UpdatedAt:       policy.UpdatedAt.String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RiskHandler) updatePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := r.PathValue("id")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	var req riskPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	policy := &domain.RiskPolicy{
		ID:              policyID,
		UserID:          req.UserID,
		StrategyID:      req.StrategyID,
		MaxPositionSize: req.MaxPositionSize,
		MaxLeverage:     req.MaxLeverage,
		MinMarginRatio:  req.MinMarginRatio,
		MaxDailyLoss:    req.MaxDailyLoss,
		MaxExposure:     req.MaxExposure,
		IsActive:        true,
	}

	if err := h.service.UpdatePolicy(r.Context(), policy); err != nil {
		h.logger.Error("failed to update policy", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update policy")
		return
	}

	resp := riskPolicyResponse{
		ID:              policy.ID,
		UserID:          policy.UserID,
		StrategyID:      policy.StrategyID,
		MaxPositionSize: policy.MaxPositionSize,
		MaxLeverage:     policy.MaxLeverage,
		MinMarginRatio:  policy.MinMarginRatio,
		MaxDailyLoss:    policy.MaxDailyLoss,
		MaxExposure:     policy.MaxExposure,
		IsActive:        policy.IsActive,
		CreatedAt:       policy.CreatedAt.String(),
		UpdatedAt:       policy.UpdatedAt.String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RiskHandler) deletePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	policyID := r.PathValue("id")
	if policyID == "" {
		writeError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	if err := h.service.DeletePolicy(r.Context(), policyID); err != nil {
		h.logger.Error("failed to delete policy", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete policy")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "policy_id": policyID})
}

func (h *RiskHandler) listPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	policies, err := h.service.ListPolicies(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list policies", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list policies")
		return
	}

	responses := make([]riskPolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = riskPolicyResponse{
			ID:              policy.ID,
			UserID:          policy.UserID,
			StrategyID:      policy.StrategyID,
			MaxPositionSize: policy.MaxPositionSize,
			MaxLeverage:     policy.MaxLeverage,
			MinMarginRatio:  policy.MinMarginRatio,
			MaxDailyLoss:    policy.MaxDailyLoss,
			MaxExposure:     policy.MaxExposure,
			IsActive:        policy.IsActive,
			CreatedAt:       policy.CreatedAt.String(),
			UpdatedAt:       policy.UpdatedAt.String(),
		}
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h *RiskHandler) getDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	decisionID := r.PathValue("id")
	if decisionID == "" {
		writeError(w, http.StatusBadRequest, "decision_id is required")
		return
	}

	decision, err := h.service.GetDecision(decisionID)
	if err != nil {
		h.logger.Error("failed to get decision", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get decision")
		return
	}

	if decision == nil {
		writeError(w, http.StatusNotFound, "decision not found")
		return
	}

	checks := formatChecks(&decision.Checks)

	resp := riskDecisionResponse{
		ID:              decision.ID,
		SignalID:        decision.SignalID,
		UserID:          decision.UserID,
		StrategyID:      decision.StrategyID,
		Symbol:          decision.Symbol,
		Side:            decision.Side,
		Quantity:        decision.Quantity,
		EstimatedPrice:  decision.EstimatedPrice,
		IsApproved:      decision.IsApproved,
		RejectionReason: decision.RejectionReason,
		Checks:          checks,
		TraceID:         decision.TraceID,
		DecidedAt:       decision.DecidedAt.String(),
		CreatedAt:       decision.CreatedAt.String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RiskHandler) listDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	decisions, err := h.service.ListDecisions(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list decisions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list decisions")
		return
	}

	responses := make([]riskDecisionResponse, len(decisions))
	for i, decision := range decisions {
		checks := formatChecks(&decision.Checks)

		responses[i] = riskDecisionResponse{
			ID:              decision.ID,
			SignalID:        decision.SignalID,
			UserID:          decision.UserID,
			StrategyID:      decision.StrategyID,
			Symbol:          decision.Symbol,
			Side:            decision.Side,
			Quantity:        decision.Quantity,
			EstimatedPrice:  decision.EstimatedPrice,
			IsApproved:      decision.IsApproved,
			RejectionReason: decision.RejectionReason,
			Checks:          checks,
			TraceID:         decision.TraceID,
			DecidedAt:       decision.DecidedAt.String(),
			CreatedAt:       decision.CreatedAt.String(),
		}
	}

	writeJSON(w, http.StatusOK, responses)
}

// formatChecks converts ChecksDetail to a readable map format
func formatChecks(checks *domain.ChecksDetail) map[string]interface{} {
	if checks == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"position_size": map[string]interface{}{
			"passed": checks.PositionSizeCheck.Passed,
			"reason": checks.PositionSizeCheck.Reason,
			"value":  checks.PositionSizeCheck.Value,
		},
		"leverage": map[string]interface{}{
			"passed": checks.LeverageCheck.Passed,
			"reason": checks.LeverageCheck.Reason,
			"value":  checks.LeverageCheck.Value,
		},
		"margin": map[string]interface{}{
			"passed": checks.MarginCheck.Passed,
			"reason": checks.MarginCheck.Reason,
			"value":  checks.MarginCheck.Value,
		},
		"daily_loss": map[string]interface{}{
			"passed": checks.DailyLossCheck.Passed,
			"reason": checks.DailyLossCheck.Reason,
			"value":  checks.DailyLossCheck.Value,
		},
		"exposure": map[string]interface{}{
			"passed": checks.ExposureCheck.Passed,
			"reason": checks.ExposureCheck.Reason,
			"value":  checks.ExposureCheck.Value,
		},
	}
}
