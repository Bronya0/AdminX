package service

import (
	"time"

	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

// ClusterService 集群管理业务。
type ClusterService struct {
	db   *gorm.DB
	repo *repository.ClusterRepo
}

func NewClusterService(db *gorm.DB, repo *repository.ClusterRepo) *ClusterService {
	return &ClusterService{db: db, repo: repo}
}

// ListNodes 节点列表。
func (s *ClusterService) ListNodes(offset, limit int) ([]model.ClusterNode, int64, error) {
	return s.repo.ListNodes(offset, limit)
}

func (s *ClusterService) CreateNode(n *model.ClusterNode) (*model.ClusterNode, error) {
	if err := s.repo.CreateNode(n); err != nil {
		return nil, apperr.ErrInternal
	}
	return n, nil
}

func (s *ClusterService) UpdateNode(n *model.ClusterNode) (*model.ClusterNode, error) {
	if err := s.repo.UpdateNode(n); err != nil {
		return nil, apperr.ErrInternal
	}
	return n, nil
}

func (s *ClusterService) DeleteNode(id string) error {
	return s.repo.DeleteNode(id)
}

// ── 业务组件注册/心跳 ──

func (s *ClusterService) ListComponents(offset, limit int) ([]model.ServiceComponent, int64, error) {
	return s.repo.ListComponents(offset, limit)
}

// RegisterInput 业务组件注册入参。
type RegisterInput struct {
	AppLabel    string                 `json:"app_label" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Version     string                 `json:"version"`
	Host        string                 `json:"host"`
	Description string                 `json:"description"`
	ExtraInfo   map[string]interface{} `json:"extra_info"`
}

// Register 业务组件注册（幂等：存在则更新，不存在则创建）。
func (s *ClusterService) Register(in RegisterInput) (*model.ServiceComponent, error) {
	existing, err := s.repo.FindComponentByLabel(in.AppLabel)
	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		// 新注册
		c := &model.ServiceComponent{
			AppLabel:    in.AppLabel,
			Name:        in.Name,
			Version:     in.Version,
			Host:        in.Host,
			Description: in.Description,
			ExtraInfo:   mustJSONMap(in.ExtraInfo),
		}
		c.LastHeartbeat = &now
		if err := s.repo.CreateComponent(c); err != nil {
			return nil, apperr.ErrInternal
		}
		return c, nil
	}
	if err != nil {
		return nil, apperr.ErrInternal
	}
	// 已存在，更新
	existing.Name = in.Name
	existing.Version = in.Version
	existing.Host = in.Host
	existing.Description = in.Description
	if in.ExtraInfo != nil {
		existing.ExtraInfo = mustJSONMap(in.ExtraInfo)
	}
	existing.LastHeartbeat = &now
	if err := s.repo.UpdateComponent(existing); err != nil {
		return nil, apperr.ErrInternal
	}
	return existing, nil
}

// Heartbeat 业务组件心跳。
func (s *ClusterService) Heartbeat(appLabel string) (*model.ServiceComponent, error) {
	c, err := s.repo.FindComponentByLabel(appLabel)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(404, "组件未注册: "+appLabel)
		}
		return nil, apperr.ErrInternal
	}
	now := time.Now()
	c.LastHeartbeat = &now
	if err := s.repo.UpdateComponent(c); err != nil {
		return nil, apperr.ErrInternal
	}
	return c, nil
}

func (s *ClusterService) SetUpgrade(id, version, url, checksum string) error {
	var c model.ServiceComponent
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return apperr.ErrNotFound
	}
	c.UpgradeVersion = version
	c.UpgradeURL = url
	c.UpgradeChecksum = checksum
	return s.repo.UpdateComponent(&c)
}

func (s *ClusterService) CancelUpgrade(id string) error {
	var c model.ServiceComponent
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return apperr.ErrNotFound
	}
	c.UpgradeVersion = ""
	c.UpgradeURL = ""
	c.UpgradeChecksum = ""
	return s.repo.UpdateComponent(&c)
}

func (s *ClusterService) SetUninstall(id string) error {
	return s.db.Model(&model.ServiceComponent{}).Where("id = ?", id).
		Update("uninstall_pending", true).Error
}

func (s *ClusterService) CancelUninstall(id string) error {
	return s.db.Model(&model.ServiceComponent{}).Where("id = ?", id).
		Update("uninstall_pending", false).Error
}

// mustJSONMap map → datatypes.JSON。
func mustJSONMap(m map[string]interface{}) datatypesJSON {
	if m == nil {
		return datatypesJSON([]byte("{}"))
	}
	b, _ := jsonMarshal(m)
	return datatypesJSON(b)
}
