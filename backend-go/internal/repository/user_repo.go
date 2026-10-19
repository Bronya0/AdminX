// Package repository 数据访问层（薄封装 GORM）。
package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"adminx/internal/model"
)

// ErrLastSuperuser 本次更新或删除会让系统不再有启用状态的超级管理员。
var ErrLastSuperuser = errors.New("操作后系统不再有启用状态的超级管理员")

// UserRepo 用户数据访问。
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// WithTx 返回绑定到事务 tx 的新 Repo（读/写均走同一事务）。
func (r *UserRepo) WithTx(tx *gorm.DB) *UserRepo { return &UserRepo{db: tx} }

// UpdateColumns 按列更新用户（只写 cols 白名单列，不触碰其他并发单列）。
func (r *UserRepo) UpdateColumns(id string, cols map[string]interface{}) error {
	if len(cols) == 0 {
		return nil
	}
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(cols).Error
}

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

// FindAnyByUsername 按用户名查询（含软删除记录）。
// 写入场景用：只看未删除记录会把"已删除但用户名仍被占"的账号当成不存在，
// 进而在 INSERT 时撞上唯一索引（username 唯一索引不区分软删除）报 500。
func (r *UserRepo) FindAnyByUsername(username string) (*model.User, error) {
	var u model.User
	if err := r.db.Unscoped().Where("username = ?", username).First(&u).Error; err != nil {
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
		like := "%" + EscapeLike(search) + "%"
		// "desc" 必须加引号：desc 是 PostgreSQL 保留字，裸写会让 PG 报 42601
		// （sqlite 容忍，所以本地开发看不出来）
		q = q.Where("username LIKE ? ESCAPE '\\' OR email LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR \"desc\" LIKE ? ESCAPE '\\'", like, like, like, like)
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

// SaveAndMaybeRevoke 按列更新用户；revokeTokens 为真时在同一事务内更新 last_logout。
// 只写 cols 白名单列，不做整行 Save：整行写回会覆盖 last_activity/last_login 等并发单列更新。
// guardSuperuser 为真时，事务内要求写完之后系统仍有启用中的超管，否则整体回滚。
func (r *UserRepo) SaveAndMaybeRevoke(id string, cols map[string]interface{}, revokeTokens, guardSuperuser bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(cols) > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", id).Updates(cols).Error; err != nil {
				return err
			}
		}
		if guardSuperuser {
			if err := ensureSuperuserRemains(tx); err != nil {
				return err
			}
		}
		if !revokeTokens {
			return nil
		}
		return tx.Model(&model.User{}).Where("id = ?", id).
			Update("last_logout", time.Now()).Error
	})
}

// UpdatePassword 仅更新密码字段。
func (r *UserRepo) UpdatePassword(id, hashedPassword string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

// AssignRoles 替换用户的角色关联。
// AssignRoles 设置用户的角色列表。
// roleRefs 接受角色 ID（uuid）或角色名称混传（前端用户管理页提交的是名称）。
func (r *UserRepo) AssignRoles(userID string, roleRefs []string) error {
	var u model.User
	if err := r.db.First(&u, "id = ?", userID).Error; err != nil {
		return err
	}
	var roles []model.Role
	if len(roleRefs) > 0 {
		ids, names := SplitRoleRefs(roleRefs)
		q := r.db.Model(&model.Role{})
		switch {
		case len(ids) > 0 && len(names) > 0:
			q = q.Where("id IN ? OR name IN ?", ids, names)
		case len(ids) > 0:
			q = q.Where("id IN ?", ids)
		default:
			q = q.Where("name IN ?", names)
		}
		if err := q.Find(&roles).Error; err != nil {
			return err
		}
	}
	// Association("Roles").Replace 会清除旧关联并写入新关联
	return r.db.Model(&u).Association("Roles").Replace(roles)
}

// SplitRoleRefs 把角色引用列表拆分为 uuid ID 与名称两组（uuid 列不能与任意字符串比较）。
func SplitRoleRefs(refs []string) (ids []string, names []string) {
	ids = make([]string, 0, len(refs))
	names = make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if _, err := uuid.Parse(ref); err == nil {
			ids = append(ids, ref)
		} else {
			names = append(names, ref)
		}
	}
	return ids, names
}

// SoftDelete 软删除用户。guardSuperuser 为真时要求删完之后系统仍有启用中的超管。
func (r *UserRepo) SoftDelete(id string, guardSuperuser bool) error {
	if !guardSuperuser {
		return r.db.Delete(&model.User{}, "id = ?", id).Error
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.User{}, "id = ?", id).Error; err != nil {
			return err
		}
		return ensureSuperuserRemains(tx)
	})
}

// ensureSuperuserRemains 事务内校验：写完/删完之后仍至少剩一个启用中的超管。
// 校验放在写之后而不是写之前：并发下（两个超管同时互相降级）后提交的事务能看到
// 前一个的结果，不会出现两人各自都看到“还有两个”而双双放行。
func ensureSuperuserRemains(tx *gorm.DB) error {
	var n int64
	if err := tx.Model(&model.User{}).
		Where("is_superuser = ? AND is_active = ?", true, true).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrLastSuperuser
	}
	return nil
}

// UpdateActivity 更新最后活动时间。
// 时间由 Go 侧传入而非 SQL NOW()：postgres / sqlite 通用，且与 JWT 签发时钟同源
// （JWTAuth 用 last_logout 与 token 的 iat 比较，同源可避免库与应用的时钟偏差）。
func (r *UserRepo) UpdateActivity(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_activity", time.Now()).Error
}

// UpdateLogout 更新登出时间。
func (r *UserRepo) UpdateLogout(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_logout", time.Now()).Error
}

// UpdateLogin 更新最后登录时间。
func (r *UserRepo) UpdateLogin(id string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).
		Update("last_login", time.Now()).Error
}
