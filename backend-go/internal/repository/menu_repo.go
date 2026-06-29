package repository

import (
	"gorm.io/gorm"

	"adminx/internal/model"
)

// MenuRepo 菜单数据访问。
type MenuRepo struct {
	db *gorm.DB
}

func NewMenuRepo(db *gorm.DB) *MenuRepo { return &MenuRepo{db: db} }

func (r *MenuRepo) FindByID(id int64) (*model.Menu, error) {
	var m model.Menu
	err := r.db.First(&m, id).Error
	return &m, err
}

// All 查询所有菜单（按 sort_order, id 排序）。
func (r *MenuRepo) All() ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Order("sort_order ASC, id ASC").Find(&menus).Error
	return menus, err
}

// ActiveByPaths 按前端 path 集合查询活跃菜单（RBAC 用）。
func (r *MenuRepo) ActiveByPaths(paths []string) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Where("is_active = ? AND path IN ?", true, paths).Find(&menus).Error
	return menus, err
}

// MenusByUser 通过 user_id 查关联的活跃菜单（用户角色 → 菜单）。
func (r *MenuRepo) MenusByUser(userID string) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Distinct("menus.*").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_menus.role_id").
		Where("user_roles.user_id = ? AND menus.is_active = ?", userID, true).
		Order("menus.sort_order ASC, menus.id ASC").
		Find(&menus).Error
	return menus, err
}

// Create 创建菜单。
func (r *MenuRepo) Create(m *model.Menu) error {
	return r.db.Create(m).Error
}

// Update 更新菜单。
func (r *MenuRepo) Update(m *model.Menu) error {
	return r.db.Save(m).Error
}

// Delete 删除菜单。
func (r *MenuRepo) Delete(id int64) error {
	return r.db.Delete(&model.Menu{}, id).Error
}
