package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"djangoadminx/pkg/response"

	"djangoadminx/internal/model"
	"djangoadminx/internal/service"
)

// PolicyHandler 密码策略 HTTP 处理器。
type PolicyHandler struct {
	svc    *service.PolicyService
	logger *slog.Logger
}

func NewPolicyHandler(svc *service.PolicyService, logger *slog.Logger) *PolicyHandler {
	return &PolicyHandler{svc: svc, logger: logger}
}

// GetPolicy GET /policy/policy/
func (h *PolicyHandler) GetPolicy(c *gin.Context) {
	policy, err := h.svc.GetPolicy()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, policy)
}

// UpdatePolicy PUT/PATCH /policy/policy/
func (h *PolicyHandler) UpdatePolicy(c *gin.Context) {
	var p model.PasswordPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BindingError(c, err)
		return
	}
	result, err := h.svc.UpdatePolicy(&p)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, result)
}

// ChangePassword POST /policy/change-password/
func (h *PolicyHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindingError(c, err)
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	if err := h.svc.ChangePassword(uid, req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, h.logger, err)
		return
	}
	response.OK(c, nil)
}
