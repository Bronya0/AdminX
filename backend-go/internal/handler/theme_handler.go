package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/response"

	"adminx/internal/service"
)

// ThemeHandler 全局主题设置 HTTP 处理器。
type ThemeHandler struct {
	audit  *service.AuditService
	svc    *service.ThemeService
	logger *slog.Logger
}

func NewThemeHandler(svc *service.ThemeService, audit *service.AuditService, logger *slog.Logger) *ThemeHandler {
	return &ThemeHandler{svc: svc, audit: audit, logger: logger}
}

// Get GET /common/theme/ — 全局主题（公共端点：登录页也要先应用主题）。
func (h *ThemeHandler) Get(c *gin.Context) {
	cfg, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, cfg)
}

// Update POST /common/theme/ — 覆盖式保存：请求体里出现的字段覆盖当前值。
func (h *ThemeHandler) Update(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		response.Fail(c, 400, "读取请求体失败")
		return
	}
	old, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	cfg, err := h.svc.Update(c.Request.Context(), raw)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditOperation(c, h.audit, "update", "Theme", "theme.settings", "全局主题", "保存主题", old, cfg)
	response.OK(c, cfg)
}

// Export GET /common/theme/export/ — 导出主题文件内容。
func (h *ThemeHandler) Export(c *gin.Context) {
	file, err := h.svc.ExportFile(c.Request.Context())
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, file)
}

// Import POST /common/theme/import/ — 导入主题文件，按字段覆盖式合并。
func (h *ThemeHandler) Import(c *gin.Context) {
	raw, err := c.GetRawData()
	if err != nil {
		response.Fail(c, 400, "读取请求体失败")
		return
	}
	old, err := h.svc.Get(c.Request.Context())
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	cfg, err := h.svc.ImportFile(c.Request.Context(), raw)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditOperation(c, h.audit, "update", "Theme", "theme.settings", "全局主题", "导入主题", old, cfg)
	response.OK(c, cfg)
}
