package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"adminx/pkg/pagination"
	"adminx/pkg/response"

	"adminx/internal/model"
	"adminx/internal/service"
)

// UserHandler 用户 CRUD。
type UserHandler struct {
	svc    *service.UserService
	logger *slog.Logger
}

func NewUserHandler(svc *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{svc: svc, logger: logger}
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
	user, err := h.svc.Create(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.Created(c, model.ToUserDTO(user))
}

// Update PUT/PATCH /accounts/users/:id/
func (h *UserHandler) Update(c *gin.Context) {
	var in service.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	user, err := h.svc.Update(c.Param("id"), in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, model.ToUserDTO(user))
}

// Delete DELETE /accounts/users/:id/
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.NoContent(c)
}

// 防止 strconv 未使用警告
var _ = strconv.Itoa
