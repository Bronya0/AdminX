package repository

import (
	"time"

	"gorm.io/gorm"

	"adminx/internal/model"
)

// AuditRepo 审计日志数据访问。
type AuditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) *AuditRepo { return &AuditRepo{db: db} }

// Create 写入审计日志。
func (r *AuditRepo) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

// List 分页查询审计日志。
func (r *AuditRepo) List(offset, limit int, action, model_ string, start, end *time.Time) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var count int64
	q := r.db.Model(&model.AuditLog{})
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if model_ != "" {
		q = q.Where("model_name LIKE ?", "%"+model_+"%")
	}
	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("created_at <= ?", *end)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, count, err
}
