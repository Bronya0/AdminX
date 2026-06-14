// Package middleware — 基于 Redis 的限流中间件（固定窗口计数器）。
//
// 对齐 Django DRF Throttling 三级限流:
//   - anon: 匿名用户（按 IP）
//   - user: 认证用户（按 user_id）
//   - introspect: token 校验端点专用
//
// 用 Redis INCR + EXPIRE 实现固定窗口计数。
// 无 Redis 时降级为放行（不阻断主流程）。
package middleware

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"djangoadminx/pkg/response"
)

// Limit 限流配置。
type Limit struct {
	Requests int           // 窗口内允许的请求数
	Window   time.Duration // 窗口大小
}

// RateLimit 按场景限流。
//
// 用法:
//
//	r.Use(middleware.RateLimit(rdb, "anon", Limit{Requests: 30, Window: time.Minute}))
func RateLimit(rdb *redis.Client, scope string, limit Limit) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next() // 无 Redis，放行
			return
		}

		// 构造限流 key
		var identifier string
		if scope == "user" {
			if uid, ok := c.Get("user_id"); ok && uid != nil {
				identifier = "user:" + uid.(string)
			} else {
				identifier = "ip:" + c.ClientIP()
			}
		} else {
			identifier = "ip:" + c.ClientIP()
		}

		key := "ratelimit:" + scope + ":" + identifier
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		// INCR 计数
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis 出错不阻断请求
			c.Next()
			return
		}
		// 首次访问设置窗口过期
		if count == 1 {
			_ = rdb.Expire(ctx, key, limit.Window).Err()
		}

		// 超过阈值
		if count > int64(limit.Requests) {
			// 计算剩余等待时间
			ttl, _ := rdb.TTL(ctx, key).Result()
			retryAfter := int(ttl.Seconds())
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			response.Fail(c, 429, "请求过于频繁，请"+strconv.Itoa(retryAfter)+"秒后重试")
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limit.Requests-int(count)))
		c.Next()
	}
}

// ParseRate 从字符串解析限流配置，如 "30/minute" → Limit{30, time.Minute}。
// 支持 second/minute/hour/day。
func ParseRate(rate string) Limit {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return Limit{Requests: 100, Window: time.Minute}
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil || n <= 0 {
		return Limit{Requests: 100, Window: time.Minute}
	}
	var window time.Duration
	switch strings.ToLower(parts[1]) {
	case "second", "seconds", "s":
		window = time.Second
	case "minute", "minutes", "m":
		window = time.Minute
	case "hour", "hours", "h":
		window = time.Hour
	case "day", "days", "d":
		window = 24 * time.Hour
	default:
		window = time.Minute
	}
	return Limit{Requests: n, Window: window}
}
