package service

import (
	"testing"

	"adminx/internal/model"
	"adminx/internal/repository"
)

func TestMenuService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, err := svc.Create(MenuCreateInput{
		Code:      "dashboard",
		Name:      "仪表盘",
		Path:      "/dashboard",
		Component: "DashboardView",
		MenuType:  "menu",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if menu.Name != "仪表盘" {
		t.Errorf("Name = %s", menu.Name)
	}
	if menu.ID == 0 {
		t.Error("ID 为 0")
	}
}

func TestMenuService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, _ := svc.Create(MenuCreateInput{
		Code: "old",
		Name: "旧名称",
		Path: "/old",
	})

	updated, err := svc.Update(menu.ID, MenuUpdateInput{
		Name: "新名称",
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("Name = %s", updated.Name)
	}
}

func TestMenuService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, _ := svc.Create(MenuCreateInput{
		Code: "todelete",
		Name: "待删除",
		Path: "/todelete",
	})

	if err := svc.Delete(menu.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	_, err := svc.GetByID(menu.ID)
	if err == nil {
		t.Fatal("删除后应查不到菜单")
	}
}

func TestMenuService_All(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	seedMenu(t, db, "m1", "菜单1", "/m1")
	seedMenu(t, db, "m2", "菜单2", "/m2")

	menus, err := svc.All()
	if err != nil {
		t.Fatalf("All 失败: %v", err)
	}
	if len(menus) != 2 {
		t.Errorf("len = %d, want 2", len(menus))
	}
}

// TestMenuService_MenusByUser 侧边栏菜单：超管“全放行”，普通用户仅看角色绑定的菜单。
func TestMenuService_MenusByUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	dashboard := seedMenu(t, db, "dashboard", "仪表盘", "/dashboard")
	disabled := seedMenu(t, db, "disabled-menu", "停用菜单", "/disabled")
	if err := db.Model(disabled).Update("is_active", false).Error; err != nil {
		t.Fatalf("停用菜单失败: %v", err)
	}

	// 超级管理员：无任何角色绑定，也应看到全部启用菜单（种子超管正是这个状态）
	super := seedUser(t, db, "root", "Root@12345")
	if err := db.Model(super).Update("is_superuser", true).Error; err != nil {
		t.Fatalf("设置超级管理员失败: %v", err)
	}
	menus, err := svc.MenusByUser(super.ID, true)
	if err != nil {
		t.Fatalf("MenusByUser(超管) 失败: %v", err)
	}
	if codes := menuCodes(menus); len(codes) != 1 || codes[0] != "dashboard" {
		t.Errorf("超管应只看到启用菜单 [dashboard]，实际: %v", codes)
	}

	// 普通用户：无角色 → 无菜单
	plain := seedUser(t, db, "bob", "Bob@12345")
	if menus, err = svc.MenusByUser(plain.ID, false); err != nil {
		t.Fatalf("MenusByUser(无角色用户) 失败: %v", err)
	}
	if codes := menuCodes(menus); len(codes) != 0 {
		t.Errorf("无角色用户不应有菜单，实际: %v", codes)
	}

	// 普通用户：绑定角色 → 看到该角色的启用菜单
	role := seedRole(t, db, "viewer", false)
	if err := db.Model(role).Association("Menus").Append(dashboard); err != nil {
		t.Fatalf("绑定角色菜单失败: %v", err)
	}
	viewer := seedUser(t, db, "carol", "Carol@12345", role.ID)
	if menus, err = svc.MenusByUser(viewer.ID, false); err != nil {
		t.Fatalf("MenusByUser(普通用户) 失败: %v", err)
	}
	if codes := menuCodes(menus); len(codes) != 1 || codes[0] != "dashboard" {
		t.Errorf("绑定角色后应看到 [dashboard]，实际: %v", codes)
	}
}

func menuCodes(menus []model.Menu) []string {
	codes := make([]string, 0, len(menus))
	for _, m := range menus {
		codes = append(codes, m.Code)
	}
	return codes
}

// TestMenuService_ChildDepth 新建子菜单时 depth 由父级派生（不信任前端传值）。
func TestMenuService_ChildDepth(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	parent, err := svc.Create(MenuCreateInput{Code: "system", Name: "系统管理", Path: "/system"})
	if err != nil {
		t.Fatalf("Create(父) 失败: %v", err)
	}
	if parent.Depth != 1 {
		t.Errorf("根菜单 depth = %d, want 1", parent.Depth)
	}

	child, err := svc.Create(MenuCreateInput{Code: "system-user", Name: "用户管理", Path: "/system/users", ParentID: &parent.ID})
	if err != nil {
		t.Fatalf("Create(子) 失败: %v", err)
	}
	if child.Depth != 2 {
		t.Errorf("子菜单 depth = %d, want 2", child.Depth)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("子菜单 parent_id = %v, want %d", child.ParentID, parent.ID)
	}

	// 父菜单不存在 → 404，不落脏数据
	missing := int64(99999)
	if _, err := svc.Create(MenuCreateInput{Code: "orphan", Name: "孤儿", ParentID: &missing}); err == nil {
		t.Error("父菜单不存在时应报错")
	}
}

// TestMenuService_UpdateHierarchy 移动菜单时重算 depth，并拒绝成环。
func TestMenuService_UpdateHierarchy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	root, _ := svc.Create(MenuCreateInput{Code: "audit", Name: "安全审计", Path: "/audit"})
	child, _ := svc.Create(MenuCreateInput{Code: "audit-login-log", Name: "登录日志", Path: "/audit/login-log", ParentID: &root.ID})
	grandchild, _ := svc.Create(MenuCreateInput{Code: "audit-detail", Name: "审计详情", Path: "/audit/login-log/detail", ParentID: &child.ID})

	// nil = 不改动父子关系（改名不应把子菜单拉平）
	renamed, err := svc.Update(child.ID, MenuUpdateInput{Name: "登录日志2"})
	if err != nil {
		t.Fatalf("Update(改名) 失败: %v", err)
	}
	if renamed.Depth != 2 || renamed.ParentID == nil || *renamed.ParentID != root.ID {
		t.Errorf("改名后层级被改动: depth=%d parent_id=%v", renamed.Depth, renamed.ParentID)
	}

	// 改挂到根下 → depth 重算为 2
	moved, err := svc.Update(grandchild.ID, MenuUpdateInput{ParentID: &root.ID})
	if err != nil {
		t.Fatalf("Update(改挂父级) 失败: %v", err)
	}
	if moved.Depth != 2 {
		t.Errorf("改挂后 depth = %d, want 2", moved.Depth)
	}

	// 挂到自己或自己的后代下 → 400
	if _, err := svc.Update(root.ID, MenuUpdateInput{ParentID: &root.ID}); err == nil {
		t.Error("挂到自己下应报错")
	}
	if _, err := svc.Update(root.ID, MenuUpdateInput{ParentID: &child.ID}); err == nil {
		t.Error("挂到自己的后代下应报错")
	}
}
