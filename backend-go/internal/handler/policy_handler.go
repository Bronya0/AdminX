package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"adminx/pkg/response"

	"adminx/internal/service"
)

// PolicyHandler 密码策略 HTTP 处理器。
type PolicyHandler struct {
	audit  *service.AuditService
	svc    *service.PolicyService
	logger *slog.Logger
}

func NewPolicyHandler(svc *service.PolicyService, audit *service.AuditService, logger *slog.Logger) *PolicyHandler {
	return &PolicyHandler{svc: svc, audit: audit, logger: logger}
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
	var in service.PolicyUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BindingError(c, err)
		return
	}
	old, err := h.svc.GetPolicy()
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	result, err := h.svc.UpdatePolicy(in)
	if err != nil {
		response.Error(c, h.logger, err)
		return
	}
	auditRecordDiff(c, h.audit, "update", "PasswordPolicy", "1", "密码策略", old, result)
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
	auditOperation(c, h.audit, "update", "User", uid, c.GetString("username"), "修改密码", nil, gin.H{"password_changed": true, "sessions_revoked": true})
	response.OK(c, nil)
}
