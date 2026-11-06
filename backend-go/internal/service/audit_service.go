package service

import (
	"time"

	"gorm.io/datatypes"

	"adminx/internal/model"
	"adminx/internal/repository"
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

// GetByID 查询单条审计日志。
func (s *AuditService) GetByID(id string) (*model.AuditLog, error) {
	return s.repo.FindByID(id)
}

// AuditRecord 一条审计记录的字段集合（已脱敏），批量写入时先攒再写。
type AuditRecord struct {
	Action     string
	ModelName  string
	ObjectID   string
	ObjectRepr string
	Operator   string
	OperatorIP string
	OldValues  datatypes.JSON
	NewValues  datatypes.JSON
	Diff       string
}

// toLog 构造审计模型：截断规则集中在 service 层，单条与批量口径一致。
func (r AuditRecord) toLog() *model.AuditLog {
	return &model.AuditLog{
		Action:      r.Action,
		ModelName:   r.ModelName,
		ObjectID:    truncate(r.ObjectID, 128),
		ObjectRepr:  truncate(r.ObjectRepr, 255),
		Operator:    truncate(r.Operator, 128),
		OperatorIP:  r.OperatorIP,
		OldValues:   r.OldValues,
		NewValues:   r.NewValues,
		DiffSummary: truncate(r.Diff, 500),
	}
}

// Record 记录一条审计日志（通用入口）。
func (s *AuditService) Record(action, modelName, objectID, objectRepr, operator, ip string, oldVals, newVals datatypes.JSON, diff string) error {
	return s.RecordMany([]AuditRecord{{
		Action:     action,
		ModelName:  modelName,
		ObjectID:   objectID,
		ObjectRepr: objectRepr,
		Operator:   operator,
		OperatorIP: ip,
		OldValues:  oldVals,
		NewValues:  newVals,
		Diff:       diff,
	}})
}

// RecordOne 写入一条已构造好的审计记录（字段口径与 Record 一致）。
func (s *AuditService) RecordOne(rec AuditRecord) error {
	return s.RecordMany([]AuditRecord{rec})
}

// RecordMany 批量写入审计记录：批量导入等循环场景一次 INSERT 多行，不逐条往返。
func (s *AuditService) RecordMany(records []AuditRecord) error {
	if len(records) == 0 {
		return nil
	}
	logs := make([]*model.AuditLog, 0, len(records))
	for _, rec := range records {
		logs = append(logs, rec.toLog())
	}
	return s.repo.CreateBatch(logs)
}
