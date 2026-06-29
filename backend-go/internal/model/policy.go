package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordPolicy 密码策略（单例，固定 ID=1）。
type PasswordPolicy struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MinLength       int       `gorm:"not null;default:8" json:"min_length"`
	RequireUpper    bool      `gorm:"not null;default:true" json:"require_upper"`
	RequireLower    bool      `gorm:"not null;default:true" json:"require_lower"`
	RequireDigit    bool      `gorm:"not null;default:true" json:"require_digit"`
	RequireSpecial  bool      `gorm:"not null;default:true" json:"require_special"`
	ExpireDays      int       `gorm:"not null;default:90" json:"expire_days"`
	HistoryCount    int       `gorm:"not null;default:5" json:"history_count"` // 密码历史记录数（防重用）
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PasswordPolicy) TableName() string { return "password_policies" }

// BeforeSave 单例强制：ID 恒为 1。
func (p *PasswordPolicy) BeforeSave(tx *gorm.DB) error {
	p.ID = 1
	return nil
}

// Validate 校验密码是否符合策略。返回 nil 表示通过。
func (p *PasswordPolicy) Validate(password string) error {
	if p == nil || !p.IsActive {
		return nil
	}
	if len(password) < p.MinLength {
		return errPolicy("密码长度不能少于 %d 位", p.MinLength)
	}
	if p.RequireUpper && !hasUpper(password) {
		return errPolicy("密码必须包含大写字母")
	}
	if p.RequireLower && !hasLower(password) {
		return errPolicy("密码必须包含小写字母")
	}
	if p.RequireDigit && !hasDigit(password) {
		return errPolicy("密码必须包含数字")
	}
	if p.RequireSpecial && !hasSpecial(password) {
		return errPolicy("密码必须包含特殊字符")
	}
	return nil
}

// ── PasswordHistory 密码历史 ──

// PasswordHistory 用户历史密码（用于防止重用最近 N 个密码）。
type PasswordHistory struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       string    `gorm:"type:uuid;not null;index" json:"user_id"`
	PasswordHash string    `gorm:"size:256;not null" json:"-"` // bcrypt 哈希，不序列化
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (PasswordHistory) TableName() string { return "password_histories" }

func (h *PasswordHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	return nil
}

// errPolicy 密码策略错误。
type policyError struct{ msg string }

func (e *policyError) Error() string { return e.msg }
func errPolicy(format string, args ...interface{}) error {
	return &policyError{msg: fmt.Sprintf(format, args...)}
}

// 字符检查辅助函数。
func hasUpper(s string) bool   { return strings.IndexFunc(s, isUpper) >= 0 }
func hasLower(s string) bool   { return strings.IndexFunc(s, isLower) >= 0 }
func hasDigit(s string) bool   { return strings.IndexFunc(s, isDigit) >= 0 }
func hasSpecial(s string) bool { return strings.IndexFunc(s, isSpecial) >= 0 }

func isUpper(r rune) bool   { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool   { return r >= 'a' && r <= 'z' }
func isDigit(r rune) bool   { return r >= '0' && r <= '9' }
func isSpecial(r rune) bool {
	switch r {
	case '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+', '-', '=',
		'{', '}', '[', ']', '|', '\\', ':', ';', '"', '\'', '<', '>', ',',
		'.', '/', '?', '`', '~':
		return true
	default:
		return false
	}
}
