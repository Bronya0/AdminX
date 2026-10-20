package repository

import (
	"errors"

	"gorm.io/gorm"

	"adminx/internal/model"
)

var ErrSystemRoleNotDeletable = errors.New("system role cannot be deleted")
var ErrInvalidMenuIDs = errors.New("one or more menu IDs do not exist")

// RoleRepo 角色数据访问。
type RoleRepo struct {
	db *gorm.DB
}

func NewRoleRepo(db *gorm.DB) *RoleRepo { return &RoleRepo{db: db} }

// WithTx 返回绑定到事务 tx 的新 Repo。
func (r *RoleRepo) WithTx(tx *gorm.DB) *RoleRepo { return &RoleRepo{db: tx} }

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

// RoleListFilter 角色列表筛选条件。
type RoleListFilter struct {
	Search   string
	IsActive *bool
	IsSystem *bool
	MenuCode string // 只看绑定了该菜单（按 code 标识）的角色
}

func (r *RoleRepo) List(offset, limit int, f RoleListFilter) ([]model.Role, int64, error) {
	var roles []model.Role
	var count int64
	q := r.db.Model(&model.Role{})
	if f.Search != "" {
		like := "%" + EscapeLike(f.Search) + "%"
		// "desc" 是 PostgreSQL 保留字，必须加引号（裸写 PG 报 42601，sqlite 容忍）
		// display_name 也要参与：界面上展示的是中文名，只搜 name 会让人以为搜索坏了
		q = q.Where("roles.name LIKE ? ESCAPE '\\' OR roles.display_name LIKE ? ESCAPE '\\' OR roles.\"desc\" LIKE ? ESCAPE '\\'", like, like, like)
	}
	if f.IsActive != nil {
		q = q.Where("roles.is_active = ?", *f.IsActive)
	}
	if f.IsSystem != nil {
		q = q.Where("roles.is_system = ?", *f.IsSystem)
	}
	if f.MenuCode != "" {
		// 同一角色对同一菜单只有一条关联，JOIN 不会放大行数，count 仍准确
		q = q.Joins("JOIN role_menus ON role_menus.role_id = roles.id").
			Joins("JOIN menus ON menus.id = role_menus.menu_id").
			Where("menus.code = ?", f.MenuCode)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Menus").Order("roles.created_at DESC").Offset(offset).Limit(limit).Find(&roles).Error
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
		// 授权是集合语义：重复 ID 不算错误，但不存在的 ID 不能被静默忽略。
		uniqueIDs := make(map[int64]struct{}, len(menuIDs))
		for _, id := range menuIDs {
			uniqueIDs[id] = struct{}{}
		}
		if len(menus) != len(uniqueIDs) {
			return ErrInvalidMenuIDs
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
