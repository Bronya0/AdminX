// Package captcha 登录图形验证码。
//
// 渲染交给成熟库 github.com/mojocn/base64Captcha（内嵌字体、逐字符随机旋转/缩放、
// 正弦扭曲，可叠加空心线/粘曲线/噪点，输出 PNG）；本包只负责三件事：
//
//  1. 答案用 crypto/rand 生成：库内 RandText 走 math/rand（按时间播种），作为验证码答案不安全；
//  2. 一次性存储（校验即删除）：有 Redis 用 Redis（原子 GET+DEL，多实例共享、TTL 自动清理），
//     未配置 Redis 时降级到数据库表，本地调试不必先装 Redis；
//  3. 输出 PNG 而非 SVG：SVG 是文本格式，答案就写在 <text> 节点里，
//     抓一次响应就能读出，等于没有验证码。
package captcha

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"adminx/internal/model"
)

const (
	// captchaPrefix Redis key 前缀。
	captchaPrefix = "captcha:"
	// captchaTTL 验证码有效期。
	captchaTTL = 5 * time.Minute
	// storeTimeout Redis 单次操作超时（Store 接口无 context，内部自建）。
	storeTimeout = 2 * time.Second
	// purgeInterval 数据库存储清理过期行的最小间隔（见 dbStore.purgeExpired）。
	purgeInterval = time.Minute

	// 渲染参数：宽高、噪点数、干扰线、答案长度。
	// 难度就改这几项：噪点越多、干扰线种类越多越难认。
	// 干扰线现状：只留粘曲线（细斜线）；空心线是横穿文字的粗白线、正弦线是粗波浪线，
	// 两者最挡字，认字成本高于防机器收益，已关掉。
	captchaWidth  = 240
	captchaHeight = 80
	captchaLength = 4
	captchaNoise  = 4
	captchaLines  = base64Captcha.OptionShowSlimeLine
	// captchaSource 字符集：库内置的“已剔除易混淆字符”集合（无 0/O/1/I/l 等），大小写混合。
	captchaSource = base64Captcha.TxtSimpleCharaters
)

// captchaFonts 渲染字体（库内嵌）。默认会用上全部内嵌字体，其中多款是装饰体
// （3Dumb / Flim-Flam / DeborahFancyDress 等），笔画细碎，叠上干扰线后基本认不出。
// 这里只留笔画饱满清晰的两款：难度应由干扰线和抖动提供，而不是靠字体本身看不清。
var captchaFonts = []string{"RitaSmith.ttf", "chromohv.ttf"}

// errUnavailable 既无 Redis 也无数据库时验证码不可用（调用方必须 fail-closed，不能放行）。
var errUnavailable = errors.New("验证码存储不可用（未配置 Redis 与数据库）")

// store 验证码答案存储。有 Redis 用 Redis，未配置时落库（见 dbStore）。
type store interface {
	// Set 写入答案（同 id 覆盖）。
	Set(id, value string) error
	// take 原子取值并删除：返回空串表示不存在 / 已过期 / 已被取走。
	take(id string) (string, error)
}

// Manager 验证码管理器。
type Manager struct {
	driver *base64Captcha.DriverString
	store  store
}

// NewManager 构造验证码管理器：优先 Redis（多实例共享、TTL 自动回收），
// 未配置 Redis 时降级到数据库表（本地调试不必先装 Redis），
// 两者都不可用时 Available 返回 false，调用方必须 fail-closed。
func NewManager(rdb *redis.Client, db *gorm.DB) *Manager {
	switch {
	case rdb != nil:
		return &Manager{driver: newDriver(), store: &redisStore{rdb: rdb, ttl: captchaTTL}}
	case db != nil:
		return &Manager{driver: newDriver(), store: &dbStore{db: db, ttl: captchaTTL}}
	default:
		return &Manager{}
	}
}

// newDriver 渲染驱动（字体内嵌在库里，无需外部字体文件）。
func newDriver() *base64Captcha.DriverString {
	return base64Captcha.NewDriverString(captchaHeight, captchaWidth, captchaNoise, captchaLines,
		captchaLength, captchaSource, nil, nil, captchaFonts)
}

// Available 验证码服务是否可用。
func (m *Manager) Available() bool { return m.driver != nil && m.store != nil }

// Generate 生成验证码，返回 (captchaID, "data:image/png;base64,...")。
func (m *Manager) Generate() (string, string, error) {
	if !m.Available() {
		return "", "", errUnavailable
	}
	text, err := secureText(captchaLength, captchaSource)
	if err != nil {
		return "", "", err
	}
	item, err := m.driver.DrawCaptcha(text)
	if err != nil {
		return "", "", fmt.Errorf("渲染验证码失败: %w", err)
	}
	id := uuid.NewString()
	if err := m.store.Set(id, text); err != nil {
		return "", "", err
	}
	// EncodeB64string 返回的就是完整 data URI（data:image/png;base64,...），不要再拼前缀
	return id, item.EncodeB64string(), nil
}

