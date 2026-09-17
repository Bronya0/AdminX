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

// CommonHandler 通用端点（dashboard/health/site-info/monitor）。
type CommonHandler struct {
	db               *gorm.DB
	monitor          *service.MonitorService
	siteInfo         *service.SiteInfoService
	systemComponents *service.SystemComponentService
	logger           *slog.Logger
}

func NewCommonHandler(
	db *gorm.DB,
	monitor *service.MonitorService,
	siteInfo *service.SiteInfoService,
	systemComponents *service.SystemComponentService,
	logger *slog.Logger,
) *CommonHandler {
	return &CommonHandler{
		db: db, monitor: monitor, siteInfo: siteInfo,
		systemComponents: systemComponents, logger: logger,
	}
}

// Dashboard GET /common/dashboard/stats/ — 仪表盘统计（键名对齐前端 api/dashboard.ts 的 DashboardStats）。
func (h *CommonHandler) Dashboard(c *gin.Context) {
	var userCount, roleCount, menuCount int64
	h.db.Model(&model.User{}).Where("deleted_at IS NULL").Count(&userCount)
	h.db.Model(&model.Role{}).Count(&roleCount)
	h.db.Model(&model.Menu{}).Where("is_active = ?", true).Count(&menuCount)

	// 节点按状态统计（前端顶部三张卡：在线/离线/维护中）
	nodeCounts := make(map[string]int64, 3)
	for _, status := range []string{"online", "offline", "maintenance"} {
		var cnt int64
		h.db.Model(&model.ClusterNode{}).Where("status = ?", status).Count(&cnt)
		nodeCounts[status] = cnt
	}

	// 最近登录日志（前端「最近登录」表格：username / ip / success / message / created_at）
	var recentLogs []model.UserLoginLog
	h.db.Order("created_at DESC").Limit(10).Find(&recentLogs)

	// 实时资源（同时补一条历史采样，供资源趋势图）
	res := h.monitor.CollectResources()

	response.OK(c, gin.H{
		"user_count":        userCount,
		"role_count":        roleCount,
		"menu_count":        menuCount,
		"cpu_usage":         res.CPU.Percent,
		"cpu_cores":         res.CPU.Count,
		"memory_usage":      res.Memory.Percent,
		"memory_total":      res.Memory.Total,
		"memory_used":       res.Memory.Used,
		"online_nodes":      nodeCounts["online"],
		"offline_nodes":     nodeCounts["offline"],
		"maintenance_nodes": nodeCounts["maintenance"],
		"recent_logs":       recentLogs,
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

// Monitor GET /monitor/resources/ — 实时资源快照（并补一条历史采样）。
func (h *CommonHandler) Monitor(c *gin.Context) {
	response.OK(c, h.monitor.CollectResources())
}

// ResourceHistory GET /monitor/resources/history/ — 资源历史趋势（range=1h/6h/24h/7d）。
func (h *CommonHandler) ResourceHistory(c *gin.Context) {
	history, err := h.monitor.History(c.Query("range"), c.Query("interval"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, history)
}

// SystemComponents GET /common/components/ — 系统内置组件（数据库/Redis/调度器）状态。
func (h *CommonHandler) SystemComponents(c *gin.Context) {
	response.OK(c, h.systemComponents.Status(c.Request.Context()))
}
// NetStat GET /monitor/netstat/
func (h *CommonHandler) NetStat(c *gin.Context) {
	stat := h.monitor.GetNetStat()
	response.OK(c, stat)
}

// SiteInfo GET /common/site-info/ — 站点信息（站点名/描述/Logo/空闲超时等）。
// 公共端点：登录页在未登录时也要展示站点名称与背景图。值存配置中心 group=site。
func (h *CommonHandler) SiteInfo(c *gin.Context) {
	info, err := h.siteInfo.Get(c.Request.Context())
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, info)
}

// UpdateSiteInfo POST /common/site-info/ — 保存站点信息（走 RBAC 权限，与配置中心同级别）。
func (h *CommonHandler) UpdateSiteInfo(c *gin.Context) {
	var in service.SiteInfoUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	info, err := h.siteInfo.Update(c.Request.Context(), in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, info)
}
