// Package model 定义所有 GORM 数据模型。
//
// 设计约定（独立于 schema，Go 风格）:
//   - UUID 主键: string 类型 + gorm type:uuid + default:gen_random_uuid()
//   - 软删除: 仅 User 使用 gorm.DeletedAt
//   - 时间戳: time.Time + autoCreateTime/autoUpdateTime
//   - 表名: GORM 默认复数（users, roles），自定义时实现 Tabler 接口
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── User ──

// User 用户模型（软删除）。
type User struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string         `gorm:"size:150;uniqueIndex;not null" json:"username"`
	Password     string         `gorm:"size:128;not null" json:"-"` // bcrypt 哈希，不序列化
	Email        string         `gorm:"size:254;not null;default:''" json:"email"`
	Phone        string         `gorm:"size:20;not null;default:''" json:"phone"`
	Avatar       string         `gorm:"size:500;not null;default:''" json:"avatar"`
	Desc         string         `gorm:"type:text;not null;default:''" json:"desc"`
	HomePage     string         `gorm:"size:255;not null;default:''" json:"home_page"`
	IsActive     bool           `gorm:"not null;default:true" json:"is_active"`
	IsStaff      bool           `gorm:"not null;default:false" json:"is_staff"`
	IsSuperuser  bool           `gorm:"not null;default:false" json:"is_superuser"`
	LastLogin    *time.Time     `gorm:"index" json:"last_login"`
	LastLogout   *time.Time     `json:"last_logout"`
	LastActivity *time.Time     `gorm:"index" json:"last_activity"`
	DateJoined   time.Time      `gorm:"autoCreateTime" json:"date_joined"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// M2M: 用户 ↔ 角色（GORM 自动建表 user_roles）
	Roles []Role `gorm:"many2many:user_roles;" json:"roles"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	if u.DateJoined.IsZero() {
		u.DateJoined = time.Now()
	}
	return nil
}

// OnlineWindow 在线判定窗口：最后活动时间在这个窗口内即视为在线。
// 列表的在线/离线筛选复用同一口径，避免筛选结果与行内“在线”标记对不上。
const OnlineWindow = 5 * time.Minute

// IsOnline 在线判定：最后活动时间在窗口内。
func (u *User) IsOnline() bool {
	if u.LastActivity == nil {
		return false
	}
	return u.LastActivity.After(time.Now().Add(-OnlineWindow))
}

// ── Role ──

// Role 角色模型。
type Role struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	// DisplayName 界面展示名（中文）。权限判定与角色绑定一律用 Name 做标识，
	// 展示时用这个：内置角色是 super_admin 这类英文标识，直接拉到列表里没人认得。
	// 自定义角色留空时前端回退显示 Name。
	DisplayName string  `gorm:"size:128;not null;default:''" json:"display_name"`
	Desc      string    `gorm:"type:text;not null;default:''" json:"desc"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	IsSystem  bool      `gorm:"not null;default:false" json:"is_system"` // 系统内置角色不可删
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// M2M: 角色 ↔ 菜单（GORM 自动建表 role_menus）
	Menus []Menu `gorm:"many2many:role_menus;" json:"menus,omitempty"`
	// 反向: 通过 Users 字段可查角色的用户（不默认序列化）
}

func (Role) TableName() string { return "roles" }

// RoleDisplayName 角色在界面上的名字：优先 display_name，未设置时回退标识。
// 展示与搜索各司其职：界面用这个，权限匹配与表单提交用 Name。
func RoleDisplayName(r Role) string {
	if r.DisplayName != "" {
		return r.DisplayName
	}
	return r.Name
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ── LoginLock 登录锁定 ──

// LoginLock 登录失败锁定记录（按用户名，自增主键）。
type LoginLock struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"size:128;uniqueIndex;not null" json:"username"`
	FailedCount  int        `gorm:"not null;default:0" json:"failed_count"`
	LockedAt     *time.Time `json:"locked_at"`
	UnlockedAt   *time.Time `json:"unlocked_at"` // 保留字段，当前逻辑用删除代替
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (LoginLock) TableName() string { return "login_locks" }

// ── UserLoginLog 登录日志 ──

// UserLoginLog 用户登录日志。
type UserLoginLog struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    *string    `gorm:"type:uuid;index" json:"user_id"` // SET_NULL on delete
	Username  string     `gorm:"size:128;not null" json:"username"`
	IP        string     `gorm:"size:39" json:"ip"`
	UserAgent string     `gorm:"type:text;not null;default:''" json:"user_agent"`
	Success   bool       `gorm:"not null;default:true" json:"success"`
	Message   string     `gorm:"size:255;not null;default:''" json:"message"`
	CreatedAt time.Time  `gorm:"index;autoCreateTime" json:"created_at"`
}

func (UserLoginLog) TableName() string { return "user_login_logs" }

func (l *UserLoginLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}
