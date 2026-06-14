package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditLog 操作审计日志（create/update/delete）。
type AuditLog struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Action      string         `gorm:"size:20;not null" json:"action"` // create/update/delete
	ModelName   string         `gorm:"size:128;not null;index" json:"model_name"`
	ObjectID    string         `gorm:"size:128;not null;default:''" json:"object_id"`
	ObjectRepr  string         `gorm:"size:255;not null;default:''" json:"object_repr"`
	Operator    string         `gorm:"size:128;not null;default:'';index" json:"operator"`
	OperatorIP  string         `gorm:"size:39" json:"operator_ip"`
	OldValues   datatypes.JSON `gorm:"type:jsonb" json:"old_values"`
	NewValues   datatypes.JSON `gorm:"type:jsonb" json:"new_values"`
	DiffSummary string         `gorm:"size:500;not null;default:''" json:"diff_summary"`
	CreatedAt   time.Time      `gorm:"index;autoCreateTime" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
