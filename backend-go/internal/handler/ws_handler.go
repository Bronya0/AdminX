package handler

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"

	"adminx/internal/jwt"
	wsport "adminx/internal/websocket"
	"adminx/pkg/response"
)

// WSHandler WebSocket 处理器。
type WSHandler struct {
	hub            *wsport.Hub
	jwtMgr         *jwt.Manager
	allowedOrigins []string // 额外允许的 Origin；空 = 仅同源
	logger         *slog.Logger
}

// newUpgrader 构造 Upgrader：Origin 白名单 = 同源 + 配置项。
// 空 Origin（非浏览器客户端）放行，由 token 校验兜底。
func newUpgrader(allowedOrigins []string) gorillaws.Upgrader {
	return gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			// 同源请求放行
			if u, err := url.Parse(origin); err == nil && u.Host == r.Host {
				return true
			}
			for _, o := range allowedOrigins {
				if o == "*" || strings.EqualFold(o, origin) {
					return true
				}
			}
			return false
		},
	}
}

func NewWSHandler(hub *wsport.Hub, jwtMgr *jwt.Manager, allowedOrigins []string, logger *slog.Logger) *WSHandler {
	return &WSHandler{hub: hub, jwtMgr: jwtMgr, allowedOrigins: allowedOrigins, logger: logger}
}

// Log GET /ws/log/ — 实时日志 WebSocket。
// 鉴权: 浏览器 WebSocket 无法自定义 Header，token 走 Sec-WebSocket-Protocol
// 携带（不进 URL/日志），兼容 query 参数 ?token=<access>。
// 校验失败返回 401，不升级连接。
func (h *WSHandler) Log(c *gin.Context) {
	token := c.Query("token")
	proto := c.GetHeader("Sec-WebSocket-Protocol")
	if token == "" && proto != "" {
		token = proto
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

	upg := newUpgrader(h.allowedOrigins)
	// 回显 subprotocol（浏览器要求服务端选择一个子协议，否则连接会被关闭）；
	// 仅当 token 是通过 subprotocol 携带时才回显。
	var respHeader http.Header
	if token == proto {
		respHeader = http.Header{"Sec-WebSocket-Protocol": []string{proto}}
	}
	conn, err := upg.Upgrade(c.Writer, c.Request, respHeader)
	if err != nil {
		h.logger.Warn("WebSocket 升级失败", "error", err)
		return
	}
	// 限制单帧/总读取大小，防止恶意客户端发送超大帧打内存
	conn.SetReadLimit(64 * 1024)
	h.hub.HandleConn(conn)
}
