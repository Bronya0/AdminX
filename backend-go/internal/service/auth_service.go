// Package service 业务逻辑层。
//
// auth_service 实现登录/登出/刷新/introspect 完整流程，
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	apperr "adminx/pkg/errors"
	"adminx/pkg/crypto"

	"adminx/internal/jwt"
	"adminx/internal/model"
	"adminx/internal/repository"
)

// AuthService 认证业务。
type AuthService struct {
	db            *gorm.DB
	userRepo      *repository.UserRepo
	lockRepo      *repository.LoginLockRepo
	logRepo       *repository.LoginLogRepo
	jwtMgr        *jwt.Manager
	logger        *slog.Logger

	// 可配置项
	loginLockEnabled bool
	maxAttempts      int
	lockDuration     time.Duration
}

// NewAuthService 构造 AuthService。
func NewAuthService(
	db *gorm.DB,
	userRepo *repository.UserRepo,
	lockRepo *repository.LoginLockRepo,
	logRepo *repository.LoginLogRepo,
	jwtMgr *jwt.Manager,
	logger *slog.Logger,
	loginLockEnabled bool,
	maxAttempts int,
	lockDuration time.Duration,
) *AuthService {
	return &AuthService{
		db: db, userRepo: userRepo, lockRepo: lockRepo, logRepo: logRepo,
		jwtMgr: jwtMgr, logger: logger,
		loginLockEnabled: loginLockEnabled,
		maxAttempts:      maxAttempts,
		lockDuration:     lockDuration,
	}
}

// LoginResult 登录成功返回结构。
type LoginResult struct {
	Access  string      `json:"access"`
	Refresh string      `json:"refresh"`
	User    *model.User `json:"user"`
}

// Login 用户登录。
func (s *AuthService) Login(ctx context.Context, username, password, ip, userAgent string) (*LoginResult, error) {
	// 1. 登录锁检查
	if s.loginLockEnabled {
		locked, remaining, err := s.lockRepo.IsLocked(username, s.lockDuration)
		if err != nil {
			s.logger.ErrorContext(ctx, "检查登录锁失败", "error", err, "username", username)
				return nil, apperr.ErrInternal
		}
		if locked {
			s.recordLoginLog(nil, username, ip, userAgent, false, fmt.Sprintf("账号已锁定，%d秒后重试", remaining))
			return nil, apperr.New(423, fmt.Sprintf("账号已锁定，请 %d 秒后重试", remaining))
		}
	}

	// 2. 查询用户
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.recordLoginFailure(nil, username, ip, userAgent, "用户名或密码错误")
			return nil, apperr.New(401, "用户名或密码错误")
		}
		s.logger.ErrorContext(ctx, "查询用户失败", "error", err)
		return nil, apperr.ErrInternal
	}

	// 3. 密码验证（bcrypt）
	if err := crypto.CheckPassword(user.Password, password); err != nil {
		if s.loginLockEnabled {
			_ = s.lockRepo.IncrementFailed(username, s.maxAttempts)
		}
		uidCopy := user.ID
		s.recordLoginFailure(&uidCopy, username, ip, userAgent, "用户名或密码错误")
		return nil, apperr.New(401, "用户名或密码错误")
	}

	// 4. 激活检查
	if !user.IsActive {
		uidCopy := user.ID
		s.recordLoginFailure(&uidCopy, username, ip, userAgent, "账号已被禁用")
		return nil, apperr.New(403, "账号已被禁用")
	}

	// 5. 清除登录锁
	if s.loginLockEnabled {
		_ = s.lockRepo.Delete(username)
	}

	// 6. 签发 token
	access, err := s.jwtMgr.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		s.logger.ErrorContext(ctx, "签发 access token 失败", "error", err)
		return nil, apperr.ErrInternal
	}
	refresh, err := s.jwtMgr.GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		s.logger.ErrorContext(ctx, "签发 refresh token 失败", "error", err)
		return nil, apperr.ErrInternal
	}

	// 7. 更新最后登录时间
	_ = s.userRepo.UpdateLogin(user.ID)

	// 8. 写登录日志（成功）
	uidCopy2 := user.ID
	s.recordLoginLog(&uidCopy2, username, ip, userAgent, true, "登录成功")

	return &LoginResult{Access: access, Refresh: refresh, User: user}, nil
}

// Logout 登出。将 refresh token 加入黑名单 + 更新 last_logout。
// 校验 refresh token 的归属（防止用户拉黑他人 token 的 DoS）。
func (s *AuthService) Logout(ctx context.Context, userID, refreshToken string) error {
	// 校验 refresh token 归属当前用户（防 DoS: 用户拿别人的 refresh 来登出）
	if refreshToken != "" {
		if claims, err := s.jwtMgr.Parse(refreshToken); err == nil {
			if claims.UserID != userID {
				// refresh token 不属于当前用户，拒绝拉黑（静默成功，不暴露差异）
				s.logger.WarnContext(ctx, "logout 拒绝: refresh token 归属不匹配",
					"current_user", userID, "token_user", claims.UserID)
				return nil
			}
		}
		// token 解析失败（无效/过期）也允许 logout 成功，只是不拉黑
		if err := s.jwtMgr.BlacklistRefreshToken(ctx, refreshToken); err != nil {
			s.logger.WarnContext(ctx, "拉黑 refresh token 失败", "error", err)
		}
	}
	// 更新 last_logout（使该时间前签发的 access token 失效）
	if userID != "" {
		if err := s.userRepo.UpdateLogout(userID); err != nil {
			s.logger.WarnContext(ctx, "更新 last_logout 失败", "error", err)
		}
	}
	return nil
}

