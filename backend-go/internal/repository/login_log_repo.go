package repository

import (
	"gorm.io/gorm"

	"adminx/internal/model"
)

// LoginLogRepo 登录日志数据访问。
type LoginLogRepo struct {
	db *gorm.DB
}

func NewLoginLogRepo(db *gorm.DB) *LoginLogRepo { return &LoginLogRepo{db: db} }

// Create 写入一条登录日志。
func (r *LoginLogRepo) Create(log *model.UserLoginLog) error {
	return r.db.Create(log).Error
}

// List 分页查询登录日志。
func (r *LoginLogRepo) List(offset, limit int, username, ip string, success *bool) ([]model.UserLoginLog, int64, error) {
	var logs []model.UserLoginLog
	var count int64
	q := r.db.Model(&model.UserLoginLog{})
	if username != "" {
		q = q.Where("username LIKE ?", "%"+EscapeLike(username)+"%")
	}
	if ip != "" {
		q = q.Where("ip = ?", ip)
	}
	if success != nil {
		q = q.Where("success = ?", *success)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, count, err
}
