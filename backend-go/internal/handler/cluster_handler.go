package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"djangoadminx/pkg/pagination"
	"djangoadminx/pkg/response"

	"djangoadminx/internal/model"
	"djangoadminx/internal/service"
)

var _ = model.ClusterNode{}

// ClusterHandler 集群管理 HTTP 处理器。
type ClusterHandler struct {
	svc    *service.ClusterService
	logger *slog.Logger
}

func NewClusterHandler(svc *service.ClusterService, logger *slog.Logger) *ClusterHandler {
	return &ClusterHandler{svc: svc, logger: logger}
}

// ── 节点 ──

func (h *ClusterHandler) ListNodes(c *gin.Context) {
	p := pagination.Parse(c)
	nodes, count, err := h.svc.ListNodes(p.Offset(), p.Size)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, nodes)
}

func (h *ClusterHandler) CreateNode(c *gin.Context) {
	var n model.ClusterNode
	if err := c.ShouldBindJSON(&n); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.CreateNode(&n)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, result)
}

func (h *ClusterHandler) UpdateNode(c *gin.Context) {
	var n model.ClusterNode
	if err := c.ShouldBindJSON(&n); err != nil {
		response.BindingError(c, err)
		return
	}
	n.ID = c.Param("id")
	result, err := h.svc.UpdateNode(&n)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

func (h *ClusterHandler) DeleteNode(c *gin.Context) {
	if err := h.svc.DeleteNode(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}

// ── 业务组件 ──

func (h *ClusterHandler) ListComponents(c *gin.Context) {
	p := pagination.Parse(c)
	items, count, err := h.svc.ListComponents(p.Offset(), p.Size)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	// 转换为 DTO（含计算字段 status，前端需要）
	dtos := make([]model.ServiceComponentDTO, len(items))
	for i := range items {
		dtos[i] = model.ToServiceComponentDTO(&items[i])
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, dtos)
}

// Register POST /cluster/components/register/ (AllowAny)
func (h *ClusterHandler) Register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.Register(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToServiceComponentDTO(result))
}

// Heartbeat POST /cluster/components/heartbeat/ (AllowAny)
func (h *ClusterHandler) Heartbeat(c *gin.Context) {
	var req struct {
		AppLabel string `json:"app_label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.Heartbeat(req.AppLabel)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToServiceComponentDTO(result))
}

// SetUpgrade POST /cluster/components/:id/set_upgrade/
func (h *ClusterHandler) SetUpgrade(c *gin.Context) {
	var req struct {
		Version  string `json:"version"`
		URL      string `json:"url"`
		Checksum string `json:"checksum"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	if err := h.svc.SetUpgrade(c.Param("id"), req.Version, req.URL, req.Checksum); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

func (h *ClusterHandler) CancelUpgrade(c *gin.Context) {
	if err := h.svc.CancelUpgrade(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

func (h *ClusterHandler) SetUninstall(c *gin.Context) {
	if err := h.svc.SetUninstall(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

func (h *ClusterHandler) CancelUninstall(c *gin.Context) {
	if err := h.svc.CancelUninstall(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}

var _ = strconv.Atoi
