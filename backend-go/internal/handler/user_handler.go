package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/model"
	"adminx/internal/service"
)

// UserHandler 用户 CRUD。
type UserHandler struct {
	svc    *service.UserService
	audit  *service.AuditService
	logger *slog.Logger
}

func NewUserHandler(svc *service.UserService, auditSvc *service.AuditService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, audit: auditSvc, logger: logger}
}

// List GET /accounts/users/
func (h *UserHandler) List(c *gin.Context) {
	p := pagination.Parse(c)
	var isActive *bool
	if v := c.Query("is_active"); v == "true" {
		t := true
		isActive = &t
	} else if v == "false" {
		f := false
		isActive = &f
	}
	search := c.Query("search")
	role := c.Query("role")

	users, count, err := h.svc.List(p.Offset(), p.Size, search, role, isActive)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	next := pagination.NextURL(c, p.Page, p.Size, count)
	prev := pagination.PreviousURL(c, p.Page, p.Size)
	response.Paginated(c, count, next, prev, model.ToUserDTOList(users))
}

// Get GET /accounts/users/:id/
func (h *UserHandler) Get(c *gin.Context) {
	user, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToUserDTO(user))
}

// Create POST /accounts/users/
func (h *UserHandler) Create(c *gin.Context) {
	var in service.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	// 创建超管仅限超级管理员，防止普通管理员提权
	if in.IsSuperuser && !isSuperuser(c) {
		response.Fail(c, 403, "仅超级管理员可创建超管账号")
		return
	}
	// super_admin 角色赋权仅限超级管理员，防止普通管理员借用户管理提权
	if err := h.svc.CheckRoleAssignment(in.Roles, "", isSuperuser(c)); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	user, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecord(c, h.audit, "create", "User", user.ID, user.Username)
	response.Created(c, model.ToUserDTO(user))
}

// Update PUT/PATCH /accounts/users/:id/
func (h *UserHandler) Update(c *gin.Context) {
	var in service.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	// 授予/撤销超管仅限超级管理员，防止普通管理员提权
	if in.IsSuperuser != nil && *in.IsSuperuser && !isSuperuser(c) {
		response.Fail(c, 403, "仅超级管理员可授予超管权限")
		return
	}
	if in.IsSuperuser != nil && !*in.IsSuperuser && !isSuperuser(c) {
		response.Fail(c, 403, "仅超级管理员可撤销超管权限")
		return
	}
	// super_admin 角色的授予/撤销仅限超级管理员（含"改角色会顺带撤掉 super_admin"的场景）
	id := c.Param("id")
	if in.Roles != nil {
		if err := h.svc.CheckRoleAssignment(in.Roles, id, isSuperuser(c)); err != nil {
			response.Error(c, h.logger, err)
			return
		}
	}
	// 变更前快照（用于审计 old/new/diff；失败不影响主流程）
	oldUser, _ := h.svc.GetByID(id)
	user, err := h.svc.Update(id, in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "update", "User", user.ID, user.Username, toUserDTOSafe(oldUser), model.ToUserDTO(user))
	response.OK(c, model.ToUserDTO(user))
}

// Delete DELETE /accounts/users/:id/
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	// 删除前快照（审计留痕）
	oldUser, _ := h.svc.GetByID(id)
	repr := id
	if oldUser != nil {
		repr = oldUser.Username
	}
	if err := h.svc.Delete(id); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "delete", "User", id, repr, toUserDTOSafe(oldUser), nil)
	response.NoContent(c)
}

// 防止 strconv 未使用警告

// toUserDTOSafe nil 安全的 DTO 转换（审计快照用，nil 记录为空对象）。
func toUserDTOSafe(u *model.User) model.UserDTO {
	if u == nil {
		return model.UserDTO{}
	}
	return model.ToUserDTO(u)
}

// isSuperuser 从 gin context 判断当前用户是否为超级管理员。
func isSuperuser(c *gin.Context) bool {
	v, exists := c.Get("is_superuser")
	if !exists {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