// RefreshResult 刷新返回结构。
type RefreshResult struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

// Refresh 用 refresh token 换发新的 access + refresh（带黑名单轮换。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	// 校验 token 类型
	claims, err := s.jwtMgr.Parse(refreshToken)
	if err != nil {
		return nil, apperr.New(401, "refresh token 无效或已过期")
	}
	if claims.GetTokenType() != jwt.TokenTypeRefresh {
		return nil, apperr.New(401, "令牌类型错误，需要 refresh token")
	}

	// 黑名单检查
	blacklisted, err := s.jwtMgr.IsBlacklisted(ctx, refreshToken)
	if err != nil {
		s.logger.WarnContext(ctx, "黑名单检查失败", "error", err)
	}
	if blacklisted {
		return nil, apperr.New(401, "refresh token 已被吊销")
	}

	// 校验用户仍存在且活跃
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, apperr.New(401, "用户不存在")
	}
	if !user.IsActive {
		return nil, apperr.New(403, "账号已被禁用")
	}

	// 轮换: 旧 refresh 拉黑 + 签发新 token
	_ = s.jwtMgr.BlacklistRefreshToken(ctx, refreshToken)

	access, err := s.jwtMgr.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		return nil, apperr.ErrInternal
	}
	newRefresh, err := s.jwtMgr.GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		return nil, apperr.ErrInternal
	}

	return &RefreshResult{Access: access, Refresh: newRefresh}, nil
}

// IntrospectResult token 校验返回结构（供业务容器调用）。
type IntrospectResult struct {
	Valid       bool     `json:"valid"`
	UserID      string   `json:"user_id,omitempty"`
	Username    string   `json:"username,omitempty"`
	Email       string   `json:"email,omitempty"`
	Phone       string   `json:"phone,omitempty"`
	Avatar      string   `json:"avatar,omitempty"`
	IsSuperuser bool     `json:"is_superuser,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Exp         int64    `json:"exp,omitempty"`
	Iat         int64    `json:"iat,omitempty"`
	TokenType   string   `json:"token_type,omitempty"`
	LoginURL    string   `json:"login_url,omitempty"`
}

// Introspect 校验 access token 有效性并返回用户信息（供业务容器调用）。
func (s *AuthService) Introspect(ctx context.Context, tokenString string) (*IntrospectResult, error) {
	result := &IntrospectResult{LoginURL: "/login"}

	claims, err := s.jwtMgr.Parse(tokenString)
	if err != nil {
		result.Valid = false
		return result, nil // 不报错，返回 valid=false
	}
	if claims.GetTokenType() != jwt.TokenTypeAccess {
		result.Valid = false
		return result, nil
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		result.Valid = false
		return result, nil
	}
	if !user.IsActive {
		result.Valid = false
		return result, nil
	}

	// last_logout 二次失效检查
	if user.LastLogout != nil && claims.IssuedAt != nil {
		if claims.IssuedAt.Time.Before(*user.LastLogout) {
			result.Valid = false
			return result, nil
		}
	}

	// 组装角色名
	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}

	result.Valid = true
	result.UserID = user.ID
	result.Username = user.Username
	result.Email = user.Email
	result.Phone = user.Phone
	result.Avatar = user.Avatar
	result.IsSuperuser = user.IsSuperuser
	result.Roles = roleNames
	if claims.ExpiresAt != nil {
		result.Exp = claims.ExpiresAt.Time.Unix()
	}
	if claims.IssuedAt != nil {
		result.Iat = claims.IssuedAt.Time.Unix()
	}
	result.TokenType = jwt.TokenTypeAccess
	return result, nil
}

// recordLoginLog 写登录日志。
func (s *AuthService) recordLoginLog(userID *string, username, ip, userAgent string, success bool, msg string) {
	log := &model.UserLoginLog{
		UserID:    userID,
		Username:  username,
		IP:        ip,
		UserAgent: userAgent,
		Success:   success,
		Message:   msg,
	}
	if err := s.logRepo.Create(log); err != nil {
		s.logger.Warn("写登录日志失败", "error", err)
	}
}

// recordLoginFailure 登录失败日志的便捷方法（含锁定递增）。
func (s *AuthService) recordLoginFailure(userID *string, username, ip, userAgent, msg string) {
	s.recordLoginLog(userID, username, ip, userAgent, false, msg)
}

// ListLoginLogs 分页查询登录日志（供 handler 调用）。
func (s *AuthService) ListLoginLogs(offset, limit int, username, ip string, success *bool) ([]model.UserLoginLog, int64, error) {
	return s.logRepo.List(offset, limit, username, ip, success)
}
