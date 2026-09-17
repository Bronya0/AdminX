package repository

import (
	"gorm.io/gorm"

	"adminx/internal/model"
)

// ClusterRepo 集群管理数据访问。
type ClusterRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) *ClusterRepo { return &ClusterRepo{db: db} }

// ── ClusterNode ──

// ListNodes 节点列表（支持 search/status 过滤）。
func (r *ClusterRepo) ListNodes(offset, limit int, search, status string) ([]model.ClusterNode, int64, error) {
	var nodes []model.ClusterNode
	var count int64
	q := r.db.Model(&model.ClusterNode{})
	if search != "" {
		like := "%" + EscapeLike(search) + "%"
		q = q.Where("name LIKE ? OR host LIKE ?", like, like)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("name ASC").Offset(offset).Limit(limit).Find(&nodes).Error
	return nodes, count, err
}

// FindNodeByID 按 ID 查询节点。
func (r *ClusterRepo) FindNodeByID(id string) (*model.ClusterNode, error) {
	var n model.ClusterNode
	err := r.db.First(&n, "id = ?", id).Error
	return &n, err
}

// ListAllNodes 查询全部节点（概览用）。
func (r *ClusterRepo) ListAllNodes() ([]model.ClusterNode, error) {
	var nodes []model.ClusterNode
	err := r.db.Order("name ASC").Find(&nodes).Error
	return nodes, err
}

// CountNodesByStatus 按状态统计节点数（键为 status，值条数）。
func (r *ClusterRepo) CountNodesByStatus() (map[string]int64, error) {
	var rows []struct {
		Status string
		Total  int64
	}
	err := r.db.Model(&model.ClusterNode{}).
		Select("status, count(*) AS total").Group("status").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Total
	}
	return out, nil
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

// FindComponentByID 按 ID 查询业务组件。
func (r *ClusterRepo) FindComponentByID(id string) (*model.ServiceComponent, error) {
	var c model.ServiceComponent
	err := r.db.First(&c, "id = ?", id).Error
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
