package service

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// RoleService 角色业务逻辑。
type RoleService struct {
	db       *gorm.DB
	roleRepo *repository.RoleRepo
}

func NewRoleService(db *gorm.DB, roleRepo *repository.RoleRepo) *RoleService {
	return &RoleService{db: db, roleRepo: roleRepo}
}

func (s *RoleService) List(offset, limit int, f repository.RoleListFilter) ([]model.Role, int64, error) {
	return s.roleRepo.List(offset, limit, f)
}

// AccessibleMenus 查询一组角色（接受角色 ID 或名称混传）可访问的活跃菜单。
// 供用户管理页的 home_page 过滤等场景使用。
func (s *RoleService) AccessibleMenus(roleRefs []string) ([]model.Menu, error) {
	if len(roleRefs) == 0 {
		return []model.Menu{}, nil
	}
	// roles.id 是 uuid 列：把合法 UUID 与名称分开匹配，避免 PG 对 uuid 列的类型错误
	ids, names := make([]string, 0, len(roleRefs)), make([]string, 0, len(roleRefs))
	for _, r := range roleRefs {
		if r == "" {
			continue
		}
		if _, err := uuid.Parse(r); err == nil {
			ids = append(ids, r)
		} else {
			names = append(names, r)
		}
	}
	if len(ids) == 0 && len(names) == 0 {
		return []model.Menu{}, nil
	}

	q := s.db.Distinct("menus.*").
		Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Joins("JOIN roles ON roles.id = role_menus.role_id").
		Where("menus.is_active = ?", true).
		Where("roles.is_active = ?", true)
	switch {
	case len(ids) > 0 && len(names) > 0:
		q = q.Where("roles.id IN ? OR roles.name IN ?", ids, names)
	case len(ids) > 0:
		q = q.Where("roles.id IN ?", ids)
	default:
		q = q.Where("roles.name IN ?", names)
	}

	var menus []model.Menu
	if err := q.Order("menus.sort_order").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
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
	Name        string  `json:"name" binding:"required,max=128"`
	DisplayName string  `json:"display_name" binding:"max=128"`
	Desc        string  `json:"desc"`
	IsActive    *bool   `json:"is_active"`
	MenuIDs     []int64 `json:"menu_ids"`
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
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Desc:        in.Desc,
		IsActive:    isActive,
	}
	// 建角色与绑菜单包在同一事务内：AssignMenus 失败时整体回滚。
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.roleRepo.WithTx(tx)
		if err := txRepo.Create(role); err != nil {
			if isDupKey(err) {
				return apperr.New(409, "角色名已存在")
			}
			return apperr.ErrInternal
		}
		if len(in.MenuIDs) > 0 {
			if err := txRepo.AssignMenus(role.ID, in.MenuIDs); err != nil {
				if errors.Is(err, repository.ErrInvalidMenuIDs) {
					return apperr.New(400, "菜单不存在或已删除，请刷新后重新选择")
				}
				return apperr.ErrInternal
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.roleRepo.FindByID(role.ID)
}

// UpdateInput 更新角色入参。Desc 用指针区分「未传」与「传空串」。
type RoleUpdateInput struct {
	Name        string  `json:"name"`
	DisplayName *string `json:"display_name"`
	Desc        *string `json:"desc"`
	IsActive    *bool   `json:"is_active"`
	MenuIDs     []int64 `json:"menu_ids"`
}

func (s *RoleService) Update(id string, in RoleUpdateInput) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	// 系统角色不允许改名，也不允许停用：停用 super_admin 会让所有超管当场失去全部权限
	if role.IsSystem {
		if in.Name != "" && in.Name != role.Name {
			return nil, apperr.New(403, "系统内置角色不允许修改名称")
		}
		if in.IsActive != nil && !*in.IsActive {
			return nil, apperr.New(403, "系统内置角色不允许停用")
		}
	}

	if in.Name != "" {
		role.Name = in.Name
	}
	if in.Desc != nil {
		role.Desc = *in.Desc
	}
	// 展示名可随时改，包括系统内置角色：它不参与任何权限判定
	if in.DisplayName != nil {
		role.DisplayName = *in.DisplayName
	}
	if in.IsActive != nil {
		role.IsActive = *in.IsActive
	}

	var updated *model.Role
	// 角色资料、授权关联与结果回读共用事务，任一步失败都不留下部分修改。
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.roleRepo.WithTx(tx)
		if err := txRepo.Update(role); err != nil {
			if isDupKey(err) {
				return apperr.New(409, "角色名已存在")
			}
			return apperr.ErrInternal
		}
		if in.MenuIDs != nil {
			if err := txRepo.AssignMenus(role.ID, in.MenuIDs); err != nil {
				if errors.Is(err, repository.ErrInvalidMenuIDs) {
					return apperr.New(400, "菜单不存在或已删除，请刷新后重新选择")
				}
				return apperr.ErrInternal
			}
		}
		var err error
		updated, err = txRepo.FindByID(role.ID)
		if err != nil {
			return apperr.ErrInternal
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return updated, nil
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
