package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"djangoadminx/pkg/pagination"
	"djangoadminx/pkg/response"

	"djangoadminx/internal/service"
)

// RoleHandler 角色 CRUD。
type RoleHandler struct {
	svc    *service.RoleService
	logger *slog.Logger
}

func NewRoleHandler(svc *service.RoleService, logger *slog.Logger) *RoleHandler {
	return &RoleHandler{svc: svc, logger: logger}
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
	response.Created(c, role)
}

// Update PUT/PATCH /accounts/roles/:id/
func (h *RoleHandler) Update(c *gin.Context) {
	var in service.RoleUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	role, err := h.svc.Update(c.Param("id"), in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, role)
}

// Delete DELETE /accounts/roles/:id/
func (h *RoleHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}
