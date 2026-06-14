// Package middleware — JWT 鉴权中间件。
//
// 从 Authorization: Bearer <token> 解析 token，验证签名+过期，
// 将 user_id/username/is_superuser 写入 gin.Context。
// 同时更新用户 last_activity（60s 防抖，对齐 Django UserActivityMiddleware）。
package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"djangoadminx/pkg/response"

	"djangoadminx/internal/jwt"
	"djangoadminx/internal/model"
)

// JWTAuth 返回 JWT 鉴权中间件。
// 未携带 token 或 token 无效 → 401。
func JWTAuth(jwtMgr *jwt.Manager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, 401, "未提供认证令牌")
			c.Abort()
			return
		}

		// 期望格式: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Fail(c, 401, "认证令牌格式错误")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := jwtMgr.Parse(tokenString)
		if err != nil {
			response.Fail(c, 401, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// 只允许 access token 用于 API 鉴权
		if claims.GetTokenType() != jwt.TokenTypeAccess {
			response.Fail(c, 401, "令牌类型错误，请使用 access token")
			c.Abort()
			return
		}

		// 查询用户（校验存在性 + is_active）
		var user model.User
		if err := db.First(&user, "id = ? AND deleted_at IS NULL", claims.UserID).Error; err != nil {
			response.Fail(c, 401, "用户不存在或已被删除")
			c.Abort()
			return
		}
		if !user.IsActive {
			response.Fail(c, 403, "账号已被禁用")
			c.Abort()
			return
		}

		// 校验 last_logout（对齐 Django: logout 后旧 access token 失效）
		if user.LastLogout != nil && claims.IssuedAt != nil {
			if claims.IssuedAt.Time.Before(*user.LastLogout) {
				response.Fail(c, 401, "认证令牌已失效，请重新登录")
				c.Abort()
				return
			}
		}

		// 写入 context，供后续 handler/middleware 使用
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("is_superuser", user.IsSuperuser)

		// 更新 last_activity（60s 防抖，避免每请求写库）
		updateActivity(c, db, user.ID, user.LastActivity)

		c.Next()
	}
}

// updateActivity 更新用户最后活动时间（60s 防抖）。
func updateActivity(c *gin.Context, db *gorm.DB, userID string, lastActivity *time.Time) {
	now := time.Now()
	if lastActivity != nil && now.Sub(*lastActivity) < 60*time.Second {
		return // 60s 内不重复写
	}
	// 解析 UUID 校验合法性，避免脏数据
	if _, err := uuid.Parse(userID); err != nil {
		return
	}
	go func() {
		// 异步更新，不阻塞请求
		db.Model(&model.User{}).Where("id = ?", userID).Update("last_activity", now)
	}()
}

// RequireSuperuser 要求超级管理员（可选使用，部分端点需要）。
func RequireSuperuser() gin.HandlerFunc {
	return func(c *gin.Context) {
		isSuper, exists := c.Get("is_superuser")
		if !exists {
			response.Fail(c, 401, "未认证")
			c.Abort()
			return
		}
		isSuperBool, ok := isSuper.(bool)
		if !ok || !isSuperBool {
			response.Fail(c, 403, "仅超级管理员可访问")
			c.Abort()
			return
		}
		c.Next()
	}
}
