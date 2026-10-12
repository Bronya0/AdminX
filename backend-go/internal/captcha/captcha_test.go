package captcha

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"adminx/internal/database"
	"adminx/internal/model"
)

// newTestDB 建内存 SQLite 库：复用生产同一份 DDL（顺带守住 captcha_codes 的建表语句）。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("连接 SQLite 失败: %v", err)
	}
	// :memory: 每条连接一个独立库，必须限制为单连接
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}
	for _, stmt := range database.SQLiteStatements() {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表失败: %v\nSQL: %s", err, stmt)
		}
	}
	return db
}

// answerOf 读回某 id 的答案（校验会删行，必须先读）。
func answerOf(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var row model.CaptchaCode
	if err := db.First(&row, "id = ?", id).Error; err != nil {
		t.Fatalf("读取验证码失败: %v", err)
	}
	return row.Answer
}

// TestSecureText 答案必须来自 crypto/rand：长度正确、字符落在字符集内、多次结果不同。
func TestSecureText(t *testing.T) {
	charset := "ABCD23"
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		got, err := secureText(4, charset)
		if err != nil {
			t.Fatalf("secureText 失败: %v", err)
		}
		if len([]rune(got)) != 4 {
			t.Errorf("长度 = %d, want 4", len([]rune(got)))
		}
		for _, ch := range got {
			if !strings.ContainsRune(charset, ch) {
				t.Errorf("字符 %q 不在字符集内", ch)
			}
		}
		seen[got] = true
	}
	// 20 次全同意味着随机源没在工作
	if len(seen) < 2 {
		t.Errorf("20 次生成结果完全相同，随机源异常: %v", seen)
	}
}

// TestSecureText_Invalid 非法入参直接报错（不返回空答案蒙混过关）。
func TestSecureText_Invalid(t *testing.T) {
	if _, err := secureText(4, ""); err == nil {
		t.Error("空字符集应报错")
	}
	if _, err := secureText(0, "ABC"); err == nil {
		t.Error("长度为 0 应报错")
	}
}

// TestManager_NoStore 既无 Redis 也无数据库时必须 fail-closed：生成与校验都报错，不能静默放行。
func TestManager_NoStore(t *testing.T) {
	m := NewManager(nil, nil)
	if m.Available() {
		t.Error("无任何存储时 Available 应为 false")
	}
	if _, _, err := m.Generate(); err == nil {
		t.Error("无存储时 Generate 应报错")
	}
	if _, err := m.Verify("id", "text"); err == nil {
		t.Error("无存储时 Verify 应报错（fail-closed）")
	}
	// 空入参属于“用户没填”，是普通校验失败而非服务故障
	ok, err := m.Verify("", "")
	if err != nil || ok {
		t.Errorf("空入参应返回 (false, nil)，实际 (%v, %v)", ok, err)
	}
}

// TestRenderPNG 渲染链路可用：产出 PNG 的 data URI（内嵌字体加载失败会在此暴露）。
func TestRenderPNG(t *testing.T) {
	m := NewManager(nil, nil)
	m.driver = newDriver()
	text, err := secureText(captchaLength, captchaSource)
	if err != nil {
		t.Fatalf("secureText 失败: %v", err)
	}
	item, err := m.driver.DrawCaptcha(text)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	uri := item.EncodeB64string()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(uri, prefix) {
		t.Fatalf("输出应为 %s 开头的 data URI，实际: %.40s", prefix, uri)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, prefix))
	if err != nil {
		t.Fatalf("data URI 主体不是合法 base64: %v", err)
	}
	// PNG 签名：89 50 4E 47
	if len(raw) < 8 || raw[0] != 0x89 || raw[1] != 'P' || raw[2] != 'N' || raw[3] != 'G' {
		t.Errorf("输出不是 PNG，前 8 字节: % x", raw[:min(8, len(raw))])
	}
}

// TestManager_DBStore 无 Redis 时降级到数据库：生成可用、校验通过、一次性、大小写不敏感。
func TestManager_DBStore(t *testing.T) {
	db := newTestDB(t)
	m := NewManager(nil, db)
	if !m.Available() {
		t.Fatal("无 Redis 但有数据库时应可用（本地调试降级路径）")
	}

	id, image, err := m.Generate()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if id == "" || !strings.HasPrefix(image, "data:image/png;base64,") {
		t.Fatalf("生成结果异常: id=%q image=%.30s", id, image)
	}

	answer := answerOf(t, db, id)
	ok, err := m.Verify(id, strings.ToLower(answer))
	if err != nil || !ok {
		t.Fatalf("正确验证码应通过（大小写不敏感）: ok=%v err=%v", ok, err)
	}

	// 一次性：同一个 id 不能再用
	ok, err = m.Verify(id, answer)
	if err != nil || ok {
		t.Errorf("同一 id 二次校验应失败: ok=%v err=%v", ok, err)
	}
	var n int64
	db.Model(&model.CaptchaCode{}).Where("id = ?", id).Count(&n)
	if n != 0 {
		t.Errorf("校验后应删除该行，实际剩 %d 行", n)
	}
}

// TestDBStore_PurgeRateLimited 清理限频：第一次写入清理过期行，窗口内不再清理，
// 但限频不能让生成失败（家务不能拖垮验证码生成）。
func TestDBStore_PurgeRateLimited(t *testing.T) {
	db := newTestDB(t)
	m := NewManager(nil, db)

	// 首次写入：过期行应被清掉（lastPurge 为 0，不受限频影响）
	if err := db.Create(&model.CaptchaCode{ID: "stale-1", Answer: "AAAA",
		ExpiresAt: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatalf("预置过期行失败: %v", err)
	}
	if _, _, err := m.Generate(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	var n int64
	db.Model(&model.CaptchaCode{}).Where("id = ?", "stale-1").Count(&n)
	if n != 0 {
		t.Error("首次写入应清理过期行")
	}

	// 限频窗口内：不再清理（否则每次生成都多一次写库，SQLite 单写者下会互相等待）
	if err := db.Create(&model.CaptchaCode{ID: "stale-2", Answer: "BBBB",
		ExpiresAt: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatalf("预置过期行失败: %v", err)
	}
	if _, _, err := m.Generate(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	db.Model(&model.CaptchaCode{}).Where("id = ?", "stale-2").Count(&n)
	if n != 1 {
		t.Error("限频窗口内不应再清理过期行")
	}
}

// TestManager_DBStore_Expired 过期验证码拒绝，且写入新验证码时顺手清理过期行。
func TestManager_DBStore_Expired(t *testing.T) {
	db := newTestDB(t)
	m := NewManager(nil, db)

	id, _, err := m.Generate()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	answer := answerOf(t, db, id)
	if err := db.Model(&model.CaptchaCode{}).Where("id = ?", id).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("改有效期失败: %v", err)
	}
	ok, err := m.Verify(id, answer)
	if err != nil || ok {
		t.Errorf("过期验证码应被拒绝: ok=%v err=%v", ok, err)
	}

	// 残留的过期行（用户从没提交过）必须在下一次写入时被清掉
	stale, _, err := m.Generate()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if err := db.Model(&model.CaptchaCode{}).Where("id = ?", stale).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("改有效期失败: %v", err)
	}
	// 清理有最小间隔（家务不能每次生成都做），测试里强制放行一次
	m.store.(*dbStore).lastPurge.Store(0)
	if _, _, err := m.Generate(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	var n int64
	db.Model(&model.CaptchaCode{}).Where("id = ?", stale).Count(&n)
	if n != 0 {
		t.Error("写入新验证码时应清理过期行")
	}
}
