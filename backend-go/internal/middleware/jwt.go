// Package middleware — JWT 鉴权中间件。
//
// 从 Authorization: Bearer <token> 解析 token，验证签名+过期，
// 将 user_id/username/is_superuser 写入 gin.Context。
// 同时更新用户 last_activity（60s 防抖。
package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"adminx/pkg/response"

	"adminx/internal/jwt"
	"adminx/internal/model"
)

// JWTAuth 返回 JWT 鉴权中间件。
// 未携带 token 或 token 无效 → 401。
func JWTAuth(jwtMgr *jwt.Manager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, _, ok := parseBearerToken(c, jwtMgr)
		if !ok {
			c.Abort()
			return
		}
		user, ok := loadAuthUser(c, db, claims)
		if !ok {
			c.Abort()
			return
		}
		// 更新 last_activity（60s 防抖，避免每请求写库）
		updateActivity(c, db, user.ID, user.LastActivity)
		c.Next()
	}
}

// JWTSoft 可选鉴权中间件：token 有效时写入 context，无效/缺失时放行（不写 context）。
// 用于 logout 等需要在 access token 过期后仍可调用的端点（凭 refresh token 吊销会话）。
func JWTSoft(jwtMgr *jwt.Manager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			if claims, _, ok := parseBearerToken(c, jwtMgr); ok {
				loadAuthUser(c, db, claims)
			}
		}
		c.Next()
	}
}

// parseBearerToken 解析并校验 Authorization: Bearer <token>。
// 失败时已写入 401 响应，返回 ok=false。
func parseBearerToken(c *gin.Context, jwtMgr *jwt.Manager) (*jwt.Claims, string, bool) {
	fail := func(msg string) (*jwt.Claims, string, bool) {
		response.Fail(c, 401, msg)
		return nil, "", false
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return fail("未提供认证令牌")
	}

	// 期望格式: Bearer <token>
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fail("认证令牌格式错误")
	}

	tokenString := strings.TrimSpace(parts[1])
	claims, err := jwtMgr.Parse(tokenString)
	if err != nil {
		return fail("认证令牌无效或已过期")
	}

	// 只允许 access token 用于 API 鉴权
	if claims.GetTokenType() != jwt.TokenTypeAccess {
		return fail("令牌类型错误，请使用 access token")
	}
	return claims, tokenString, true
}

// loadAuthUser 查询用户并写入 context（存在性 + is_active + last_logout 校验）。
// 返回 false 时已写入 401/403 响应。
func loadAuthUser(c *gin.Context, db *gorm.DB, claims *jwt.Claims) (*model.User, bool) {
	// 查询用户（校验存在性 + is_active）
	var user model.User
	if err := db.First(&user, "id = ? AND deleted_at IS NULL", claims.UserID).Error; err != nil {
		response.Fail(c, 401, "用户不存在或已被删除")
		return nil, false
	}
	if !user.IsActive {
		response.Fail(c, 403, "账号已被禁用")
		return nil, false
	}

	// 校验 last_logout
	if user.LastLogout != nil && claims.IssuedAt != nil {
		if claims.IssuedAt.Time.Before(*user.LastLogout) {
			response.Fail(c, 401, "认证令牌已失效，请重新登录")
			return nil, false
		}
	}

	// 写入 context，供后续 handler/middleware 使用
	c.Set("user_id", user.ID)
	c.Set("username", user.Username)
	c.Set("is_superuser", user.IsSuperuser)
	return &user, true
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
	// 异步更新，不阻塞请求；使用独立 context（请求 context 在返回后即取消）
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := db.WithContext(ctx).Model(&model.User{}).
			Where("id = ?", userID).Update("last_activity", now).Error; err != nil {
			slog.Default().Warn("更新用户活动时间失败", "user_id", userID, "error", err)
		}
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
