package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// OrderService defines the methods needed by HTTP handlers.
type OrderService interface {
	CreateOrder(ctx context.Context, userID, strategyID, symbol string, side domain.OrderSide, quantity float64, signalID, riskDecisionID string) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID string) (*domain.Order, error)
	ListOrdersByUser(ctx context.Context, userID string, status *domain.OrderStatus) ([]domain.Order, error)
	ListOrdersByStrategy(ctx context.Context, strategyID string, status *domain.OrderStatus) ([]domain.Order, error)
	CancelOrder(ctx context.Context, orderID, reason string) (*domain.Order, error)
	UpdateOrderFill(ctx context.Context, orderID string, executedQty, avgPrice, fees float64, newStatus domain.OrderStatus) (*domain.Order, error)
}

// OrderHandler wraps order service HTTP endpoints
type OrderHandler struct {
	service OrderService
	logger  *slog.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(service OrderService, logger *slog.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers order endpoints
func (h *OrderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders", h.listOrders)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("PUT /orders/{id}/fill", h.updateOrderFill)
	mux.HandleFunc("DELETE /orders/{id}", h.cancelOrder)
}

type createOrderRequest struct {
	UserID         string  `json:"user_id"`
	StrategyID     string  `json:"strategy_id"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"` // BUY or SELL
	Quantity       float64 `json:"quantity"`
	SignalID       string  `json:"signal_id"`
	RiskDecisionID string  `json:"risk_decision_id"`
}

type orderResponse struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	StrategyID     string  `json:"strategy_id"`
	ClientOrderID  string  `json:"client_order_id"`
	CorrelationID  string  `json:"correlation_id"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Type           string  `json:"type"`
	Quantity       float64 `json:"quantity"`
	FilledQuantity float64 `json:"filled_quantity"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type updateOrderFillRequest struct {
	FillQuantity float64 `json:"fill_quantity"`
	FillPrice    float64 `json:"fill_price"`
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Convert side string to OrderSide
	var side domain.OrderSide
	switch req.Side {
	case "BUY":
		side = domain.OrderSideBuy
	case "SELL":
		side = domain.OrderSideSell
	default:
		writeError(w, http.StatusBadRequest, "invalid side: must be BUY or SELL")
		return
	}

	order, err := h.service.CreateOrder(r.Context(), req.UserID, req.StrategyID, req.Symbol, side, req.Quantity, req.SignalID, req.RiskDecisionID)
	if err != nil {
		h.logger.Error("failed to create order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	resp := orderResponse{
		ID:             order.ID,
		UserID:         order.UserID,
		StrategyID:     order.StrategyID,
		ClientOrderID:  order.ClientOrderID,
		CorrelationID:  order.CorrelationID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		Type:           string(order.OrderType),
		Quantity:       order.Quantity,
		FilledQuantity: order.ExecutedQuantity,
		Status:         string(order.Status),
		CreatedAt:      order.CreatedAt.String(),
		UpdatedAt:      order.UpdatedAt.String(),
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		h.logger.Error("failed to get order", "error", err, "order_id", orderID)
		writeError(w, http.StatusInternalServerError, "failed to get order")
		return
	}

	if order == nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	resp := orderResponse{
		ID:             order.ID,
		UserID:         order.UserID,
		StrategyID:     order.StrategyID,
		ClientOrderID:  order.ClientOrderID,
		CorrelationID:  order.CorrelationID,
		Symbol:         order.Symbol,
		Side:           string(order.Side),
		Type:           string(order.OrderType),
		Quantity:       order.Quantity,
		FilledQuantity: order.ExecutedQuantity,
		Status:         string(order.Status),
		CreatedAt:      order.CreatedAt.String(),
		UpdatedAt:      order.UpdatedAt.String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	orders, err := h.service.ListOrdersByUser(r.Context(), userID, nil)
	if err != nil {
		h.logger.Error("failed to list orders", "error", err, "user_id", userID)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	responses := make([]orderResponse, len(orders))
	for i, order := range orders {
		responses[i] = orderResponse{
			ID:             order.ID,
			UserID:         order.UserID,
			StrategyID:     order.StrategyID,
			ClientOrderID:  order.ClientOrderID,
			CorrelationID:  order.CorrelationID,
			Symbol:         order.Symbol,
			Side:           string(order.Side),
			Type:           string(order.OrderType),
			Quantity:       order.Quantity,
			FilledQuantity: order.ExecutedQuantity,
			Status:         string(order.Status),
			CreatedAt:      order.CreatedAt.String(),
			UpdatedAt:      order.UpdatedAt.String(),
		}
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h *OrderHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	ord, err := h.service.CancelOrder(r.Context(), orderID, "cancelled via api")
	if err != nil {
		h.logger.Error("failed to cancel order", "error", err, "order_id", orderID)
		writeError(w, http.StatusInternalServerError, "failed to cancel order")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(ord.Status), "order_id": ord.ID})
}

func (h *OrderHandler) updateOrderFill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	var req updateOrderFillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	ord, err := h.service.UpdateOrderFill(r.Context(), orderID, req.FillQuantity, req.FillPrice, 0, domain.OrderStatusPartialFilled)
	if err != nil {
		h.logger.Error("failed to update order fill", "error", err, "order_id", orderID)
		writeError(w, http.StatusInternalServerError, "failed to update order fill")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(ord.Status), "order_id": ord.ID})
}
