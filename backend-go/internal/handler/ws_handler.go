package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"

	"adminx/internal/jwt"
	wsport "adminx/internal/websocket"
	"adminx/pkg/response"
)

// WSHandler WebSocket 处理器。
type WSHandler struct {
	hub    *wsport.Hub
	jwtMgr *jwt.Manager
	logger *slog.Logger
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 校验 Origin 防 CSWSH（跨站 WebSocket 劫持）；无 Origin 头（非浏览器客户端）放行
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// 同源请求放行（浏览器自带 Origin）；非 http/https 协议拒绝
		return strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://")
	},
}

func NewWSHandler(hub *wsport.Hub, jwtMgr *jwt.Manager, logger *slog.Logger) *WSHandler {
	return &WSHandler{hub: hub, jwtMgr: jwtMgr, logger: logger}
}

// Log GET /ws/log/ — 实时日志 WebSocket。
// 鉴权: 浏览器 WebSocket 无法自定义 Header，改用 query 参数 ?token=<access>。
// 校验失败返回 401，不升级连接。
func (h *WSHandler) Log(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Sec-WebSocket-Protocol") // 兼容 subprotocol 携带
	}
	if token == "" {
		response.Fail(c, 401, "WebSocket 缺少认证 token")
		return
	}
	claims, err := h.jwtMgr.Parse(token)
	if err != nil {
		response.Fail(c, 401, "WebSocket token 无效或已过期")
		return
	}
	// 仅允许 access token
	if claims.GetTokenType() != jwt.TokenTypeAccess {
		response.Fail(c, 401, "令牌类型错误")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("WebSocket 升级失败", "error", err)
		return
	}
	// 限制单帧/总读取大小，防止恶意客户端发送超大帧打内存
	conn.SetReadLimit(64 * 1024)
	h.hub.HandleConn(conn)
}
