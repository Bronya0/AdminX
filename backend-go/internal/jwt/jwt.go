// Package jwt 封装 JWT token 的生成、解析与黑名单。
//
// 使用 golang-jwt/v5，HS256 对称签名。
// Access token (30min) 用于 API 鉴权；Refresh token (7d) 用于换发。
// 黑名单: logout 时将 refresh token 的 jti 写入 Redis set，
// refresh 时校验是否在黑名单内（对齐 Django 的 BLACKLIST_AFTER_ROTATION）。
package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

// TokenType token 类型。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// blacklistKey Redis 中存放 refresh token 黑名单的 set key。
const blacklistKey = "jwt:blacklist"

// Claims JWT 自定义 claims（对齐 Django SimpleJWT 默认 claims）。
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"` // access / refresh（独立字段，不覆盖 Subject）
	jwt.RegisteredClaims
}

// Manager JWT 管理器。
type Manager struct {
	secret        []byte
	issuer        string
	accessExpire  time.Duration
	refreshExpire time.Duration
	rdb           *redis.Client // 可为 nil（无 Redis 时不校验黑名单）
}

// New 构造 JWT Manager。
func New(secret, issuer string, accessExpire, refreshExpire time.Duration, rdb *redis.Client) *Manager {
	return &Manager{
		secret:        []byte(secret),
		issuer:        issuer,
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
		rdb:           rdb,
	}
}

// GenerateAccessToken 生成 access token。
func (m *Manager) GenerateAccessToken(userID, username string) (string, error) {
	return m.generateToken(userID, username, TokenTypeAccess, m.accessExpire)
}

// GenerateRefreshToken 生成 refresh token。
func (m *Manager) GenerateRefreshToken(userID, username string) (string, error) {
	return m.generateToken(userID, username, TokenTypeRefresh, m.refreshExpire)
}

func (m *Manager) generateToken(userID, username, tokenType string, expire time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    m.issuer,
			Subject:   userID, // 标准 sub 字段保持 user_id
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 解析并验证 token，返回 Claims。
// 验证签名 + 过期时间；不在此处校验黑名单（由调用方按需调用 IsBlacklisted）。
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// 确保使用预期的签名算法（防止 alg:none 攻击）
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token 无效")
	}
	return claims, nil
}

// BlacklistRefreshToken 将 refresh token 加入黑名单（logout 时调用）。
func (m *Manager) BlacklistRefreshToken(ctx context.Context, tokenString string) error {
	if m.rdb == nil {
		return nil // 无 Redis，跳过黑名单
	}
	claims, err := m.Parse(tokenString)
	if err != nil {
		// 无效 token 也加入黑名单，防止重放（用 token 字符串本身做 key）
		return m.rdb.SAdd(ctx, blacklistKey, tokenString).Err()
	}
	// 用 jti 加入黑名单，TTL 设为 token 剩余有效期（过期后自动清理）
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining <= 0 {
		return nil // 已过期，无需拉黑
	}
	return m.rdb.SAdd(ctx, blacklistKey, claims.ID).Err()
}

// IsBlacklisted 检查 token 是否在黑名单内。
func (m *Manager) IsBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if m.rdb == nil {
		return false, nil
	}
	// 先按 jti 查
	claims, err := m.Parse(tokenString)
	if err == nil {
		exists, err := m.rdb.SIsMember(ctx, blacklistKey, claims.ID).Result()
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	// 再按完整 token 字符串查（兼容无效 token 的拉黑）
	return m.rdb.SIsMember(ctx, blacklistKey, tokenString).Result()
}

// TokenType 返回 token 类型（独立字段，不再复用 Subject）。
func (c *Claims) GetTokenType() string { return c.TokenType }
