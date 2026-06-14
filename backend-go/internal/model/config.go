package model

import "time"

// Config 配置中心（自增主键，KV 存储 + 5 种值类型 + AES-GCM 加密）。
type Config struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Key             string    `gorm:"size:255;uniqueIndex;not null" json:"key"`
	Value           string    `gorm:"type:text;not null;default:''" json:"value"`             // 明文或加密后的值
	ValueType       string    `gorm:"size:20;not null;default:'string'" json:"value_type"`    // string/int/bool/json/options
	EncryptedValue  string    `gorm:"size:1024;not null;default:''" json:"encrypted_value"`   // 独立加密槽（兼容历史）
	IsEncrypted     bool      `gorm:"not null;default:false" json:"is_encrypted"`             // 加密开关
	Desc            string    `gorm:"size:500;not null;default:''" json:"desc"`
	Group           string    `gorm:"size:128;not null;default:'default';index:idx_config_group" json:"group"`
	IsActive        bool      `gorm:"not null;default:true;index:idx_config_group" json:"is_active"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Config) TableName() string { return "configs" }

// DisplayValue 返回给前端展示的值（加密配置显示 ********）。
func (c *Config) DisplayValue() string {
	if c.IsEncrypted {
		return "********"
	}
	return c.Value
}
