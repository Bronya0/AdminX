package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/model"
	"adminx/internal/service"
)

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
	nodes, count, err := h.svc.ListNodes(p.Offset(), p.Size, c.Query("search"), c.Query("status"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, nodes)
}

// GetNode GET /cluster/nodes/:id/
func (h *ClusterHandler) GetNode(c *gin.Context) {
	node, err := h.svc.GetNode(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, node)
}

// NodesOverview GET /cluster/nodes/overview/
func (h *ClusterHandler) NodesOverview(c *gin.Context) {
	overview, err := h.svc.NodesOverview()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, overview)
}

func (h *ClusterHandler) CreateNode(c *gin.Context) {
	var in service.NodeCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.CreateNodeInput(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, result)
}

func (h *ClusterHandler) UpdateNode(c *gin.Context) {
	var in service.NodeUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	id := c.Param("id")
	result, err := h.svc.UpdateNode(id, in)
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

// GetComponent GET /cluster/components/:id/
func (h *ClusterHandler) GetComponent(c *gin.Context) {
	component, err := h.svc.GetComponent(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToServiceComponentDTO(component))
}

// DeleteComponent DELETE /cluster/components/:id/
func (h *ClusterHandler) DeleteComponent(c *gin.Context) {
	if err := h.svc.DeleteComponent(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}

// ConfirmUpgrade POST /cluster/components/:id/confirm_upgrade/
// 确认升级指令已被业务侧执行完毕（清除待执行指令）。
func (h *ClusterHandler) ConfirmUpgrade(c *gin.Context) {
	if err := h.svc.ConfirmUpgrade(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
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
//
// 响应契约（业务组件据此触发升级/卸载，见根 AGENTS.md「业务服务对接」）：
//
//	data.upgrade   = {"version","url","checksum"} | null
//	data.uninstall = bool
//
// 平铺的组件字段继续保留，供组件管理页展示；但指令必须以 upgrade/uninstall 下发，
// 否则业务侧读不到（升级/卸载指令会静默失效）。
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
	response.OK(c, model.ToHeartbeatDTO(result))
}

// SetUpgrade POST /cluster/components/:id/set_upgrade/
// 请求字段与组件 DTO 同名（upgrade_version / upgrade_url / upgrade_checksum）：
// 组件管理页用 GET 返回的同名字段回填表单再提交，两端字段名必须一致，
// 否则前端发来的指令在服务端全是零值（静默空操作，接口还返回成功）。
func (h *ClusterHandler) SetUpgrade(c *gin.Context) {
	var req struct {
		Version  string `json:"upgrade_version"`
		URL      string `json:"upgrade_url"`
		Checksum string `json:"upgrade_checksum"`
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

// Unregister POST /cluster/components/unregister/（业务组件注销，幂等）
func (h *ClusterHandler) Unregister(c *gin.Context) {
	var req struct {
		AppLabel string `json:"app_label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	if err := h.svc.Unregister(req.AppLabel); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}
