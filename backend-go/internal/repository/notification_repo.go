package repository

import (
	"gorm.io/gorm"

	"djangoadminx/internal/model"
)

// NotificationRepo 通知数据访问。
type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo { return &NotificationRepo{db: db} }

// ListByUser 查询用户的通知（含广播 user_id IS NULL）。
func (r *NotificationRepo) ListByUser(offset, limit int, userID string, unreadOnly bool) ([]model.Notification, int64, error) {
	var items []model.Notification
	var count int64
	q := r.db.Model(&model.Notification{}).
		Where("user_id IS NULL OR user_id = ?", userID)
	if unreadOnly {
		q = q.Where("is_read = ?", false)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

func (r *NotificationRepo) FindByID(id string) (*model.Notification, error) {
	var n model.Notification
	err := r.db.First(&n, "id = ?", id).Error
	return &n, err
}

func (r *NotificationRepo) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

// MarkRead 标记单条已读。
func (r *NotificationRepo) MarkRead(id string) error {
	return r.db.Model(&model.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

// MarkAllRead 标记用户所有未读为已读。
func (r *NotificationRepo) MarkAllRead(userID string) error {
	return r.db.Model(&model.Notification{}).
		Where("(user_id IS NULL OR user_id = ?) AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// UnreadCount 未读数量。
func (r *NotificationRepo) UnreadCount(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).
		Where("(user_id IS NULL OR user_id = ?) AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// ── WebhookConfig ──

func (r *NotificationRepo) ListWebhooks(offset, limit int) ([]model.WebhookConfig, int64, error) {
	var items []model.WebhookConfig
	var count int64
	q := r.db.Model(&model.WebhookConfig{})
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

func (r *NotificationRepo) FindWebhook(id string) (*model.WebhookConfig, error) {
	var w model.WebhookConfig
	err := r.db.First(&w, "id = ?", id).Error
	return &w, err
}

func (r *NotificationRepo) ActiveWebhooks() ([]model.WebhookConfig, error) {
	var items []model.WebhookConfig
	err := r.db.Where("is_active = ?", true).Find(&items).Error
	return items, err
}

func (r *NotificationRepo) CreateWebhook(w *model.WebhookConfig) error {
	return r.db.Create(w).Error
}

func (r *NotificationRepo) UpdateWebhook(w *model.WebhookConfig) error {
	return r.db.Save(w).Error
}

func (r *NotificationRepo) DeleteWebhook(id string) error {
	return r.db.Delete(&model.WebhookConfig{}, "id = ?", id).Error
}

// ── WebhookLog ──

func (r *NotificationRepo) CreateWebhookLog(l *model.WebhookLog) error {
	return r.db.Create(l).Error
}

func (r *NotificationRepo) ListWebhookLogs(offset, limit int, webhookID string) ([]model.WebhookLog, int64, error) {
	var items []model.WebhookLog
	var count int64
	q := r.db.Model(&model.WebhookLog{})
	if webhookID != "" {
		q = q.Where("webhook_id = ?", webhookID)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}
