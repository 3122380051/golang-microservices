package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// PortfolioService defines methods needed by HTTP handlers
type PortfolioService interface {
	GetPortfolio(ctx context.Context, userID string) (*domain.Portfolio, error)
	ListPortfolios(ctx context.Context) ([]domain.Portfolio, error)
	GetTradeHistory(ctx context.Context, userID string) ([]domain.TradeResult, error)
	UpdatePortfolioPrices(ctx context.Context, userID string, prices map[string]float64) error
}

// PortfolioHandler wraps portfolio service HTTP endpoints
type PortfolioHandler struct {
	service PortfolioService
	logger  *slog.Logger
}

// NewPortfolioHandler creates a new portfolio handler
func NewPortfolioHandler(service PortfolioService, logger *slog.Logger) *PortfolioHandler {
	return &PortfolioHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers portfolio endpoints
func (h *PortfolioHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /portfolios", h.listPortfolios)
	mux.HandleFunc("GET /portfolios/{user_id}", h.getPortfolio)
	mux.HandleFunc("GET /portfolios/{user_id}/trades", h.getTradeHistory)
	mux.HandleFunc("PUT /portfolios/{user_id}/prices", h.updatePrices)
}

type portfolioResponse struct {
	ID                string                      `json:"id"`
	UserID            string                      `json:"user_id"`
	Status            string                      `json:"status"`
	TotalBalance      float64                     `json:"total_balance"`
	AvailableMargin   float64                     `json:"available_margin"`
	UsedMargin        float64                     `json:"used_margin"`
	MaintenanceMargin float64                     `json:"maintenance_margin"`
	Positions         map[string]positionResponse `json:"positions"`
	RealizedPnL       float64                     `json:"realized_pnl"`
	UnrealizedPnL     float64                     `json:"unrealized_pnl"`
	TotalPnL          float64                     `json:"total_pnl"`
	MarginRatio       float64                     `json:"margin_ratio"`
	TotalTrades       int                         `json:"total_trades"`
	WinRate           float64                     `json:"win_rate"`
	CreatedAt         string                      `json:"created_at"`
	UpdatedAt         string                      `json:"updated_at"`
}

type positionResponse struct {
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"`
	Quantity          float64 `json:"quantity"`
	AverageEntryPrice float64 `json:"average_entry_price"`
	CurrentPrice      float64 `json:"current_price"`
	UnrealizedPnL     float64 `json:"unrealized_pnl"`
	RealizedPnL       float64 `json:"realized_pnl"`
	PositionValue     float64 `json:"position_value"`
	UpdatedAt         string  `json:"updated_at"`
}

type tradeResultResponse struct {
	ID             string  `json:"id"`
	ExecutionID    string  `json:"execution_id"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	EntryPrice     float64 `json:"entry_price"`
	ExitPrice      float64 `json:"exit_price"`
	Quantity       float64 `json:"quantity"`
	RealizedPnL    float64 `json:"realized_pnl"`
	RealizedReturn float64 `json:"realized_return"`
	Fees           float64 `json:"fees"`
	NetPnL         float64 `json:"net_pnl"`
	EntryTime      string  `json:"entry_time"`
	ExitTime       string  `json:"exit_time"`
	DurationSecs   int64   `json:"duration_seconds"`
	IsWin          bool    `json:"is_win"`
}

func (h *PortfolioHandler) getPortfolio(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	portfolio, err := h.service.GetPortfolio(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get portfolio", "error", err, "user_id", userID)
		writeError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	if portfolio == nil {
		writeError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	resp := h.toPortfolioResponse(portfolio)
	writeJSON(w, http.StatusOK, resp)
}

func (h *PortfolioHandler) listPortfolios(w http.ResponseWriter, r *http.Request) {
	portfolios, err := h.service.ListPortfolios(r.Context())
	if err != nil {
		h.logger.Error("failed to list portfolios", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list portfolios")
		return
	}

	responses := make([]portfolioResponse, len(portfolios))
	for i, p := range portfolios {
		responses[i] = h.toPortfolioResponse(&p)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"portfolios": responses,
		"count":      len(responses),
	})
}

func (h *PortfolioHandler) getTradeHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	trades, err := h.service.GetTradeHistory(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get trade history", "error", err, "user_id", userID)
		writeError(w, http.StatusInternalServerError, "failed to get trade history")
		return
	}

	responses := make([]tradeResultResponse, len(trades))
	for i, trade := range trades {
		responses[i] = h.toTradeResponse(&trade)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"trades": responses,
		"count":  len(responses),
	})
}

func (h *PortfolioHandler) updatePrices(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	var req map[string]float64
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.UpdatePortfolioPrices(r.Context(), userID, req); err != nil {
		h.logger.Error("failed to update prices", "error", err, "user_id", userID)
		writeError(w, http.StatusInternalServerError, "failed to update prices")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "prices updated"})
}

func (h *PortfolioHandler) toPortfolioResponse(p *domain.Portfolio) portfolioResponse {
	positions := make(map[string]positionResponse)
	for symbol, pos := range p.Positions {
		positions[symbol] = positionResponse{
			Symbol:            pos.Symbol,
			Side:              string(pos.Side),
			Quantity:          pos.Quantity,
			AverageEntryPrice: pos.AverageEntryPrice,
			CurrentPrice:      pos.CurrentPrice,
			UnrealizedPnL:     pos.UnrealizedPnL,
			RealizedPnL:       pos.RealizedPnL,
			PositionValue:     pos.PositionValue,
			UpdatedAt:         pos.UpdatedAt.String(),
		}
	}

	return portfolioResponse{
		ID:                p.ID,
		UserID:            p.UserID,
		Status:            string(p.Status),
		TotalBalance:      p.TotalBalance,
		AvailableMargin:   p.AvailableMargin,
		UsedMargin:        p.UsedMargin,
		MaintenanceMargin: p.MaintenanceMargin,
		Positions:         positions,
		RealizedPnL:       p.RealizedPnL,
		UnrealizedPnL:     p.UnrealizedPnL,
		TotalPnL:          p.TotalPnL,
		MarginRatio:       p.MarginRatio,
		TotalTrades:       p.TotalTrades,
		WinRate:           p.WinRate,
		CreatedAt:         p.CreatedAt.String(),
		UpdatedAt:         p.UpdatedAt.String(),
	}
}

func (h *PortfolioHandler) toTradeResponse(t *domain.TradeResult) tradeResultResponse {
	return tradeResultResponse{
		ID:             t.ID,
		ExecutionID:    t.ExecutionID,
		Symbol:         t.Symbol,
		Side:           string(t.Side),
		EntryPrice:     t.EntryPrice,
		ExitPrice:      t.ExitPrice,
		Quantity:       t.Quantity,
		RealizedPnL:    t.RealizedPnL,
		RealizedReturn: t.RealizedReturn,
		Fees:           t.Fees,
		NetPnL:         t.NetPnL,
		EntryTime:      t.EntryTime.String(),
		ExitTime:       t.ExitTime.String(),
		DurationSecs:   t.DurationSeconds,
		IsWin:          t.IsWin,
	}
}
