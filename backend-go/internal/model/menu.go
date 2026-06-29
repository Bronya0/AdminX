package model

import (
	"time"

	"gorm.io/datatypes"
)

// Menu 动态菜单（自增主键。
//
// 注意: path 字段是前端路由路径（非 treebeard 物化路径）。
// 树形结构在前端按 path 前缀匹配重建（参考 menuTree.ts）。
type Menu struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Code           string         `gorm:"size:128;uniqueIndex;not null" json:"code"`
	Name           string         `gorm:"size:128;not null" json:"name"`
	Icon           string         `gorm:"size:64;not null;default:''" json:"icon"`
	Path           string         `gorm:"size:512;not null;default:''" json:"path"`
	Component      string         `gorm:"size:512;not null;default:''" json:"component"`
	PermissionCode string         `gorm:"size:256;not null;default:''" json:"permission_code"`
	MenuType       string         `gorm:"size:20;not null;default:'menu'" json:"menu_type"` // menu/button/iframe
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	IsVisible      bool           `gorm:"not null;default:true" json:"is_visible"`
	SortOrder      int            `gorm:"not null;default:0" json:"sort_order"`
	// ParentID 父菜单 ID（自引用，null=顶级）。简化树结构，不用 treebeard。
	ParentID       *int64         `gorm:"index" json:"parent_id"`
	// AllowedPaths 接口路径白名单 JSON（如 ["GET:/api/v1/users/*"]），用于 RBAC 校验。
	// default 用 cast 形式避免 PG jsonb 隐式 cast 失败。
	AllowedPaths   datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb" json:"allowed_paths"`
	// Depth 树深度（1=根），用于前端按层级渲染。
	Depth          int            `gorm:"not null;default:1" json:"depth"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Menu) TableName() string { return "menus" }
