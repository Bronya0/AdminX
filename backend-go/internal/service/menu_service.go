package service

import (
	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// MenuService 菜单业务逻辑。
type MenuService struct {
	db       *gorm.DB
	menuRepo *repository.MenuRepo
}

func NewMenuService(db *gorm.DB, menuRepo *repository.MenuRepo) *MenuService {
	return &MenuService{db: db, menuRepo: menuRepo}
}

// All 查询所有菜单（扁平列表，前端按 path 前缀构建树）。
// Tree 返回嵌套菜单树（按 ParentID 组装，供角色授权树等管理界面使用）。
func (s *MenuService) Tree() ([]model.Menu, error) {
	menus, err := s.All()
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus), nil
}

// buildMenuTree 平铺菜单按 ParentID 组装为嵌套树（孤儿节点视为顶级）。
func buildMenuTree(all []model.Menu) []model.Menu {
	childrenOf := make(map[int64][]model.Menu, len(all))
	roots := make([]model.Menu, 0)
	for _, m := range all {
		if m.ParentID == nil {
			roots = append(roots, m)
		} else {
			childrenOf[*m.ParentID] = append(childrenOf[*m.ParentID], m)
		}
	}
	var attach func(items []model.Menu) []model.Menu
	attach = func(items []model.Menu) []model.Menu {
		out := make([]model.Menu, 0, len(items))
		for _, m := range items {
			if kids := attach(childrenOf[m.ID]); len(kids) > 0 {
				m.Children = kids
			}
			out = append(out, m)
		}
		return out
	}
	return attach(roots)
}

func (s *MenuService) All() ([]model.Menu, error) {
	return s.menuRepo.All()
}

// MenusByUser 查询用户关联的菜单（RBAC 用）。
func (s *MenuService) MenusByUser(userID string) ([]model.Menu, error) {
	return s.menuRepo.MenusByUser(userID)
}

// GetByID 查询单个菜单。
func (s *MenuService) GetByID(id int64) (*model.Menu, error) {
	m, err := s.menuRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return m, nil
}

// CreateInput 创建菜单入参。
type MenuCreateInput struct {
	Code           string   `json:"code" binding:"required,max=128"`
	Name           string   `json:"name" binding:"required,max=128"`
	Icon           string   `json:"icon"`
	Path           string   `json:"path"`
	Component      string   `json:"component"`
	PermissionCode string   `json:"permission_code"`
	MenuType       string   `json:"menu_type"`
	IsActive       *bool    `json:"is_active"`
	IsVisible      *bool    `json:"is_visible"`
	SortOrder      int      `json:"sort_order"`
	ParentID       *int64   `json:"parent_id"`
	AllowedPaths   []string `json:"allowed_paths"`
}

func (s *MenuService) Create(in MenuCreateInput) (*model.Menu, error) {
	menuType := in.MenuType
	if menuType == "" {
		menuType = "menu"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	isVisible := true
	if in.IsVisible != nil {
		isVisible = *in.IsVisible
	}
	allowedPaths := mustJSON(in.AllowedPaths)

	menu := &model.Menu{
		Code:           in.Code,
		Name:           in.Name,
		Icon:           in.Icon,
		Path:           in.Path,
		Component:      in.Component,
		PermissionCode: in.PermissionCode,
		MenuType:       menuType,
		IsActive:       isActive,
		IsVisible:      isVisible,
		SortOrder:      in.SortOrder,
		ParentID:       in.ParentID,
		AllowedPaths:   allowedPaths,
		Depth:          1, // 默认根节点，后续可调整
	}

	if err := s.menuRepo.Create(menu); err != nil {
		return nil, apperr.ErrInternal
	}
	return menu, nil
}

// RegisterOrUpdate 按 code 幂等注册菜单（业务容器对接用）：存在则更新，不存在则创建。
func (s *MenuService) RegisterOrUpdate(in MenuCreateInput) (*model.Menu, error) {
	existing, err := s.menuRepo.FindByCode(in.Code)
	if err == nil && existing != nil {
		// 已存在：更新业务可写字段
		if in.Name != "" {
			existing.Name = in.Name
		}
		if in.Icon != "" {
			existing.Icon = in.Icon
		}
		if in.Path != "" {
			existing.Path = in.Path
		}
		if in.Component != "" {
			existing.Component = in.Component
		}
		if in.PermissionCode != "" {
			existing.PermissionCode = in.PermissionCode
		}
		if in.MenuType != "" {
			existing.MenuType = in.MenuType
		}
		if in.IsActive != nil {
			existing.IsActive = *in.IsActive
		}
		if in.IsVisible != nil {
			existing.IsVisible = *in.IsVisible
		}
		if in.SortOrder != 0 {
			existing.SortOrder = in.SortOrder
		}
		if in.AllowedPaths != nil {
			existing.AllowedPaths = mustJSON(in.AllowedPaths)
		}
		if err := s.menuRepo.Update(existing); err != nil {
			return nil, apperr.ErrInternal
		}
		return existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, apperr.ErrInternal
	}
	// 不存在：走常规创建
	return s.Create(in)
}

// UpdateInput 更新菜单入参（指针字段区分"未传"和"清空"）。
type MenuUpdateInput struct {
	Code           string   `json:"code"`
	Name           string   `json:"name"`
	Icon           *string  `json:"icon"`
	Path           *string  `json:"path"`
	Component      *string  `json:"component"`
	PermissionCode *string  `json:"permission_code"`
	MenuType       string   `json:"menu_type"`
	IsActive       *bool    `json:"is_active"`
	IsVisible      *bool    `json:"is_visible"`
	SortOrder      int      `json:"sort_order"`
	ParentID       *int64   `json:"parent_id"`
	AllowedPaths   []string `json:"allowed_paths"`
}

func (s *MenuService) Update(id int64, in MenuUpdateInput) (*model.Menu, error) {
	menu, err := s.menuRepo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	if in.Code != "" {
		menu.Code = in.Code
	}
	if in.Name != "" {
		menu.Name = in.Name
	}
	if in.Icon != nil {
		menu.Icon = *in.Icon
	}
	if in.Path != nil {
		menu.Path = *in.Path
	}
	if in.Component != nil {
		menu.Component = *in.Component
	}
	if in.PermissionCode != nil {
		menu.PermissionCode = *in.PermissionCode
	}
	if in.MenuType != "" {
		menu.MenuType = in.MenuType
	}
	if in.IsActive != nil {
		menu.IsActive = *in.IsActive
	}
	if in.IsVisible != nil {
		menu.IsVisible = *in.IsVisible
	}
	menu.SortOrder = in.SortOrder
	menu.ParentID = in.ParentID
	if in.AllowedPaths != nil {
		menu.AllowedPaths = mustJSON(in.AllowedPaths)
	}

	if err := s.menuRepo.Update(menu); err != nil {
		return nil, apperr.ErrInternal
	}
	return menu, nil
}

func (s *MenuService) Delete(id int64) error {
	if err := s.menuRepo.Delete(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}
