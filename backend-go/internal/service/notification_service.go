package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

// NotificationService 通知中心。
type NotificationService struct {
	db   *gorm.DB
	repo *repository.NotificationRepo
	log  *slog.Logger
}

func NewNotificationService(db *gorm.DB, repo *repository.NotificationRepo, logger *slog.Logger) *NotificationService {
	return &NotificationService{db: db, repo: repo, log: logger}
}

// CreateInput 创建通知入参。
type NotificationCreateInput struct {
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content"`
	Type       string `json:"notification_type"` // info/success/warning/error
	UserID     string `json:"user_id"`           // 空 = 广播
}

// Create 创建通知 + 异步触发 webhook 外发。
func (s *NotificationService) Create(ctx context.Context, in NotificationCreateInput) (*model.Notification, error) {
	nType := in.Type
	if nType == "" {
		nType = "info"
	}
	n := &model.Notification{
		Title:            in.Title,
		Content:          in.Content,
		NotificationType: nType,
	}
	if in.UserID != "" {
		uid := in.UserID
		n.UserID = &uid
	}
	if err := s.repo.Create(n); err != nil {
		return nil, apperr.ErrInternal
	}

	// 异步触发 webhook（信号驱动，对齐 Django post_save signal）
	go s.dispatchWebhooks(context.Background(), n)
	return n, nil
}

// ListByUser 查询用户通知。
func (s *NotificationService) ListByUser(offset, limit int, userID string, unreadOnly bool) ([]model.Notification, int64, error) {
	return s.repo.ListByUser(offset, limit, userID, unreadOnly)
}

// UnreadCount 未读数。
func (s *NotificationService) UnreadCount(userID string) (int64, error) {
	return s.repo.UnreadCount(userID)
}

// MarkRead 标记单条已读。
func (s *NotificationService) MarkRead(id string) error {
	if err := s.repo.MarkRead(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// MarkAllRead 标记全部已读。
func (s *NotificationService) MarkAllRead(userID string) error {
	if err := s.repo.MarkAllRead(userID); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// ── Webhook 管理 ──

func (s *NotificationService) ListWebhooks(offset, limit int) ([]model.WebhookConfig, int64, error) {
	return s.repo.ListWebhooks(offset, limit)
}

func (s *NotificationService) CreateWebhook(w *model.WebhookConfig) (*model.WebhookConfig, error) {
	if err := s.repo.CreateWebhook(w); err != nil {
		return nil, apperr.ErrInternal
	}
	return w, nil
}

func (s *NotificationService) UpdateWebhook(w *model.WebhookConfig) (*model.WebhookConfig, error) {
	if err := s.repo.UpdateWebhook(w); err != nil {
		return nil, apperr.ErrInternal
	}
	return w, nil
}

func (s *NotificationService) DeleteWebhook(id string) error {
	return s.repo.DeleteWebhook(id)
}

func (s *NotificationService) ListWebhookLogs(offset, limit int, webhookID string) ([]model.WebhookLog, int64, error) {
	return s.repo.ListWebhookLogs(offset, limit, webhookID)
}

// ── Webhook 外发 ──

// dispatchWebhooks 向所有匹配的活跃 webhook 发送通知。
func (s *NotificationService) dispatchWebhooks(ctx context.Context, n *model.Notification) {
	configs, err := s.repo.ActiveWebhooks()
	if err != nil {
		s.log.Error("查询活跃 webhook 失败", "error", err)
		return
	}
	for i := range configs {
		cfg := &configs[i]
		if !matchEvent(cfg.Events, n.NotificationType) {
			continue
		}
		s.sendWebhook(ctx, cfg, n)
	}
}

// sendWebhook 发送单个 webhook（HMAC-SHA256 签名，对齐 Django）。
func (s *NotificationService) sendWebhook(ctx context.Context, cfg *model.WebhookConfig, n *model.Notification) {
	payload := map[string]interface{}{
		"event":      n.NotificationType,
		"title":      n.Title,
		"content":    n.Content,
		"created_at": n.CreatedAt.Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)

	headers := map[string]string{"Content-Type": "application/json"}
	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(body)
		headers["X-Signature"] = hex.EncodeToString(mac.Sum(nil))
	}

	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.URL, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	logEntry := &model.WebhookLog{
		WebhookID: cfg.ID,
	}
	notificationID := n.ID
	logEntry.NotificationID = &notificationID

	resp, err := client.Do(req)
	if err != nil {
		logEntry.Status = "failed"
		logEntry.ErrorMessage = err.Error()
		_ = s.repo.CreateWebhookLog(logEntry)
		s.log.Warn("webhook 发送失败", "name", cfg.Name, "error", err)
		return
	}
	defer resp.Body.Close()

	logEntry.Status = map[bool]string{true: "success", false: "failed"}[resp.StatusCode < 400]
	logEntry.ResponseStatus = &resp.StatusCode
	// 简化: 不读 body 详情

	if err := s.repo.CreateWebhookLog(logEntry); err != nil {
		s.log.Warn("写 webhook 日志失败", "error", err)
	}
	s.log.Info("webhook 已发送", "name", cfg.Name, "status", resp.StatusCode)
}

// matchEvent 检查事件类型是否在 webhook 关注列表中（对齐 Django _matches_events）。
// 空 events 表示匹配所有。
func matchEvent(eventsCSV, eventType string) bool {
	if eventsCSV == "" {
		return true
	}
	for _, e := range strings.Split(eventsCSV, ",") {
		if strings.TrimSpace(e) == eventType {
			return true
		}
	}
	return false
}

var _ = fmt.Sprintf
