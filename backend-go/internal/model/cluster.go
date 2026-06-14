package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClusterNode 集群节点。
type ClusterNode struct {
	ID            string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name          string     `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Host          string     `gorm:"size:255;not null" json:"host"`
	Port          int        `gorm:"not null;default:8000" json:"port"`
	Role          string     `gorm:"size:32;not null;default:'slave'" json:"role"` // master/slave
	Status        string     `gorm:"size:20;not null;default:'offline'" json:"status"` // online/offline/maintenance
	Version       string     `gorm:"size:32;not null;default:''" json:"version"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
	IsActive      bool       `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ClusterNode) TableName() string { return "cluster_nodes" }

func (n *ClusterNode) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

// ServiceComponent 业务组件注册表（三方业务系统注册，心跳 60s TTL）。
type ServiceComponent struct {
	ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AppLabel         string         `gorm:"size:128;uniqueIndex;not null" json:"app_label"`
	Name             string         `gorm:"size:255;not null" json:"name"`
	Version          string         `gorm:"size:64;not null;default:''" json:"version"`
	Host             string         `gorm:"size:255;not null;default:''" json:"host"`
	Description      string         `gorm:"type:text;not null;default:''" json:"description"`
	LastHeartbeat    *time.Time     `json:"last_heartbeat"`
	UpgradeVersion   string         `gorm:"size:64;not null;default:''" json:"upgrade_version"`
	UpgradeURL       string         `gorm:"type:text;not null;default:''" json:"upgrade_url"`
	UpgradeChecksum  string         `gorm:"size:64;not null;default:''" json:"upgrade_checksum"`
	UninstallPending bool           `gorm:"not null;default:false" json:"uninstall_pending"`
	ExtraInfo        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb" json:"extra_info"`
	RegisteredAt     time.Time      `gorm:"autoCreateTime" json:"registered_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ServiceComponent) TableName() string { return "service_components" }

func (s *ServiceComponent) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

// Status 业务组件在线状态（计算属性）。
// 60s 内有心跳视为在线。
func (s *ServiceComponent) Status() string {
	if s.LastHeartbeat == nil {
		return "offline"
	}
	return map[bool]string{true: "online", false: "offline"}[s.LastHeartbeat.After(time.Now().Add(-60*time.Second))]
}
