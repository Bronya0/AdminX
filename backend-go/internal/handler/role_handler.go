package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/service"
)

// RoleHandler 角色 CRUD。
type RoleHandler struct {
	svc    *service.RoleService
	audit  *service.AuditService
	logger *slog.Logger
}

func NewRoleHandler(svc *service.RoleService, auditSvc *service.AuditService, logger *slog.Logger) *RoleHandler {
	return &RoleHandler{svc: svc, audit: auditSvc, logger: logger}
}

// List GET /accounts/roles/
func (h *RoleHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	roles, count, err := h.svc.List(p.Offset(), p.Size, c.Query("search"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, roles)
}

// All GET /accounts/roles/all/ — 全部角色不分页（给下拉用）。
func (h *RoleHandler) All(c *gin.Context) {
	roles, err := h.svc.All()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, roles)
}

// Get GET /accounts/roles/:id/
func (h *RoleHandler) Get(c *gin.Context) {
	role, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, role)
}

// Create POST /accounts/roles/
func (h *RoleHandler) Create(c *gin.Context) {
	var in service.RoleCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	role, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "create", "Role", role.ID, role.Name)
	response.Created(c, role)
}

// Update PUT/PATCH /accounts/roles/:id/
func (h *RoleHandler) Update(c *gin.Context) {
	var in service.RoleUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	id := c.Param("id")
	// 变更前快照（审计 old/new/diff；失败不影响主流程）
	oldRole, _ := h.svc.GetByID(id)
	role, err := h.svc.Update(id, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "update", "Role", role.ID, role.Name, oldRole, role)
	response.OK(c, role)
}

// AccessibleMenus POST /accounts/roles/accessible_menus/ — 按角色列表（ID/名称）查询可访问菜单。
func (h *RoleHandler) AccessibleMenus(c *gin.Context) {
	var req struct {
		Roles []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	menus, err := h.svc.AccessibleMenus(req.Roles)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, menus)
}

// Delete DELETE /accounts/roles/:id/
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// 删除前快照（审计留痕）
	oldRole, _ := h.svc.GetByID(id)
	repr := id
	if oldRole != nil {
		repr = oldRole.Name
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "delete", "Role", id, repr, oldRole, nil)
	response.NoContent(c)
}
