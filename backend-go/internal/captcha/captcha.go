// Package captcha 图片验证码（简化实现: Redis 存储文本 + 返回 ID）。
//
// 对齐 Django captcha: 生成验证码 → 存 Redis → 返回 captcha_id + 图片。
// 这里简化为只返回文本 ID（前端可后续接 SVG 渲染库）。
package captcha

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	captchaPrefix = "captcha:"
	captchaTTL    = 5 * time.Minute
)

// Manager 验证码管理器。
type Manager struct {
	rdb *redis.Client
}

func NewManager(rdb *redis.Client) *Manager { return &Manager{rdb: rdb} }

// Generate 生成验证码。返回 (captchaID, text)。
// text 应由前端渲染成图片，这里只返回明文（前端可自行用 canvas 渲染干扰线）。
func (m *Manager) Generate(ctx context.Context) (string, string, error) {
	text := randomText(4)
	captchaID := uuid.New().String()

	if m.rdb != nil {
		if err := m.rdb.Set(ctx, captchaPrefix+captchaID, text, captchaTTL).Err(); err != nil {
			return "", "", err
		}
	}
	return captchaID, text, nil
}

// Verify 校验验证码（一次性，校验后删除）。
// 无 Redis 时拒绝校验（返回 false + error），避免验证码被绕过。
func (m *Manager) Verify(ctx context.Context, captchaID, text string) (bool, error) {
	if m.rdb == nil {
		return false, fmt.Errorf("验证码服务不可用（未配置 Redis）")
	}
	key := captchaPrefix + captchaID
	stored, err := m.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// 一次性: 校验后删除（无论对错）
	_ = m.rdb.Del(ctx, key).Err()
	// 大小写不敏感
	return equalFold(stored, text), nil
}

func randomText(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 去掉易混淆字符
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'a' && ca <= 'z' {
			ca -= 32
		}
		if cb >= 'a' && cb <= 'z' {
			cb -= 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
