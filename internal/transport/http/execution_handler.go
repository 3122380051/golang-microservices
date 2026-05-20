package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// ExecutionService defines methods needed by HTTP handlers
type ExecutionService interface {
	GetExecution(ctx context.Context, executionID string) (*domain.Execution, error)
	ListExecutionsByUser(ctx context.Context, userID string, status *domain.ExecutionStatus) ([]domain.Execution, error)
}

// ExecutionHandler wraps execution service HTTP endpoints
type ExecutionHandler struct {
	service ExecutionService
	logger  *slog.Logger
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(service ExecutionService, logger *slog.Logger) *ExecutionHandler {
	return &ExecutionHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers execution endpoints
func (h *ExecutionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /executions", h.listExecutions)
	mux.HandleFunc("GET /executions/{id}", h.getExecution)
}

type executionResponse struct {
	ID               string  `json:"id"`
	OrderID          string  `json:"order_id"`
	ClientOrderID    string  `json:"client_order_id"`
	CorrelationID    string  `json:"correlation_id"`
	ExchangeOrderID  string  `json:"exchange_order_id"`
	UserID           string  `json:"user_id"`
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	OriginalQuantity float64 `json:"original_quantity"`
	ExecutedQuantity float64 `json:"executed_quantity"`
	ExecutedValue    float64 `json:"executed_value"`
	AverageFillPrice float64 `json:"average_fill_price"`
	Fees             float64 `json:"fees"`
	Status           string  `json:"status"`
	AttemptCount     int     `json:"attempt_count"`
	LastAttemptError string  `json:"last_attempt_error,omitempty"`
	SubmittedAt      string  `json:"submitted_at,omitempty"`
	FilledAt         string  `json:"first_filled_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func (h *ExecutionHandler) getExecution(w http.ResponseWriter, r *http.Request) {
	executionID := r.PathValue("id")
	if executionID == "" {
		writeError(w, http.StatusBadRequest, "execution_id is required")
		return
	}

	exec, err := h.service.GetExecution(r.Context(), executionID)
	if err != nil {
		h.logger.Error("failed to get execution", "error", err, "execution_id", executionID)
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}

	if exec == nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}

	resp := h.toResponse(exec)
	writeJSON(w, http.StatusOK, resp)
}

func (h *ExecutionHandler) listExecutions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	statusStr := r.URL.Query().Get("status")
	var status *domain.ExecutionStatus
	if statusStr != "" {
		s := domain.ExecutionStatus(statusStr)
		status = &s
	}

	executions, err := h.service.ListExecutionsByUser(r.Context(), userID, status)
	if err != nil {
		h.logger.Error("failed to list executions", "error", err, "user_id", userID)
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	responses := make([]executionResponse, len(executions))
	for i, exec := range executions {
		responses[i] = h.toResponse(&exec)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"executions": responses,
		"count":      len(responses),
	})
}

func (h *ExecutionHandler) toResponse(exec *domain.Execution) executionResponse {
	resp := executionResponse{
		ID:               exec.ID,
		OrderID:          exec.OrderID,
		ClientOrderID:    exec.ClientOrderID,
		CorrelationID:    exec.CorrelationID,
		ExchangeOrderID:  exec.ExchangeOrderID,
		UserID:           exec.UserID,
		Symbol:           exec.Symbol,
		Side:             string(exec.Side),
		OriginalQuantity: exec.OriginalQuantity,
		ExecutedQuantity: exec.ExecutedQuantity,
		ExecutedValue:    exec.ExecutedValue,
		AverageFillPrice: exec.AverageFillPrice,
		Fees:             exec.Fees,
		Status:           string(exec.Status),
		AttemptCount:     exec.AttemptCount,
		LastAttemptError: exec.LastAttemptError,
		CreatedAt:        exec.CreatedAt.String(),
		UpdatedAt:        exec.UpdatedAt.String(),
	}

	if exec.SubmittedAt != nil {
		resp.SubmittedAt = exec.SubmittedAt.String()
	}
	if exec.FirstFilledAt != nil {
		resp.FilledAt = exec.FirstFilledAt.String()
	}

	return resp
}
