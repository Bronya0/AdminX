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

// RequestLog 记录每个请求的方法、路径、状态码、耗时（对齐 AdminX RequestLogMiddleware）。
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
