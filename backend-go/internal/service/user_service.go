package service

import (
	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"
	"djangoadminx/pkg/crypto"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
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

// UpdateInput 更新用户入参（密码可选）。
type UpdateInput struct {
	Email       string   `json:"email"`
	Phone       string   `json:"phone"`
	Avatar      string   `json:"avatar"`
	Desc        string   `json:"desc"`
	HomePage    string   `json:"home_page"`
	IsActive    *bool    `json:"is_active"`
	IsSuperuser *bool    `json:"is_superuser"`
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

	user.Email = in.Email
	user.Phone = in.Phone
	user.Avatar = in.Avatar
	user.Desc = in.Desc
	user.HomePage = in.HomePage
	if in.IsActive != nil {
		user.IsActive = *in.IsActive
	}
	if in.IsSuperuser != nil {
		user.IsSuperuser = *in.IsSuperuser
	}

	// 密码更新
	if in.Password != "" {
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
