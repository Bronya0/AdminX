// Package repository 数据访问层（薄封装 GORM）。
package repository

import (
	"gorm.io/gorm"

	"djangoadminx/internal/model"
)

// UserRepo 用户数据访问。
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// FindByUsername 按用户名查询（含角色）。
func (r *UserRepo) FindByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.Preload("Roles").Where("username = ? AND deleted_at IS NULL", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID 按 ID 查询（含角色）。
func (r *UserRepo) FindByID(id string) (*model.User, error) {
	var u model.User
	err := r.db.Preload("Roles").First(&u, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// List 分页查询用户。
func (r *UserRepo) List(offset, limit int, search, role string, isActive *bool) ([]model.User, int64, error) {
	var users []model.User
	var count int64

	q := r.db.Model(&model.User{}).Where("deleted_at IS NULL")
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("username LIKE ? OR email LIKE ? OR phone LIKE ? OR desc LIKE ?", like, like, like, like)
	}
	if isActive != nil {
		q = q.Where("is_active = ?", *isActive)
	}
	if role != "" {
		q = q.Joins("JOIN user_roles ON user_roles.user_id = users.id").
			Joins("JOIN roles ON roles.id = user_roles.role_id AND roles.name = ?", role)
	}

	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Roles").Order("date_joined DESC").
		Offset(offset).Limit(limit).Find(&users).Error
	return users, count, err
}

// Create 创建用户。
func (r *UserRepo) Create(u *model.User) error {
	return r.db.Create(u).Error
}

// Update 更新用户（全字段）。
func (r *UserRepo) Update(u *model.User) error {
	return r.db.Save(u).Error
}

// UpdatePassword 仅更新密码字段。
func (r *UserRepo) UpdatePassword(id, hashedPassword string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

// AssignRoles 替换用户的角色关联。
func (r *UserRepo) AssignRoles(userID string, roleIDs []string) error {
	var u model.User
	if err := r.db.First(&u, "id = ?", userID).Error; err != nil {
		return err
	}
	var roles []model.Role
	if len(roleIDs) > 0 {
		if err := r.db.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
	}
	// Association("Roles").Replace 会清除旧关联并写入新关联
	return r.db.Model(&u).Association("Roles").Replace(roles)
}

// SoftDelete 软删除用户。
func (r *UserRepo) SoftDelete(id string) error {
	return r.db.Delete(&model.User{}, "id = ?", id).Error
}

// UpdateActivity 更新最后活动时间。
func (r *UserRepo) UpdateActivity(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_activity", gorm.Expr("NOW()")).Error
}

// UpdateLogout 更新登出时间。
func (r *UserRepo) UpdateLogout(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_logout", gorm.Expr("NOW()")).Error
}

// UpdateLogin 更新最后登录时间。
func (r *UserRepo) UpdateLogin(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_login", gorm.Expr("NOW()")).Error
}
