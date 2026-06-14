package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"

	wsport "djangoadminx/internal/websocket"
)

// WSHandler WebSocket 处理器。
type WSHandler struct {
	hub    *wsport.Hub
	logger *slog.Logger
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWSHandler(hub *wsport.Hub, logger *slog.Logger) *WSHandler {
	return &WSHandler{hub: hub, logger: logger}
}

// Log GET /ws/log/ — 实时日志 WebSocket。
func (h *WSHandler) Log(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("WebSocket 升级失败", "error", err)
		return
	}
	h.hub.HandleConn(conn)
}
