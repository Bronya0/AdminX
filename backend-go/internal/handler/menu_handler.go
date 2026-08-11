package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"adminx/pkg/response"

	"adminx/internal/service"
)

// MenuHandler 菜单 CRUD + 树查询。
type MenuHandler struct {
	svc    *service.MenuService
	audit  *service.AuditService
	logger *slog.Logger
}

func NewMenuHandler(svc *service.MenuService, auditSvc *service.AuditService, logger *slog.Logger) *MenuHandler {
	return &MenuHandler{svc: svc, audit: auditSvc, logger: logger}
}

// List GET /menu/ — 全部菜单（扁平，前端建树）。
func (h *MenuHandler) List(c *gin.Context) {
	menus, err := h.svc.All()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, menus)
}

// UserTree GET /menu/user_tree/ — 当前用户菜单。
func (h *MenuHandler) UserTree(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	menus, err := h.svc.MenusByUser(uid)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, menus)
}

// Get GET /menu/:id/
func (h *MenuHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的菜单 ID")
		return
	}
	menu, err := h.svc.GetByID(id)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, menu)
}

// Create POST /menu/
func (h *MenuHandler) Create(c *gin.Context) {
	var in service.MenuCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	menu, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "create", "Menu", strconv.FormatInt(menu.ID, 10), menu.Name)
	response.Created(c, menu)
}

// Register POST /menu/register/ — 按 code 幂等注册（业务容器对接用）。
func (h *MenuHandler) Register(c *gin.Context) {
	var in service.MenuCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	menu, err := h.svc.RegisterOrUpdate(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "update", "Menu", strconv.FormatInt(menu.ID, 10), menu.Name)
	response.OK(c, menu)
}

// Update PUT/PATCH /menu/:id/
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的菜单 ID")
		return
	}
	var in service.MenuUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	menu, err := h.svc.Update(id, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "update", "Menu", strconv.FormatInt(menu.ID, 10), menu.Name)
	response.OK(c, menu)
}

// Delete DELETE /menu/:id/
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "无效的菜单 ID")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "delete", "Menu", strconv.FormatInt(id, 10), strconv.FormatInt(id, 10))
	response.NoContent(c)
}
