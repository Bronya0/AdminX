package service

import (
	"gorm.io/gorm"

	apperr "adminx/pkg/errors"
	"adminx/pkg/crypto"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// UserService 用户业务逻辑。
type UserService struct {
	db       *gorm.DB
	userRepo *repository.UserRepo
}

func NewUserService(db *gorm.DB, userRepo *repository.UserRepo) *UserService {
	return &UserService{db: db, userRepo: userRepo}
}

// List 分页查询用户。
func (s *UserService) List(offset, limit int, search, role string, isActive *bool) ([]model.User, int64, error) {
	return s.userRepo.List(offset, limit, search, role, isActive)
}

// GetByID 查询单个用户。
func (s *UserService) GetByID(id string) (*model.User, error) {
	u, err := s.userRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return u, nil
}

// CreateInput 创建用户入参。
type CreateInput struct {
	Username  string   `json:"username" binding:"required,min=2,max=150"`
	Password  string   `json:"password" binding:"required,min=6"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Avatar    string   `json:"avatar"`
	Desc      string   `json:"desc"`
	HomePage  string   `json:"home_page"`
	IsActive  *bool    `json:"is_active"`
	IsSuperuser bool   `json:"is_superuser"`
	Roles     []string `json:"roles"` // role IDs
}

// Create 创建用户（含密码哈希 + 角色关联）。
func (s *UserService) Create(in CreateInput) (*model.User, error) {
	// 检查用户名唯一
	if _, err := s.userRepo.FindByUsername(in.Username); err == nil {
		return nil, apperr.New(409, "用户名已存在")
	}

	// 密码策略校验（管理员创建用户同样受策略约束）
	if err := validatePasswordPolicy(s.db, in.Password); err != nil {
		return nil, err
	}

	// 密码哈希
	hashed, err := crypto.HashPassword(in.Password)
	if err != nil {
		return nil, apperr.ErrInternal
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	user := &model.User{
		Username:    in.Username,
		Password:    hashed,
		Email:       in.Email,
		Phone:       in.Phone,
		Avatar:      in.Avatar,
		Desc:        in.Desc,
		HomePage:    in.HomePage,
		IsActive:    isActive,
		IsSuperuser: in.IsSuperuser,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, apperr.ErrInternal
	}

	// 角色关联
	if len(in.Roles) > 0 {
		if err := s.userRepo.AssignRoles(user.ID, in.Roles); err != nil {
			return nil, apperr.ErrInternal
		}
	}

	// 重新加载（含角色）
	return s.userRepo.FindByID(user.ID)
}

// UpdateInput 更新用户入参（指针字段区分"未传"和"清空"）。
type UpdateInput struct {
	Email       *string  `json:"email"`
	Phone       *string  `json:"phone"`
	Avatar      *string  `json:"avatar"`
	Desc       *string  `json:"desc"`
	HomePage   *string  `json:"home_page"`
	IsActive   *bool    `json:"is_active"`
	IsSuperuser *bool   `json:"is_superuser"`
	Password    string   `json:"password"` // 非空则更新密码
	Roles       []string `json:"roles"`
}

// Update 更新用户。
func (s *UserService) Update(id string, in UpdateInput) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	if in.Email != nil {
		user.Email = *in.Email
	}
	if in.Phone != nil {
		user.Phone = *in.Phone
	}
	if in.Avatar != nil {
		user.Avatar = *in.Avatar
	}
	if in.Desc != nil {
		user.Desc = *in.Desc
	}
	if in.HomePage != nil {
		user.HomePage = *in.HomePage
	}
	if in.IsActive != nil {
		user.IsActive = *in.IsActive
	}
	if in.IsSuperuser != nil {
		user.IsSuperuser = *in.IsSuperuser
	}

	// 密码更新（重置密码同样受策略约束）
	if in.Password != "" {
		if err := validatePasswordPolicy(s.db, in.Password); err != nil {
			return nil, err
		}
		hashed, err := crypto.HashPassword(in.Password)
		if err != nil {
			return nil, apperr.ErrInternal
		}
		user.Password = hashed
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, apperr.ErrInternal
	}

	// 角色关联（仅当传入 roles 字段时更新）
	if in.Roles != nil {
		if err := s.userRepo.AssignRoles(user.ID, in.Roles); err != nil {
			return nil, apperr.ErrInternal
		}
	}

	return s.userRepo.FindByID(user.ID)
}

// superAdminRoleName 拥有全量权限的系统角色名；对其授予/撤销仅限超级管理员。
// 这是权限体系的提权闸门：否则"用户管理"权限持有者可给自己授 super_admin 完成提权。
const superAdminRoleName = "super_admin"

// CheckRoleAssignment 校验角色赋权边界（Create/Update 保存前调用）：
//   - 调用方为超级管理员：放行；
//   - 请求的角色列表包含 super_admin：拒绝；
//   - 目标用户当前持有 super_admin（任何角色变更都可能撤掉它）：拒绝。
func (s *UserService) CheckRoleAssignment(roleRefs []string, targetUserID string, callerIsSuperuser bool) error {
	if callerIsSuperuser {
		return nil
	}
	if len(roleRefs) > 0 {
		ids, _ := repository.SplitRoleRefs(roleRefs)
		q := s.db.Model(&model.Role{}).Where("name = ?", superAdminRoleName)
		if len(ids) > 0 {
			q = q.Or("id IN ? AND name = ?", ids, superAdminRoleName)
		}
		var n int64
		if err := q.Count(&n).Error; err != nil {
			return apperr.ErrInternal
		}
		if n > 0 {
			return apperr.New(403, "super_admin 角色的授予仅限超级管理员操作")
		}
	}
	if targetUserID != "" {
		var held int64
		if err := s.db.Table("user_roles").
			Joins("JOIN roles ON roles.id = user_roles.role_id").
			Where("user_roles.user_id = ? AND roles.name = ?", targetUserID, superAdminRoleName).
			Count(&held).Error; err != nil {
			return apperr.ErrInternal
		}
		if held > 0 {
			return apperr.New(403, "super_admin 用户的角色变更仅限超级管理员操作")
		}
	}
	return nil
}

// Delete 软删除用户。
func (s *UserService) Delete(id string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if err := s.userRepo.SoftDelete(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}
