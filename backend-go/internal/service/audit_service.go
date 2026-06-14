package service

import (
	"time"

	"gorm.io/datatypes"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

// AuditService 审计日志服务。
type AuditService struct {
	repo *repository.AuditRepo
}

func NewAuditService(repo *repository.AuditRepo) *AuditService {
	return &AuditService{repo: repo}
}

// List 分页查询审计日志。
func (s *AuditService) List(offset, limit int, action, model_ string, start, end *time.Time) ([]model.AuditLog, int64, error) {
	return s.repo.List(offset, limit, action, model_, start, end)
}

// Record 记录一条审计日志（通用入口）。
func (s *AuditService) Record(action, modelName, objectID, objectRepr, operator, ip string, oldVals, newVals datatypes.JSON, diff string) error {
	log := &model.AuditLog{
		Action:      action,
		ModelName:   modelName,
		ObjectID:    objectID,
		ObjectRepr:  objectRepr,
		Operator:    operator,
		OperatorIP:  ip,
		OldValues:   oldVals,
		NewValues:   newVals,
		DiffSummary: truncate(diff, 500),
	}
	return s.repo.Create(log)
}
