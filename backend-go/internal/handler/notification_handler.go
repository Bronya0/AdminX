package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"djangoadminx/pkg/pagination"
	"djangoadminx/pkg/response"

	"djangoadminx/internal/model"
	"djangoadminx/internal/service"
)

// NotificationHandler 通知中心 HTTP 处理器。
type NotificationHandler struct {
	svc    *service.NotificationService
	logger *slog.Logger
}

func NewNotificationHandler(svc *service.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, logger: logger}
}

// List GET /notification/messages/
func (h *NotificationHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	unreadOnly := c.Query("unread") == "true"

	items, count, err := h.svc.ListByUser(p.Offset(), p.Size, uid, unreadOnly)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, items)
}

// MarkRead POST /notification/messages/:id/mark_read/
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if err := h.svc.MarkRead(c.Param("id"), uid); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

// MarkAllRead POST /notification/messages/mark_all_read/
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if err := h.svc.MarkAllRead(uid); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

// UnreadCount GET /notification/messages/unread_count/
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	count, err := h.svc.UnreadCount(uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

// ── Webhook 管理 ──

func (h *NotificationHandler) ListWebhooks(c *gin.Context) {
	p := pagination.Parse(c)
	items, count, err := h.svc.ListWebhooks(p.Offset(), p.Size)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, items)
}

func (h *NotificationHandler) CreateWebhook(c *gin.Context) {
	var w model.WebhookConfig
	if err := c.ShouldBindJSON(&w); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.CreateWebhook(&w)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, result)
}

func (h *NotificationHandler) UpdateWebhook(c *gin.Context) {
	var w model.WebhookConfig
	if err := c.ShouldBindJSON(&w); err != nil {
		response.BindingError(c, err)
		return
	}
	w.ID = c.Param("id")
	result, err := h.svc.UpdateWebhook(&w)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

func (h *NotificationHandler) DeleteWebhook(c *gin.Context) {
	if err := h.svc.DeleteWebhook(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}

func (h *NotificationHandler) WebhookLogs(c *gin.Context) {
	p := pagination.Parse(c)
	logs, count, err := h.svc.ListWebhookLogs(p.Offset(), p.Size, c.Query("webhook_id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, logs)
}
