package repository

import (
	"time"

	"gorm.io/gorm"

	"adminx/internal/model"
)

// MonitorRepo 系统资源历史采样数据访问。
type MonitorRepo struct {
	db *gorm.DB
}

func NewMonitorRepo(db *gorm.DB) *MonitorRepo { return &MonitorRepo{db: db} }

// Create 写入一条采样。
func (r *MonitorRepo) Create(s *model.MonitorSample) error {
	return r.db.Create(s).Error
}

// Latest 最近一条采样（无记录时返回 gorm.ErrRecordNotFound）。
func (r *MonitorRepo) Latest() (*model.MonitorSample, error) {
	var s model.MonitorSample
	err := r.db.Order("created_at DESC, id DESC").First(&s).Error
	return &s, err
}

// ListBetween 查询时间窗内的采样（按时间升序）。
func (r *MonitorRepo) ListBetween(from, to time.Time) ([]model.MonitorSample, error) {
	var samples []model.MonitorSample
	err := r.db.Where("created_at >= ? AND created_at <= ?", from, to).
		Order("created_at ASC, id ASC").Find(&samples).Error
	return samples, err
}

// DeleteBefore 清理窗口外的过期采样（控制表体积）。
func (r *MonitorRepo) DeleteBefore(before time.Time) error {
	return r.db.Where("created_at < ?", before).Delete(&model.MonitorSample{}).Error
}
