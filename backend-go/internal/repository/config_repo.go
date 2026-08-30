package repository

import (
	"gorm.io/gorm"

	"adminx/internal/model"
)

// ConfigRepo 配置中心数据访问。
type ConfigRepo struct {
	db *gorm.DB
}

func NewConfigRepo(db *gorm.DB) *ConfigRepo { return &ConfigRepo{db: db} }

func (r *ConfigRepo) FindByID(id int64) (*model.Config, error) {
	var c model.Config
	err := r.db.First(&c, id).Error
	return &c, err
}

func (r *ConfigRepo) FindByKey(key string) (*model.Config, error) {
	var c model.Config
	err := r.db.Where("key = ? AND is_active = ?", key, true).First(&c).Error
	return &c, err
}

func (r *ConfigRepo) List(offset, limit int, search, group string) ([]model.Config, int64, error) {
	var cfgs []model.Config
	var count int64
	q := r.db.Model(&model.Config{})
	if search != "" {
		q = q.Where("key LIKE ? OR desc LIKE ?", "%"+EscapeLike(search)+"%", "%"+EscapeLike(search)+"%")
	}
	if group != "" {
		q = q.Where("\"group\" = ?", group)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&cfgs).Error
	return cfgs, count, err
}

func (r *ConfigRepo) FindByGroup(group string) ([]model.Config, error) {
	var cfgs []model.Config
	err := r.db.Where("\"group\" = ? AND is_active = ?", group, true).Find(&cfgs).Error
	return cfgs, err
}

// Groups 查询所有不重复的 group。
func (r *ConfigRepo) Groups() ([]string, error) {
	var groups []string
	err := r.db.Model(&model.Config{}).Distinct("\"group\"").Pluck("\"group\"", &groups).Error
	return groups, err
}

func (r *ConfigRepo) Create(c *model.Config) error {
	return r.db.Create(c).Error
}

func (r *ConfigRepo) Update(c *model.Config) error {
	return r.db.Save(c).Error
}

func (r *ConfigRepo) Delete(id int64) error {
	return r.db.Delete(&model.Config{}, id).Error
}
