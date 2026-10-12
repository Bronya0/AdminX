package repository

import (
	"errors"

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

// FindByCode 按 code 查询菜单。
func (r *MenuRepo) FindByCode(code string) (*model.Menu, error) {
	var m model.Menu
	err := r.db.Where("code = ?", code).First(&m).Error
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

// AllActive 查询所有启用菜单（超级管理员按 RBAC“全放行”约定取菜单时用）。
func (r *MenuRepo) AllActive() ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&menus).Error
	return menus, err
}

// MenusByUser 通过 user_id 查关联的活跃菜单（用户角色 → 菜单）。
// 必须过滤停用角色，与 RBAC 判定口径一致：否则停用角色后侧栏仍显示菜单、点进去全是 403。
func (r *MenuRepo) MenusByUser(userID string) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Distinct("menus.*").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_menus.role_id").
		Joins("JOIN roles ON roles.id = role_menus.role_id").
		Where("user_roles.user_id = ? AND menus.is_active = ? AND roles.is_active = ?", userID, true, true).
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

// ErrMenuHasChildren 删除含子菜单的菜单时返回（service 层翻译为 400）。
var ErrMenuHasChildren = errors.New("menu has children")

// CountChildren 统计直接子菜单数量。
func (r *MenuRepo) CountChildren(id int64) (int64, error) {
	var n int64
	if err := r.db.Model(&model.Menu{}).Where("parent_id = ?", id).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// Delete 删除菜单：有子菜单时拒绝；在事务内先清 role_menus 再删行。
func (r *MenuRepo) Delete(id int64) error {
	if n, err := r.CountChildren(id); err != nil {
		return err
	} else if n > 0 {
		return ErrMenuHasChildren
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM role_menus WHERE menu_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Menu{}, id).Error
	})
}
