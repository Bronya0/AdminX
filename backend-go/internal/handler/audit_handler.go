package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/service"
)

// AuditHandler 审计日志 HTTP 处理器。
type AuditHandler struct {
	svc    *service.AuditService
	logger *slog.Logger
}

func NewAuditHandler(svc *service.AuditService, logger *slog.Logger) *AuditHandler {
	return &AuditHandler{svc: svc, logger: logger}
}

func (h *AuditHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	action := c.Query("action")
	model_ := c.Query("model")

	var start, end *time.Time
	if v := c.Query("start"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			start = &t
		}
	}
	if v := c.Query("end"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			end = &t
		}
	}

	logs, count, err := h.svc.List(p.Offset(), p.Size, action, model_, start, end)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, logs)
}

func (h *AuditHandler) Get(c *gin.Context) {
	log, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, log)
}
