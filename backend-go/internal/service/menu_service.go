package service

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/rbacrule"
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

// buildMenuTree 平铺菜单按 ParentID 组装为嵌套树。
// 孤儿节点（ParentID 非空但父不存在）提升为顶级，保证不丢。
func buildMenuTree(all []model.Menu) []model.Menu {
	exists := make(map[int64]bool, len(all))
	for _, m := range all {
		exists[m.ID] = true
	}
	childrenOf := make(map[int64][]model.Menu, len(all))
	roots := make([]model.Menu, 0)
	for _, m := range all {
		if m.ParentID == nil {
			roots = append(roots, m)
			continue
		}
		if !exists[*m.ParentID] {
			// 父不存在：当 root，保证不丢
			roots = append(roots, m)
			continue
		}
		childrenOf[*m.ParentID] = append(childrenOf[*m.ParentID], m)
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

// MenusByUser 查询用户在侧边栏可见的菜单。
// 超级管理员与 RBAC 同一条“全放行”约定：不依赖角色绑定，直接返回全部启用菜单
// （否则无角色的超管会出现“接口全能调、菜单却是空”的割裂）。
func (s *MenuService) MenusByUser(userID string, isSuperuser bool) ([]model.Menu, error) {
	if isSuperuser {
		return s.menuRepo.AllActive()
	}
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
	if err := validateAllowedPaths(in.AllowedPaths); err != nil {
		return nil, err
	}
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
	}

	// 深度由父级派生（根=1，子=父+1）
	depth, err := s.depthForParent(in.ParentID, 0)
	if err != nil {
		return nil, err
	}
	menu.Depth = depth

	if err := s.menuRepo.Create(menu); err != nil {
		return nil, apperr.ErrInternal
	}
	return menu, nil
}

// RegisterOrUpdate 按 code 幂等注册菜单（业务容器对接用）：存在则更新，不存在则创建。
// 支持 ParentID 变更（变更后重算自身 depth + 递归重算子孙 depth）。
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
			if err := validateAllowedPaths(in.AllowedPaths); err != nil {
				return nil, err
			}
			existing.AllowedPaths = mustJSON(in.AllowedPaths)
		}
		if in.ParentID != nil && !sameParent(existing.ParentID, in.ParentID) {
			depth, err := s.depthForParent(in.ParentID, existing.ID)
			if err != nil {
				return nil, err
			}
			existing.ParentID = in.ParentID
			existing.Depth = depth
			if err := s.menuRepo.Update(existing); err != nil {
				return nil, apperr.ErrInternal
			}
			if err := s.recalcDescendantDepths(existing.ID, depth); err != nil {
				return nil, err
			}
			return existing, nil
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
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Icon           *string `json:"icon"`
	Path           *string `json:"path"`
	Component      *string `json:"component"`
	PermissionCode *string `json:"permission_code"`
	MenuType       string  `json:"menu_type"`
	IsActive       *bool   `json:"is_active"`
	IsVisible      *bool   `json:"is_visible"`
	SortOrder      *int    `json:"sort_order"`
	// ParentID nil = 不改动父子关系；传父菜单 ID 时按其深度重算 depth。
	ParentID     *int64   `json:"parent_id"`
	AllowedPaths []string `json:"allowed_paths"`
}

func (s *MenuService) Update(id int64, in MenuUpdateInput) (*model.Menu, error) {
	if err := validateAllowedPaths(in.AllowedPaths); err != nil {
		return nil, err
	}
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
	if in.SortOrder != nil {
		menu.SortOrder = *in.SortOrder
	}
	// 传了 parent_id 才调整层级，并按父级重算 depth（nil = 保持原父子关系）
	parentChanged := false
	if in.ParentID != nil {
		if !sameParent(menu.ParentID, in.ParentID) {
			parentChanged = true
		}
		depth, err := s.depthForParent(in.ParentID, menu.ID)
		if err != nil {
			return nil, err
		}
		menu.ParentID = in.ParentID
		menu.Depth = depth
	}
	if in.AllowedPaths != nil {
		menu.AllowedPaths = mustJSON(in.AllowedPaths)
	}

	if err := s.menuRepo.Update(menu); err != nil {
		return nil, apperr.ErrInternal
	}
	// 换父后递归重算子孙 depth：否则子孙 depth 还停在旧层级，前端多级菜单变平铺。
	if parentChanged {
		if err := s.recalcDescendantDepths(menu.ID, menu.Depth); err != nil {
			return nil, err
		}
	}
	return menu, nil
}

