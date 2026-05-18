package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/3122380051/golang-microservices/internal/application/order"
	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure"
)

// OrderHandler handles HTTP requests for order service endpoints
type OrderHandler struct {
	orderService *order.Service
	logger       infrastructure.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService *order.Service, logger infrastructure.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// RegisterRoutes registers all order service routes
func (h *OrderHandler) RegisterRoutes(server *http.ServeMux) {
	server.HandleFunc("POST /orders", h.createOrder)
	server.HandleFunc("GET /orders/{order_id}", h.getOrder)
	server.HandleFunc("GET /orders", h.listOrders)
	server.HandleFunc("DELETE /orders/{order_id}", h.cancelOrder)
	server.HandleFunc("PUT /orders/{order_id}/fill", h.updateOrderFill)
}

// createOrder POST /orders
func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID         string  `json:"user_id"`
		StrategyID     string  `json:"strategy_id"`
		Symbol         string  `json:"symbol"`
		Side           string  `json:"side"` // BUY or SELL
		Quantity       float64 `json:"quantity"`
		SignalID       string  `json:"signal_id"`
		RiskDecisionID string  `json:"risk_decision_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse side
	var side domain.OrderSide
	if req.Side == "BUY" {
		side = domain.OrderSideBuy
	} else if req.Side == "SELL" {
		side = domain.OrderSideSell
	} else {
		h.respondError(w, http.StatusBadRequest, "side must be BUY or SELL")
		return
	}

	// Create order
	ord, err := h.orderService.CreateOrder(
		r.Context(),
		req.UserID,
		req.StrategyID,
		req.Symbol,
		side,
		req.Quantity,
		req.SignalID,
		req.RiskDecisionID,
	)
	if err != nil {
		status := http.StatusInternalServerError
		if order.IsValidationError(err) {
			status = http.StatusBadRequest
		}
		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":               ord.ID,
		"client_order_id":  ord.ClientOrderID,
		"correlation_id":   ord.CorrelationID,
		"status":           ord.Status,
		"symbol":           ord.Symbol,
		"side":             ord.Side,
		"quantity":         ord.Quantity,
		"created_at":       ord.CreatedAt,
	})
}

// getOrder GET /orders/{order_id}
func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		h.respondError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	ord, err := h.orderService.GetOrder(r.Context(), orderID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, ord)
}

// listOrders GET /orders?user_id=X&status=Y&strategy_id=Z
func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	strategyID := r.URL.Query().Get("strategy_id")
	statusStr := r.URL.Query().Get("status")

	var status *domain.OrderStatus
	if statusStr != "" {
		s := domain.OrderStatus(statusStr)
		status = &s
	}

	var orders []domain.Order
	var err error

	if userID != "" {
		orders, err = h.orderService.ListOrdersByUser(r.Context(), userID, status)
	} else if strategyID != "" {
		orders, err = h.orderService.ListOrdersByStrategy(r.Context(), strategyID, status)
	} else {
		h.respondError(w, http.StatusBadRequest, "user_id or strategy_id is required")
		return
	}

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"count":  len(orders),
	})
}

// cancelOrder DELETE /orders/{order_id}
func (h *OrderHandler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		h.respondError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ord, err := h.orderService.CancelOrder(r.Context(), orderID, req.Reason)
	if err != nil {
		status := http.StatusInternalServerError
		if order.IsValidationError(err) {
			status = http.StatusBadRequest
		}
		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":     ord.ID,
		"status": ord.Status,
		"reason": ord.CancelReason,
	})
}

// updateOrderFill PUT /orders/{order_id}/fill
func (h *OrderHandler) updateOrderFill(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		h.respondError(w, http.StatusBadRequest, "order_id is required")
		return
	}

	var req struct {
		ExecutedQuantity float64 `json:"executed_quantity"`
		AverageFillPrice float64 `json:"average_fill_price"`
		Fees             float64 `json:"fees"`
		Status           string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	status := domain.OrderStatus(req.Status)
	ord, err := h.orderService.UpdateOrderFill(
		r.Context(),
		orderID,
		req.ExecutedQuantity,
		req.AverageFillPrice,
		req.Fees,
		status,
	)
	if err != nil {
		status := http.StatusInternalServerError
		if order.IsValidationError(err) {
			status = http.StatusBadRequest
		}
		h.respondError(w, status, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, ord)
}

// Helper functions

func (h *OrderHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *OrderHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  status,
	})
}
