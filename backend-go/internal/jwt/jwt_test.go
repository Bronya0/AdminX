package jwt

import (
	"context"
	"testing"
	"time"
)

func newTestManager() *Manager {
	return New("test-secret-key-for-jwt-testing", "test-issuer",
		30*time.Minute, 7*24*time.Hour, nil)
}

func TestGenerateAccessToken(t *testing.T) {
	mgr := newTestManager()
	token, err := mgr.GenerateAccessToken("user-1", "testuser")
	if err != nil {
		t.Fatalf("GenerateAccessToken 失败: %v", err)
	}
	if token == "" {
		t.Fatal("access token 为空")
	}

	claims, err := mgr.Parse(token)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %s, want user-1", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", claims.Username)
	}
	if claims.GetTokenType() != TokenTypeAccess {
		t.Errorf("TokenType = %s, want access", claims.GetTokenType())
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer = %s, want test-issuer", claims.Issuer)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	mgr := newTestManager()
	token, err := mgr.GenerateRefreshToken("user-1", "testuser")
	if err != nil {
		t.Fatalf("GenerateRefreshToken 失败: %v", err)
	}
	claims, err := mgr.Parse(token)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if claims.GetTokenType() != TokenTypeRefresh {
		t.Errorf("TokenType = %s, want refresh", claims.GetTokenType())
	}
}

func TestParse_InvalidToken(t *testing.T) {
	mgr := newTestManager()
	_, err := mgr.Parse("invalid.token.string")
	if err == nil {
		t.Error("无效 token 应返回 error")
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	mgr := New("secret", "issuer", -1*time.Hour, 7*24*time.Hour, nil)
	token, _ := mgr.GenerateAccessToken("u1", "test")
	_, err := mgr.Parse(token)
	if err == nil {
		t.Error("过期 token 应返回 error")
	}
}

func TestParse_WrongSecret(t *testing.T) {
	mgr1 := New("secret-a", "issuer", 30*time.Minute, 7*24*time.Hour, nil)
	mgr2 := New("secret-b", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	token, _ := mgr1.GenerateAccessToken("u1", "test")
	_, err := mgr2.Parse(token)
	if err == nil {
		t.Error("不同密钥签发的 token 应无法解析")
	}
}

func TestTokenIDUnique(t *testing.T) {
	mgr := newTestManager()
	token1, _ := mgr.GenerateAccessToken("u1", "test")
	token2, _ := mgr.GenerateAccessToken("u1", "test")

	claims1, _ := mgr.Parse(token1)
	claims2, _ := mgr.Parse(token2)
	if claims1.ID == claims2.ID {
		t.Error("两次生成的 token 应有不同的 jti")
	}
}

func TestBlacklistRefreshToken_NoRedis(t *testing.T) {
	mgr := newTestManager()
	refreshToken, _ := mgr.GenerateRefreshToken("u1", "test")

	// 无 Redis，拉黑应静默成功
	err := mgr.BlacklistRefreshToken(context.Background(), refreshToken)
	if err != nil {
		t.Errorf("无 Redis 时拉黑不应报错: %v", err)
	}

	// 无 Redis，黑名单检查应返回 false
	blacklisted, err := mgr.IsBlacklisted(context.Background(), refreshToken)
	if err != nil {
		t.Errorf("无 Redis 时 IsBlacklisted 不应报错: %v", err)
	}
	if blacklisted {
		t.Error("无 Redis 时 IsBlacklisted 应返回 false")
	}
}

func TestBlacklist_InvalidToken(t *testing.T) {
	mgr := newTestManager()
	err := mgr.BlacklistRefreshToken(context.Background(), "invalid-token")
	if err != nil {
		t.Errorf("无效 token 拉黑不应报错: %v", err)
	}
}

func TestClaims_GetTokenType(t *testing.T) {
	c := &Claims{TokenType: TokenTypeAccess}
	if c.GetTokenType() != TokenTypeAccess {
		t.Errorf("GetTokenType = %s", c.GetTokenType())
	}
}
