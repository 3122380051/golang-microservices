package http

import (
	"net/http"
	"strconv"
	"strings"

	marketapp "github.com/3122380051/golang-microservices/internal/application/market"
)

// MarketHandler exposes market data endpoints.
type MarketHandler struct {
	service *marketapp.Service
}

func NewMarketHandler(service *marketapp.Service) *MarketHandler {
	return &MarketHandler{service: service}
}

func (h *MarketHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/market/price", h.getPrice)
	mux.HandleFunc("/market/candles", h.getCandles)
	mux.HandleFunc("/market/order-book", h.getOrderBook)
}

func (h *MarketHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "market-data-service"})
}

func (h *MarketHandler) getPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	item, err := h.service.GetPrice(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *MarketHandler) getCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	interval := strings.TrimSpace(r.URL.Query().Get("interval"))
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 100)

	items, err := h.service.GetCandles(r.Context(), symbol, interval, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *MarketHandler) getOrderBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 20)

	item, err := h.service.GetOrderBook(r.Context(), symbol, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func parseIntOrDefault(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
