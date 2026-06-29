// Package captcha 图片验证码（SVG 渲染 + Redis 存储）。
//
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

// Generate 生成验证码。返回 (captchaID, svgImg)。
// 返回 SVG 图片，需 Redis 存储验证码文本。
func (m *Manager) Generate(ctx context.Context) (string, string, error) {
	if m.rdb == nil {
		return "", "", fmt.Errorf("验证码服务不可用（未配置 Redis）")
	}
	text := randomText(4)
	captchaID := uuid.New().String()
	svg := renderSVG(text)
	if err := m.rdb.Set(ctx, captchaPrefix+captchaID, text, captchaTTL).Err(); err != nil {
		return "", "", err
	}
	return captchaID, svg, nil
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

// renderSVG 生成简单的 SVG 验证码图片（带干扰线和噪点）。
func renderSVG(text string) string {
	const width, height = 160, 60
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`, width, height)
	svg += `<rect width="100%" height="100%" fill="#f0f0f0"/>`
	// 干扰线
	for i := 0; i < 5; i++ {
		x1, y1 := randomXY(width, height)
		x2, y2 := randomXY(width, height)
		svg += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#ccc" stroke-width="1"/>`, x1, y1, x2, y2)
	}
	// 文字
	for i, ch := range text {
		x := 20 + i*32
		y := 38 + (int(ch)%3)*5 - 5
		svg += fmt.Sprintf(`<text x="%d" y="%d" font-size="32" font-family="monospace" fill="#333">%s</text>`, x, y, string(ch))
	}
	// 噪点
	for i := 0; i < 30; i++ {
		cx, cy := randomXY(width, height)
		svg += fmt.Sprintf(`<circle cx="%d" cy="%d" r="1" fill="#999"/>`, cx, cy)
	}
	svg += `</svg>`
	return svg
}

func randomXY(maxX, maxY int) (int, int) {
	xx, _ := rand.Int(rand.Reader, big.NewInt(int64(maxX)))
	yy, _ := rand.Int(rand.Reader, big.NewInt(int64(maxY)))
	return int(xx.Int64()), int(yy.Int64())
}