func (s *MenuService) Delete(id int64) error {
	if err := s.menuRepo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrMenuHasChildren) {
			return apperr.New(400, "该菜单下有子菜单，请先删除子菜单")
		}
		return apperr.ErrInternal
	}
	return nil
}

// validateAllowedPaths 校验接口路径白名单（规则语义与解析见 internal/rbacrule）。
//
// 必须在写入前拦住：非法规则不会让接口报错，只是运行时永不匹配，
// 表现为「菜单明明配了权限、用户仍然 403」，只能人肉比对规则字符串才能发现。
func validateAllowedPaths(paths []string) error {
	for _, p := range paths {
		reason := rbacrule.InvalidReason(p)
		if reason == "" {
			continue
		}
		return apperr.New(400, "接口路径白名单格式错误："+reason+"（"+strings.TrimSpace(p)+"）")
	}
	return nil
}

// sameParent 比较两个 ParentID 是否指向同一父级（都为 nil 或值相等）。
func sameParent(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// recalcDescendantDepths 换父后递归重算全部子孙 depth（BFS 按层下推 parent.Depth+1）。
func (s *MenuService) recalcDescendantDepths(rootID int64, rootDepth int) error {
	menus, err := s.menuRepo.All()
	if err != nil {
		return apperr.ErrInternal
	}
	childrenOf := make(map[int64][]int64, len(menus))
	depthOf := make(map[int64]int, len(menus))
	for _, m := range menus {
		depthOf[m.ID] = m.Depth
		if m.ParentID != nil {
			childrenOf[*m.ParentID] = append(childrenOf[*m.ParentID], m.ID)
		}
	}
	depthOf[rootID] = rootDepth
	queue := []int64{rootID}
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		for _, child := range childrenOf[parent] {
			want := depthOf[parent] + 1
			if depthOf[child] != want {
				if err := s.db.Model(&model.Menu{}).Where("id = ?", child).
					Update("depth", want).Error; err != nil {
					return apperr.ErrInternal
				}
				depthOf[child] = want
			}
			queue = append(queue, child)
		}
	}
	return nil
}

// depthForParent 由父级派生菜单深度：根节点为 1，子节点为父深度 +1。
// selfID 非 0（更新场景）时拒绝把菜单挂到自己或自己的后代下，避免形成环。
func (s *MenuService) depthForParent(parentID *int64, selfID int64) (int, error) {
	if parentID == nil {
		return 1, nil
	}
	if *parentID <= 0 {
		return 0, apperr.New(400, "无效的父菜单 ID")
	}
	parent, err := s.menuRepo.FindByID(*parentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, apperr.New(404, "父菜单不存在")
		}
		return 0, apperr.ErrInternal
	}
	if selfID != 0 {
		cyclic, err := s.wouldCycle(selfID, parent.ID)
		if err != nil {
			return 0, apperr.ErrInternal
		}
		if cyclic {
			return 0, apperr.New(400, "不能把菜单挂到它自己或它的子菜单下")
		}
	}
	return parent.Depth + 1, nil
}

// wouldCycle 判断把 menuID 挂到 newParentID 下是否会成环：
// 沿 parent_id 向上回溯 newParentID 的祖先链，遇到 menuID 即成环（seen 兼顾已存在的环）。
func (s *MenuService) wouldCycle(menuID, newParentID int64) (bool, error) {
	menus, err := s.menuRepo.All()
	if err != nil {
		return false, err
	}
	parentOf := make(map[int64]*int64, len(menus))
	for i := range menus {
		parentOf[menus[i].ID] = menus[i].ParentID
	}
	seen := make(map[int64]bool, len(menus))
	for id := newParentID; !seen[id]; {
		if id == menuID {
			return true, nil
		}
		seen[id] = true
		parent := parentOf[id]
		if parent == nil {
			return false, nil
		}
		id = *parent
	}
	return false, nil
}

func (s *MenuService) GetByCode(code string) (*model.Menu, error) {
	menu, err := s.menuRepo.FindByCode(code)
	if err != nil {
		return nil, err
	}
	return menu, nil
}
