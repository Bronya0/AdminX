package service

import (
	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

// RoleService 角色业务逻辑。
type RoleService struct {
	db       *gorm.DB
	roleRepo *repository.RoleRepo
}

func NewRoleService(db *gorm.DB, roleRepo *repository.RoleRepo) *RoleService {
	return &RoleService{db: db, roleRepo: roleRepo}
}

func (s *RoleService) List(offset, limit int, search string) ([]model.Role, int64, error) {
	return s.roleRepo.List(offset, limit, search)
}

func (s *RoleService) All() ([]model.Role, error) {
	return s.roleRepo.All()
}

func (s *RoleService) GetByID(id string) (*model.Role, error) {
	r, err := s.roleRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return r, nil
}

// CreateInput 创建角色入参。
type RoleCreateInput struct {
	Name     string  `json:"name" binding:"required,max=128"`
	Desc     string  `json:"desc"`
	IsActive *bool   `json:"is_active"`
	MenuIDs  []int64 `json:"menu_ids"`
}

func (s *RoleService) Create(in RoleCreateInput) (*model.Role, error) {
	// 名称唯一检查
	if _, err := s.roleRepo.FindByName(in.Name); err == nil {
		return nil, apperr.New(409, "角色名已存在")
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	role := &model.Role{
		Name:     in.Name,
		Desc:     in.Desc,
		IsActive: isActive,
	}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, apperr.ErrInternal
	}

	if len(in.MenuIDs) > 0 {
		if err := s.roleRepo.AssignMenus(role.ID, in.MenuIDs); err != nil {
			return nil, apperr.ErrInternal
		}
	}
	return s.roleRepo.FindByID(role.ID)
}

// UpdateInput 更新角色入参。
type RoleUpdateInput struct {
	Name     string  `json:"name"`
	Desc     string  `json:"desc"`
	IsActive *bool   `json:"is_active"`
	MenuIDs  []int64 `json:"menu_ids"`
}

func (s *RoleService) Update(id string, in RoleUpdateInput) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	// 系统角色不允许改名
	if role.IsSystem && in.Name != "" && in.Name != role.Name {
		return nil, apperr.New(403, "系统内置角色不允许修改名称")
	}

	if in.Name != "" {
		role.Name = in.Name
	}
	role.Desc = in.Desc
	if in.IsActive != nil {
		role.IsActive = *in.IsActive
	}

	if err := s.roleRepo.Update(role); err != nil {
		return nil, apperr.ErrInternal
	}

	if in.MenuIDs != nil {
		if err := s.roleRepo.AssignMenus(role.ID, in.MenuIDs); err != nil {
			return nil, apperr.ErrInternal
		}
	}
	return s.roleRepo.FindByID(role.ID)
}

func (s *RoleService) Delete(id string) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if err := s.roleRepo.Delete(id); err != nil {
		if err == repository.ErrSystemRoleNotDeletable {
			return apperr.New(403, "系统内置角色不允许删除")
		}
		return apperr.ErrInternal
	}
	return nil
}
