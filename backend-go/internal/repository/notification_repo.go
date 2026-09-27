package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"adminx/internal/model"
)

// NotificationRepo 通知数据访问。
type NotificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo { return &NotificationRepo{db: db} }

// unreadCond 未读条件：当前用户在 notification_reads 里没有对应记录。
// 已读状态按用户隔离，不能读通知行上的单列（否则广播通知的已读全站共享）。
const unreadCond = `NOT EXISTS (
	SELECT 1 FROM notification_reads nr
	WHERE nr.notification_id = notifications.id AND nr.user_id = ?
)`

// visible 当前用户可见的通知（本人 + 广播）。
func (r *NotificationRepo) visible(userID string) *gorm.DB {
	return r.db.Model(&model.Notification{}).Where("user_id IS NULL OR user_id = ?", userID)
}

// ListByUser 查询用户的通知（含广播 user_id IS NULL）。
func (r *NotificationRepo) ListByUser(offset, limit int, userID string, unreadOnly bool) ([]model.Notification, int64, error) {
	var items []model.Notification
	var count int64

	q := r.visible(userID)
	if unreadOnly {
		q = q.Where(unreadCond, userID)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := r.fillReadFlags(items, userID); err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

// fillReadFlags 回填 is_read（一次 IN 查询，不逐条查库）。
func (r *NotificationRepo) fillReadFlags(items []model.Notification, userID string) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	var readIDs []string
	if err := r.db.Model(&model.NotificationRead{}).
		Where("user_id = ? AND notification_id IN ?", userID, ids).
		Pluck("notification_id", &readIDs).Error; err != nil {
		return err
	}
	read := make(map[string]bool, len(readIDs))
	for _, id := range readIDs {
		read[id] = true
	}
	for i := range items {
		items[i].IsRead = read[items[i].ID]
	}
	return nil
}

func (r *NotificationRepo) FindByID(id string) (*model.Notification, error) {
	var n model.Notification
	err := r.db.First(&n, "id = ?", id).Error
	return &n, err
}

func (r *NotificationRepo) Create(n *model.Notification) error {
	return r.db.Create(n).Error
}

// MarkRead 标记单条已读（当前用户）。
// OnConflict DoNothing：重复标已读幂等，且 POST 重放不会报主键冲突。
func (r *NotificationRepo) MarkRead(id, userID string) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.NotificationRead{NotificationID: id, UserID: userID}).Error
}

// MarkAllRead 把当前用户所有可见未读通知标记为已读（只影响该用户）。
func (r *NotificationRepo) MarkAllRead(userID string) error {
	var ids []string
	if err := r.visible(userID).Where(unreadCond, userID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	rows := make([]model.NotificationRead, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, model.NotificationRead{NotificationID: id, UserID: userID})
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

// UnreadCount 未读数量。
func (r *NotificationRepo) UnreadCount(userID string) (int64, error) {
	var count int64
	err := r.visible(userID).Where(unreadCond, userID).Count(&count).Error
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
