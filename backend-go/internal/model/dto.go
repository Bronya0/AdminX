// Package model — DTO（Data Transfer Object）。
//
// 直接序列化 GORM model 会暴露内部结构（如密码字段、角色对象数组），
// 且前端期望的扁平字段（role_names、is_online）需要计算。
// DTO 层负责 model → 前端友好结构的转换。
package model

import (
	"encoding/json"
	"time"
)

// UserDTO 用户序列化视图。
type UserDTO struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	Email        string      `json:"email"`
	Phone        string      `json:"phone"`
	Avatar       string      `json:"avatar"`
	Desc         string      `json:"desc"`
	IsActive     bool        `json:"is_active"`
	IsSuperuser  bool        `json:"is_superuser"`
	IsStaff      bool        `json:"is_staff"`
	IsOnline     bool        `json:"is_online"`
	HomePage     string      `json:"home_page"`
	LastLogin    *time.Time  `json:"last_login"`
	LastActivity *time.Time  `json:"last_activity"`
	DateJoined   time.Time   `json:"date_joined"`
	Roles        []string    `json:"roles"`      // 角色名数组（前端期望 string[]）
	RoleNames    []string    `json:"role_names"` // 同 roles，兼容字段
}

// ToUserDTO 将 User model 转为 DTO。
func ToUserDTO(u *User) UserDTO {
	roleNames := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roleNames = append(roleNames, r.Name)
	}
	return UserDTO{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Phone:        u.Phone,
		Avatar:       u.Avatar,
		Desc:         u.Desc,
		IsActive:     u.IsActive,
		IsSuperuser:  u.IsSuperuser,
		IsStaff:      u.IsStaff,
		IsOnline:     u.IsOnline(),
		HomePage:     u.HomePage,
		LastLogin:    u.LastLogin,
		LastActivity: u.LastActivity,
		DateJoined:   u.DateJoined,
		Roles:        roleNames,
		RoleNames:    roleNames,
	}
}

// ToUserDTOList 批量转换。
func ToUserDTOList(users []User) []UserDTO {
	out := make([]UserDTO, len(users))
	for i := range users {
		out[i] = ToUserDTO(&users[i])
	}
	return out
}

// ServiceComponentDTO 业务组件序列化视图（含计算字段 status）。
type ServiceComponentDTO struct {
	ID               string                 `json:"id"`
	AppLabel         string                 `json:"app_label"`
	Name             string                 `json:"name"`
	Version          string                 `json:"version"`
	Host             string                 `json:"host"`
	Description      string                 `json:"description"`
	LastHeartbeat    *time.Time             `json:"last_heartbeat"`
	Status           string                 `json:"status"` // online/offline（计算字段）
	UpgradeVersion   string                 `json:"upgrade_version"`
	UpgradeURL       string                 `json:"upgrade_url"`
	UpgradeChecksum  string                 `json:"upgrade_checksum"`
	UninstallPending bool                   `json:"uninstall_pending"`
	ExtraInfo        map[string]interface{} `json:"extra_info"`
	RegisteredAt     time.Time              `json:"registered_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ToServiceComponentDTO 转换 ServiceComponent 为 DTO（含计算 status）。
func ToServiceComponentDTO(s *ServiceComponent) ServiceComponentDTO {
	extra := map[string]interface{}{}
	if len(s.ExtraInfo) > 0 {
		_ = json.Unmarshal(s.ExtraInfo, &extra)
	}
	return ServiceComponentDTO{
		ID:               s.ID,
		AppLabel:         s.AppLabel,
		Name:             s.Name,
		Version:          s.Version,
		Host:             s.Host,
		Description:      s.Description,
		LastHeartbeat:    s.LastHeartbeat,
		Status:           s.Status(),
		UpgradeVersion:   s.UpgradeVersion,
		UpgradeURL:       s.UpgradeURL,
		UpgradeChecksum:  s.UpgradeChecksum,
		UninstallPending: s.UninstallPending,
		ExtraInfo:        extra,
		RegisteredAt:     s.RegisteredAt,
		UpdatedAt:        s.UpdatedAt,
	}
}
