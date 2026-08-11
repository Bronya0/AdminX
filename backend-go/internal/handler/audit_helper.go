package handler

import (
	"github.com/gin-gonic/gin"

	"adminx/internal/service"
)

// auditRecord 记录一条操作审计（create/update/delete）。
// 从 gin context 提取操作人与 IP；审计失败不影响主流程（只留痕，不阻断）。
func auditRecord(c *gin.Context, svc *service.AuditService, action, modelName, objectID, objectRepr string) {
	if svc == nil {
		return
	}
	operator, _ := c.Get("username")
	op, _ := operator.(string)
	if err := svc.Record(action, modelName, objectID, objectRepr, op, c.ClientIP(), nil, nil, ""); err != nil {
		// 审计失败仅记录，不阻断业务
		return
	}
}
