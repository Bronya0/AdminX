package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"adminx/pkg/response"

	"adminx/internal/model"
	"adminx/internal/service"
)

// CommonHandler 通用端点（dashboard/health/site-info）。
type CommonHandler struct {
	db        *gorm.DB
	monitor   *service.MonitorService
	logger    *slog.Logger
}

func NewCommonHandler(db *gorm.DB, monitor *service.MonitorService, logger *slog.Logger) *CommonHandler {
	return &CommonHandler{db: db, monitor: monitor, logger: logger}
}

// Dashboard GET /common/dashboard/stats/
func (h *CommonHandler) Dashboard(c *gin.Context) {
	var userCount, roleCount, auditCount int64
	h.db.Model(&model.User{}).Where("deleted_at IS NULL").Count(&userCount)
	h.db.Model(&model.Role{}).Count(&roleCount)
	h.db.Model(&model.AuditLog{}).Count(&auditCount)

	// 最近审计日志
	var recentLogs []model.AuditLog
	h.db.Order("created_at DESC").Limit(10).Find(&recentLogs)

	response.OK(c, gin.H{
		"user_count":  userCount,
		"role_count":  roleCount,
		"audit_count": auditCount,
		"recent_logs": recentLogs,
	})
}

// Health GET /common/health/（已在 router 注册，这里作为备用）
func (h *CommonHandler) Health(c *gin.Context) {
	dbOK := "up"
	sqlDB, err := h.db.DB()
	if err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if sqlDB.PingContext(ctx) != nil {
			dbOK = "down"
		}
	} else {
		dbOK = "down"
	}
	response.OK(c, gin.H{"status": "ok", "db": dbOK})
}

// Monitor GET /monitor/resources/
func (h *CommonHandler) Monitor(c *gin.Context) {
	res, err := h.monitor.GetResources()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, res)
}

// NetStat GET /monitor/netstat/
func (h *CommonHandler) NetStat(c *gin.Context) {
	stat := h.monitor.GetNetStat()
	response.OK(c, stat)
}
