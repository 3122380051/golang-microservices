package ws

import (
	"net/http"
	"time"

	marketapp "github.com/3122380051/golang-microservices/internal/application/market"
	"github.com/gorilla/websocket"
)

// MarketStreamHandler pushes market ticks through websocket.
type MarketStreamHandler struct {
	service  *marketapp.Service
	upgrader websocket.Upgrader
}

func NewMarketStreamHandler(service *marketapp.Service) *MarketStreamHandler {
	return &MarketStreamHandler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *MarketStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticks, unsubscribe := h.service.Subscribe()
	defer unsubscribe()

	for {
		select {
		case item, ok := <-ticks:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(item); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