// Verify 校验验证码。无论对错都会消费掉该 id（一次性，防重放）。
// 返回 error 表示存储不可用（调用方应报 503），而不是“验证码错误”。
func (m *Manager) Verify(captchaID, text string) (bool, error) {
	if strings.TrimSpace(captchaID) == "" || strings.TrimSpace(text) == "" {
		return false, nil
	}
	if !m.Available() {
		return false, errUnavailable
	}
	stored, err := m.store.take(captchaID)
	if err != nil {
		return false, err
	}
	if stored == "" {
		return false, nil // 不存在 / 已过期 / 已被消费
	}
	// 大小写不敏感：字符集含大小写，要求用户区分大小写没有意义
	return strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(text)), nil
}

// secureText 从 charset 中取 n 个字符（crypto/rand，无模偏）。
func secureText(n int, charset string) (string, error) {
	runes := []rune(charset)
	if len(runes) == 0 || n <= 0 {
		return "", errors.New("验证码字符集为空")
	}
	out := make([]rune, n)
	max := big.NewInt(int64(len(runes)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("读取随机源失败: %w", err)
		}
		out[i] = runes[idx.Int64()]
	}
	return string(out), nil
}

// takeScript 取值与删除必须原子完成：分两步时，并发提交同一个 captcha_id
// 会有多个请求同时读到尚未删除的值，“一次性”保证失效。
// 取不到时返回空串（而非 nil），调用方不必依赖 Redis 的 nil 语义。
var takeScript = redis.NewScript(`
	local v = redis.call("GET", KEYS[1])
	if not v then return "" end
	redis.call("DEL", KEYS[1])
	return v
`)

// redisStore base64Captcha 的 Redis 存储实现（多实例共享、TTL 到期自动清理）。
type redisStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// Set 写入答案并设置 TTL。
func (s *redisStore) Set(id, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	defer cancel()
	return s.rdb.Set(ctx, captchaPrefix+id, value, s.ttl).Err()
}

// Get 实现 base64Captcha.Store：clear=true 时取值并删除。
// 出错返回空串（库内 Verify 只看字符串、无法区分错误），本包校验一律走 take。
func (s *redisStore) Get(id string, clear bool) string {
	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	defer cancel()
	if !clear {
		v, err := s.rdb.Get(ctx, captchaPrefix+id).Result()
		if err != nil {
			return ""
		}
		return v
	}
	v, err := takeScript.Run(ctx, s.rdb, []string{captchaPrefix + id}).Text()
	if err != nil {
		return ""
	}
	return v
}

// Verify 实现 base64Captcha.Store（本包未使用，仅为完整实现接口）。
func (s *redisStore) Verify(id, answer string, clear bool) bool {
	return strings.EqualFold(strings.TrimSpace(s.Get(id, clear)), strings.TrimSpace(answer))
}

// take 原子取值并删除。错误与“验证码不存在”必须区分：
// 前者要 fail-closed 报服务不可用，后者只是普通校验失败。
func (s *redisStore) take(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	defer cancel()
	return takeScript.Run(ctx, s.rdb, []string{captchaPrefix + id}).Text()
}

// dbStore 未配置 Redis 时的答案存储（表 captcha_codes）。
// 与 Redis 的差别：没有 TTL 自动回收（写入时顺手清理过期行），没有原子脚本
// （靠删除的影响行数保证一次性）。适合本地调试与单机低流量，生产仍应配 Redis。
type dbStore struct {
	db  *gorm.DB
	ttl time.Duration
	// lastPurge 上次清理过期行的毫秒时间戳（atomic：并发请求共用同一个 store）。
	lastPurge atomic.Int64
}

// purgeExpired 清理过期行：最多每分钟一次，失败只当家务没做。
// 两条硬约束：① 清理是顺手做的家务，绝不能因为它失败就让验证码生成失败
// （那会让登录页直接卡死，正是本次要修的问题）；② 不能每次生成都写一遍，
// SQLite 单写者下会与其它写入互相等待（实测会 5s 后报 database is locked）。
func (s *dbStore) purgeExpired(now time.Time) {
	last := s.lastPurge.Load()
	if last != 0 && now.UnixMilli()-last < purgeInterval.Milliseconds() {
		return
	}
	s.lastPurge.Store(now.UnixMilli())
	_ = s.db.Where("expires_at < ?", now).Delete(&model.CaptchaCode{}).Error
}

// Set 写入答案，并顺手清理过期行（没有 TTL 回收，不清理会一直堆积）。
func (s *dbStore) Set(id, value string) error {
	now := time.Now()
	s.purgeExpired(now)
	return s.db.Create(&model.CaptchaCode{ID: id, Answer: value, ExpiresAt: now.Add(s.ttl)}).Error
}

// take 取值并删除。删除影响 0 行说明已被并发请求取走，按“不存在”处理：
// 并发提交同一个 captcha_id 时只有一个能删到行，一次性保证不丢。
func (s *dbStore) take(id string) (string, error) {
	var row model.CaptchaCode
	err := s.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	res := s.db.Where("id = ?", id).Delete(&model.CaptchaCode{})
	if res.Error != nil {
		return "", res.Error
	}
	if res.RowsAffected == 0 {
		return "", nil
	}
	if time.Now().After(row.ExpiresAt) {
		return "", nil // 已过期：行已删除，值不再返回
	}
	return row.Answer, nil
}
