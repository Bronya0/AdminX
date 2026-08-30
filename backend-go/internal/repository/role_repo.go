package repository

import (
	"errors"

	"gorm.io/gorm"

	"adminx/internal/model"
)

var ErrSystemRoleNotDeletable = errors.New("system role cannot be deleted")

// RoleRepo 角色数据访问。
type RoleRepo struct {
	db *gorm.DB
}

func NewRoleRepo(db *gorm.DB) *RoleRepo { return &RoleRepo{db: db} }

func (r *RoleRepo) FindByID(id string) (*model.Role, error) {
	var role model.Role
	err := r.db.Preload("Menus").First(&role, "id = ?", id).Error
	return &role, err
}

func (r *RoleRepo) FindByName(name string) (*model.Role, error) {
	var role model.Role
	err := r.db.First(&role, "name = ?", name).Error
	return &role, err
}

func (r *RoleRepo) List(offset, limit int, search string) ([]model.Role, int64, error) {
	var roles []model.Role
	var count int64
	q := r.db.Model(&model.Role{})
	if search != "" {
		q = q.Where("name LIKE ? OR desc LIKE ?", "%"+EscapeLike(search)+"%", "%"+EscapeLike(search)+"%")
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&roles).Error
	return roles, count, err
}

func (r *RoleRepo) All() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Order("name ASC").Find(&roles).Error
	return roles, err
}

func (r *RoleRepo) Create(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepo) Update(role *model.Role) error {
	return r.db.Save(role).Error
}

func (r *RoleRepo) Delete(id string) error {
	// 系统角色不允许删除
	var role model.Role
	if err := r.db.First(&role, "id = ?", id).Error; err != nil {
		return err
	}
	if role.IsSystem {
		return ErrSystemRoleNotDeletable
	}
	// 清除菜单关联后删除
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&role).Association("Menus").Clear(); err != nil {
			return err
		}
		return tx.Delete(&role).Error
	})
}

// AssignMenus 替换角色的菜单关联。
func (r *RoleRepo) AssignMenus(roleID string, menuIDs []int64) error {
	var role model.Role
	if err := r.db.First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}
	var menus []model.Menu
	if len(menuIDs) > 0 {
		if err := r.db.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
			return err
		}
	}
	return r.db.Model(&role).Association("Menus").Replace(menus)
}

// MenuIDsByRole 查角色的菜单 ID 列表。
func (r *RoleRepo) MenuIDsByRole(roleID string) ([]int64, error) {
	var ids []int64
	err := r.db.Raw(`SELECT menu_id FROM role_menus WHERE role_id = ?`, roleID).Scan(&ids).Error
	return ids, err
}
