// Package jwt 封装 JWT token 的生成、解析与黑名单。
//
// 使用 golang-jwt/v5，HS256 对称签名。
// Access token (30min) 用于 API 鉴权；Refresh token (7d) 用于换发。
// 黑名单: logout 时将 refresh token 的 jti 写入 Redis set，
// refresh 时校验是否在黑名单内。
package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenType token 类型。
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// blacklistKey Redis 中存放 refresh token 黑名单的 key 前缀。
// 使用独立 key（jwt:blacklist:<jti>）而非单一 set，便于按成员设置 TTL，
// 避免 set 永久堆积导致 Redis 内存泄漏。
const blacklistKeyPrefix = "jwt:blacklist:"

// Claims JWT 自定义 claims。
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

// BlacklistRefreshToken 将 refresh token 加入黑名单（logout / refresh 轮换时调用）。
// 使用独立 key + TTL，过期后 Redis 自动清理，避免内存泄漏。
func (m *Manager) BlacklistRefreshToken(ctx context.Context, tokenString string) error {
	if m.rdb == nil {
		return nil // 无 Redis，跳过黑名单
	}
	claims, err := m.Parse(tokenString)
	if err != nil {
		// 无效 token 无法获取过期时间，直接忽略（无效 token 本就无法用于 refresh）
		return nil
	}
	// 用 jti 作为 key，TTL 设为 token 剩余有效期（过期后自动清理）
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining <= 0 {
		return nil // 已过期，无需拉黑
	}
	return m.rdb.Set(ctx, blacklistKeyPrefix+claims.ID, "1", remaining).Err()
}

// IsBlacklisted 检查 token 是否在黑名单内。
func (m *Manager) IsBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if m.rdb == nil {
		return false, nil
	}
	claims, err := m.Parse(tokenString)
	if err != nil {
		return false, nil // 无法解析的 token 视为未拉黑（调用方会因解析失败拒绝）
	}
	n, err := m.rdb.Exists(ctx, blacklistKeyPrefix+claims.ID).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// TokenType 返回 token 类型（独立字段，不再复用 Subject）。
func (c *Claims) GetTokenType() string { return c.TokenType }
