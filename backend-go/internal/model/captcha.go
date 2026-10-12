package model

import "time"

// CaptchaCode 登录验证码答案（仅无 Redis 时使用）。
//
// 验证码答案默认存 Redis（多实例共享 + TTL 自动回收）；未配置 Redis 的环境
// （本地调试最常见）降级到这张表，避免"没装 Redis 就登不进系统"。
// 行是一次性的：校验时取出即删除；过期行在每次写入时顺手清理。
type CaptchaCode struct {
	ID     string `gorm:"type:uuid;primaryKey" json:"id"`
	Answer string `gorm:"size:64;not null" json:"-"`
	// ExpiresAt 有效期（Redis 路径由 TTL 承担，这里必须显式判断）。
	ExpiresAt time.Time `gorm:"not null;index:idx_captcha_expires" json:"expires_at"`
}

func (CaptchaCode) TableName() string { return "captcha_codes" }
