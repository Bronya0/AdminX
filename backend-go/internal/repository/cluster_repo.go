package repository

import (
	"gorm.io/gorm"

	"djangoadminx/internal/model"
)

// ClusterRepo 集群管理数据访问。
type ClusterRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) *ClusterRepo { return &ClusterRepo{db: db} }

// ── ClusterNode ──

func (r *ClusterRepo) ListNodes(offset, limit int) ([]model.ClusterNode, int64, error) {
	var nodes []model.ClusterNode
	var count int64
	q := r.db.Model(&model.ClusterNode{})
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("name ASC").Offset(offset).Limit(limit).Find(&nodes).Error
	return nodes, count, err
}

func (r *ClusterRepo) CreateNode(n *model.ClusterNode) error {
	return r.db.Create(n).Error
}

func (r *ClusterRepo) UpdateNode(n *model.ClusterNode) error {
	return r.db.Save(n).Error
}

func (r *ClusterRepo) DeleteNode(id string) error {
	return r.db.Delete(&model.ClusterNode{}, "id = ?", id).Error
}

// ── ServiceComponent ──

func (r *ClusterRepo) ListComponents(offset, limit int) ([]model.ServiceComponent, int64, error) {
	var items []model.ServiceComponent
	var count int64
	q := r.db.Model(&model.ServiceComponent{})
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("name ASC").Offset(offset).Limit(limit).Find(&items).Error
	return items, count, err
}

func (r *ClusterRepo) FindComponentByLabel(appLabel string) (*model.ServiceComponent, error) {
	var c model.ServiceComponent
	err := r.db.Where("app_label = ?", appLabel).First(&c).Error
	return &c, err
}

func (r *ClusterRepo) CreateComponent(c *model.ServiceComponent) error {
	return r.db.Create(c).Error
}

func (r *ClusterRepo) UpdateComponent(c *model.ServiceComponent) error {
	return r.db.Save(c).Error
}

func (r *ClusterRepo) DeleteComponent(id string) error {
	return r.db.Delete(&model.ServiceComponent{}, "id = ?", id).Error
}
