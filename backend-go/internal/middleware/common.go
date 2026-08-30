// Package middleware 汇集 gin 中间件。
//
// 阶段 1: RequestID（请求追踪）、RequestLog（访问日志）、Recovery（panic 恢复）。
// 阶段 2+: JWT 鉴权、RBAC 权限、限流、活动追踪等。
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"adminx/pkg/response"
)

// RequestID 为每个请求注入唯一 request_id（写入 header + context）。
// 便于日志关联和错误追踪。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// RequestLog 记录每个请求的方法、路径、状态码、耗时。
func RequestLog() gin.HandlerFunc {
	logger := slog.Default()
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		cost := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		rid := c.GetString("request_id")

		attrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"cost_ms", cost,
			"ip", c.ClientIP(),
			"request_id", rid,
		}

		// 从 context 取当前用户名（JWT 中间件会写入）
		if username, ok := c.Get("username"); ok {
			attrs = append(attrs, "user", username)
		}

		if status >= 500 {
			logger.Error("request completed", attrs...)
		} else if status >= 400 {
			logger.Warn("request completed", attrs...)
		} else {
			logger.Info("request completed", attrs...)
		}
	}
}

// ComponentSecret 业务组件共享密钥校验。
// secret 为空时放行（本地演示兼容），否则要求请求头 X-Component-Token 恒时匹配。
// 用途: 保护 /cluster/components/register|heartbeat|unregister 这类无用户上下文的公开端点。
func ComponentSecret(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.Next()
			return
		}
		token := c.GetHeader("X-Component-Token")
		if token == "" || !constantTimeEqual(token, secret) {
			response.Fail(c, 401, "组件密钥缺失或不正确")
			c.Abort()
			return
		}
		c.Next()
	}
}

// constantTimeEqual 恒时字符串比较，防时序侧信道。
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
