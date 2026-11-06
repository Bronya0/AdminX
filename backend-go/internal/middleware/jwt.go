// Package middleware — JWT 鉴权中间件。
//
// 从 Authorization: Bearer <token> 解析 token，验证签名+过期，
// 将 user_id/username/is_superuser 写入 gin.Context。
// 同时更新用户 last_activity（60s 防抖。
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
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
		claims, _, err := parseBearerToken(c, jwtMgr)
		if err != nil {
			response.Fail(c, 401, err.Error())
			c.Abort()
			return
		}
		user, code, msg := loadAuthUser(c, db, claims)
		if code != 0 {
			response.Fail(c, code, msg)
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
		// 解析/校验失败一律按匿名处理：既不写响应也不写 context，
		// 否则中间件写的 401 会和 handler 的成功响应拼成两段 JSON。
		if claims, _, err := parseBearerToken(c, jwtMgr); err == nil {
			loadAuthUser(c, db, claims)
		}
		c.Next()
	}
}

// parseBearerToken 解析并校验 Authorization: Bearer <token>。
// 只解析、只报错，不写响应：由调用方决定是应答 401 还是按匿名放行。
func parseBearerToken(c *gin.Context, jwtMgr *jwt.Manager) (*jwt.Claims, string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, "", errors.New("未提供认证令牌")
	}

	// 期望格式: Bearer <token>
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, "", errors.New("认证令牌格式错误")
	}

	tokenString := strings.TrimSpace(parts[1])
	claims, err := jwtMgr.Parse(tokenString)
	if err != nil {
		return nil, "", errors.New("认证令牌无效或已过期")
	}

	// 只允许 access token 用于 API 鉴权
	if claims.GetTokenType() != jwt.TokenTypeAccess {
		return nil, "", errors.New("令牌类型错误，请使用 access token")
	}
	return claims, tokenString, nil
}

// loadAuthUser 查询用户并写入 context（存在性 + is_active + last_logout 校验）。
// 同样只返回结果不写响应：code 为 0 表示通过，否则由调用方应答该业务码。
// 只 Select 鉴权必要列：整行查询会把大字段/关联拖进每次请求的热路径。
// username 必须在列内：它会写入 gin context 供审计日志取操作者。
func loadAuthUser(c *gin.Context, db *gorm.DB, claims *jwt.Claims) (*model.User, int, string) {
	// 查询用户（校验存在性 + is_active）
	var user model.User
	if err := db.Select("id", "username", "is_active", "is_superuser", "last_logout", "last_activity").
		First(&user, "id = ? AND deleted_at IS NULL", claims.UserID).Error; err != nil {
		return nil, 401, "用户不存在或已被删除"
	}
	if !user.IsActive {
		return nil, 403, "账号已被禁用"
	}

	// 校验 last_logout（与 Refresh/Introspect/WS 共用同一判定）
	if jwt.RevokedByLogout(claims.IssuedAt, user.LastLogout) {
		return nil, 401, "认证令牌已失效，请重新登录"
	}

	// 写入 context，供后续 handler/middleware 使用
	c.Set("user_id", user.ID)
	c.Set("username", user.Username)
	c.Set("is_superuser", user.IsSuperuser)
	return &user, 0, ""
}

// activityDebounce last_activity 写库的最小间隔。
const activityDebounce = 60 * time.Second

// activityWrote 进程内记录「上次已发起写库」的时间：并发请求读到同一份旧
// last_activity 时会各自通过库内防抖、各起一个 goroutine UPDATE 同一行，
// 这里按 user_id 去重，60s 内只放一个过去。
var (
	activityMu    sync.Mutex
	activityWrote = make(map[string]time.Time)
	// activitySwept 上次真正清扫 activityWrote 的时间（见 claimActivityWrite 的节流）。
	activitySwept time.Time
)

// activitySweepInterval 清扫 map 的最小间隔。活跃用户接近 1 万且条目都很新时，
// 过期项清不掉，逐请求全表遍历会退化成每请求一次 O(n) 持锁扫描，因此加节流。
const activitySweepInterval = 10 * time.Second

// claimActivityWrite 抢占该用户本轮的写库资格（进程内 60s 去重）。
func claimActivityWrite(userID string, now time.Time) bool {
	activityMu.Lock()
	defer activityMu.Unlock()
	if last, ok := activityWrote[userID]; ok && now.Sub(last) < activityDebounce {
		return false
	}
	// 防 map 无限增长：条目过多时清掉已过期的。清扫结果是“清不掉”时也不能每请求都重扫，
	// 所以最多每 activitySweepInterval 真正遍历一次（定期清扫仍然成立，map 不会无限增长）。
	if len(activityWrote) > 10000 && now.Sub(activitySwept) >= activitySweepInterval {
		activitySwept = now
		for id, at := range activityWrote {
			if now.Sub(at) >= activityDebounce {
				delete(activityWrote, id)
			}
		}
	}
	activityWrote[userID] = now
	return true
}

// updateActivity 更新用户最后活动时间（60s 防抖）。
func updateActivity(c *gin.Context, db *gorm.DB, userID string, lastActivity *time.Time) {
	now := time.Now()
	if lastActivity != nil && now.Sub(*lastActivity) < activityDebounce {
		return // 60s 内不重复写
	}
	// 解析 UUID 校验合法性，避免脏数据
	if _, err := uuid.Parse(userID); err != nil {
		return
	}
	// 库内值可能是旧的（上次更新还没落盘），再用进程内记录挡一次并发重复写
	if !claimActivityWrite(userID, now) {
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
