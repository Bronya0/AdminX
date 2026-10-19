package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/service"
)

// NotificationHandler 通知中心 HTTP 处理器。
type NotificationHandler struct {
	audit  *service.AuditService
	svc    *service.NotificationService
	logger *slog.Logger
}

func NewNotificationHandler(svc *service.NotificationService, audit *service.AuditService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, audit: audit, logger: logger}
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
	auditOperation(c, h.audit, "update", "NotificationRead", c.Param("id"), "通知已读", "标记已读", nil, gin.H{"user_id": uid, "is_read": true})
	response.OK(c, nil)
}

// MarkAllRead POST /notification/messages/mark_all_read/
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	count, err := h.svc.UnreadCount(uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	if err := h.svc.MarkAllRead(uid); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditOperation(c, h.audit, "update", "NotificationRead", uid, "全部通知已读", "全部标记已读", gin.H{"unread_count": count}, gin.H{"unread_count": 0})
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
	var in service.WebhookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.CreateWebhook(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "create", "Webhook", result.ID, result.Name, nil, result)
	response.Created(c, result)
}

func (h *NotificationHandler) UpdateWebhook(c *gin.Context) {
	var in service.WebhookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	id := c.Param("id")
	old, err := h.svc.GetWebhook(id)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	result, err := h.svc.UpdateWebhook(id, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	operation := "更新 Webhook"
	if in.Secret != "" {
		operation += "；轮换签名密钥"
	}
	auditOperation(c, h.audit, "update", "Webhook", result.ID, result.Name, operation, old, result)
	response.OK(c, result)
}

func (h *NotificationHandler) DeleteWebhook(c *gin.Context) {
	old, err := h.svc.GetWebhook(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	if err := h.svc.DeleteWebhook(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "delete", "Webhook", old.ID, old.Name, old, nil)
	response.NoContent(c)
}

// TestWebhook POST /notification/webhooks/:id/test/
//
// 返回真实投递结果（ok / 状态码 / 失败原因），由前端如实提示：
// 这个动作的价值就在于把“对方到底收没收到”暴露出来，不能一律回 200 报成功。
func (h *NotificationHandler) TestWebhook(c *gin.Context) {
	result, err := h.svc.TestWebhook(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditOperation(c, h.audit, "execute", "Webhook", c.Param("id"), c.Param("id"), "测试投递", nil, result)
	response.OK(c, result)
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
