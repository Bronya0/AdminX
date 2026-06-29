package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"djangoadminx/internal/jwt"
	"djangoadminx/internal/middleware"
	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

// ── 跨实体联动: 用户 ↔ 角色 ↔ 菜单 → 权限 ──

// TestRBAC_AssignRoleToUser_ThenUserHasRole
// 给用户分配角色后，用户列表应包含该角色。
func TestRBAC_AssignRoleToUser_ThenUserHasRole(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "管理员", false)
	user := seedUser(t, db, "zhangsan", "pass123")

	// 分配角色
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// 验证
	got, _ := userRepo.FindByID(user.ID)
	if len(got.Roles) != 1 {
		t.Fatalf("角色数 = %d, want 1", len(got.Roles))
	}
	if got.Roles[0].Name != "管理员" {
		t.Errorf("角色名 = %s, want 管理员", got.Roles[0].Name)
	}
}

// TestRBAC_AssignMenuToRole_ThenRoleHasMenu
// 给角色分配菜单后，角色应包含该菜单。
func TestRBAC_AssignMenuToRole_ThenRoleHasMenu(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)

	role := seedRole(t, db, "运营", false)
	menu := seedMenu(t, db, "user_mgmt", "用户管理", "/users")

	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	got, _ := roleRepo.FindByID(role.ID)
	if len(got.Menus) != 1 {
		t.Fatalf("菜单数 = %d, want 1", len(got.Menus))
	}
	if got.Menus[0].Code != "user_mgmt" {
		t.Errorf("菜单code = %s, want user_mgmt", got.Menus[0].Code)
	}
}

// TestRBAC_AssignMultipleRoles_UserHasUnionOfMenus
// 用户有两个角色，各有一个菜单，用户应能访问两个菜单的路径。
func TestRBAC_AssignMultipleRoles_UserHasUnionOfMenus(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	// 创建两个角色
	roleA := seedRole(t, db, "用户管理角色", false)
	roleB := seedRole(t, db, "配置管理角色", false)

	// 创建两个菜单，各有 allowed_paths
	menuUsers := seedMenu(t, db, "users", "用户管理", "/api/users")
	menuUsers.AllowedPaths = mustJSONBytes(`["GET:/api/users/*"]`)
	db.Save(menuUsers)

	menuConfig := seedMenu(t, db, "config", "配置中心", "/api/config")
	menuConfig.AllowedPaths = mustJSONBytes(`["GET:/api/config/*"]`)
	db.Save(menuConfig)

	// 分配菜单给角色
	roleRepo.AssignMenus(roleA.ID, []int64{menuUsers.ID})
	roleRepo.AssignMenus(roleB.ID, []int64{menuConfig.ID})

	// 创建用户并赋予两个角色
	user := seedUser(t, db, "lisi", "pass123")
	userRepo.AssignRoles(user.ID, []string{roleA.ID, roleB.ID})

	// 验证: 用户应能访问 /api/users/xxx
	ok := checkPermissionWithDB(db, user.ID, "GET", "/api/users/list")
	if !ok {
		t.Error("用户应有 /api/users/* 的访问权限")
	}

	// 验证: 用户应能访问 /api/config/xxx
	ok = checkPermissionWithDB(db, user.ID, "GET", "/api/config/keys")
	if !ok {
		t.Error("用户应有 /api/config/* 的访问权限")
	}

	// 验证: 用户不应访问其他路径
	ok = checkPermissionWithDB(db, user.ID, "GET", "/api/secret")
	if ok {
		t.Error("用户不应有 /api/secret 的访问权限")
	}
}

// TestRBAC_RemoveRole_UserLosesPermission
// 移除用户角色后，用户失去对应菜单的访问权限。
func TestRBAC_RemoveRole_UserLosesPermission(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "临时角色", false)
	menu := seedMenu(t, db, "temp", "临时", "/api/temp")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/temp/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "wangwu", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// 先确认有权限
	if !checkPermissionWithDB(db, user.ID, "GET", "/api/temp/data") {
		t.Fatal("分配角色后应有权限")
	}

	// 移除角色（给空列表）
	userRepo.AssignRoles(user.ID, []string{})

	// 确认失去权限
	if checkPermissionWithDB(db, user.ID, "GET", "/api/temp/data") {
		t.Error("移除角色后应失去权限")
	}
}

// TestRBAC_DeactivateMenu_UserLosesPath
// 菜单被禁用后，关联用户应失去对应路径权限。
func TestRBAC_DeactivateMenu_UserLosesPath(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "测试角色", false)
	menu := seedMenu(t, db, "active_menu", "活跃菜单", "/api/active")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/active/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "zhaoliu", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// 先确认有权限
	if !checkPermissionWithDB(db, user.ID, "GET", "/api/active/page") {
		t.Fatal("菜单激活时应有权限")
	}

	// 禁用菜单
	db.Model(&model.Menu{}).Where("id = ?", menu.ID).Update("is_active", false)

	// 确认失去权限
	if checkPermissionWithDB(db, user.ID, "GET", "/api/active/page") {
		t.Error("菜单禁用后应失去权限")
	}
}

// TestRBAC_RemoveMenuFromRole_UserLosesPermission
// 从角色中移除菜单关联后，用户失去对应权限。
// 注: 当前 middleware checkPermission 不校验 roles.is_active，仅校验 menus.is_active。
func TestRBAC_RemoveMenuFromRole_UserLosesPermission(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "动态角色", false)
	menu := seedMenu(t, db, "remove_menu", "可移除菜单", "/api/removable")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/removable/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "sunqi", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	if !checkPermissionWithDB(db, user.ID, "GET", "/api/removable/data") {
		t.Fatal("分配菜单后应有权限")
	}

	// 从角色移除该菜单
	roleRepo.AssignMenus(role.ID, []int64{})

	if checkPermissionWithDB(db, user.ID, "GET", "/api/removable/data") {
		t.Error("移除菜单关联后用户应失去权限")
	}
}

// TestRBAC_DeactivateRole_UserLosesPermission
// 角色被禁用后，拥有该角色的用户失去对应权限（checks roles.is_active）。
func TestRBAC_DeactivateRole_UserLosesPermission(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "活跃角色", false)
	menu := seedMenu(t, db, "enable_menu", "启用菜单", "/api/enabled")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/enabled/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "sunqi", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	if !checkPermissionWithDB(db, user.ID, "GET", "/api/enabled/data") {
		t.Fatal("角色活跃时应有权限")
	}

	// 禁用角色
	db.Model(&model.Role{}).Where("id = ?", role.ID).Update("is_active", false)

	if checkPermissionWithDB(db, user.ID, "GET", "/api/enabled/data") {
		t.Error("角色禁用后用户应失去权限")
	}
}

// TestRBAC_MethodSpecificPermission
// allowed_paths 中的方法前缀应起效：GET 可访问，POST 不能。
func TestRBAC_MethodSpecificPermission(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "只读角色", false)
	menu := seedMenu(t, db, "readonly", "只读菜单", "/api/readonly")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/readonly/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "readonly_user", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// GET 应通过
	if !checkPermissionWithDB(db, user.ID, "GET", "/api/readonly/list") {
		t.Error("GET 应有权限")
	}
	// POST 应被拒
	if checkPermissionWithDB(db, user.ID, "POST", "/api/readonly/create") {
		t.Error("POST 不应有权限（只授权 GET）")
	}
	// DELETE 应被拒
	if checkPermissionWithDB(db, user.ID, "DELETE", "/api/readonly/1") {
		t.Error("DELETE 不应有权限（只授权 GET）")
	}
}

// TestRBAC_MultipleMethodsSamePath
// 同一路径可配置多个方法规则。
func TestRBAC_MultipleMethodsSamePath(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "读写角色", false)
	menu := seedMenu(t, db, "crud", "CRUD菜单", "/api/items")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/items/*", "POST:/api/items", "PUT:/api/items/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "crud_user", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// 三条规则均应生效
	for _, tc := range []struct{ method, path string; want bool }{
		{"GET", "/api/items/1", true},
		{"POST", "/api/items", true},
		{"PUT", "/api/items/1", true},
		{"DELETE", "/api/items/1", false},
	} {
		got := checkPermissionWithDB(db, user.ID, tc.method, tc.path)
		if got != tc.want {
			t.Errorf("%s %s: got %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

// TestRBAC_SuperuserBypassAll
// 超级管理员自动放行所有路径，不受菜单限制。
func TestRBAC_SuperuserBypassAll(t *testing.T) {
	db := setupTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)

	// 创建超级用户（无任何角色/菜单）
	hashed := fastHash("admin123")
	user := &model.User{
		Username:    "admin",
		Password:    hashed,
		IsActive:    true,
		IsSuperuser: true,
		IsStaff:     true,
	}
	db.Create(user)

	gin.SetMode(gin.TestMode)
	// 用 middleware RBAC 完整链路验证
	token, _ := jwtMgr.GenerateAccessToken(user.ID, user.Username)

	for _, path := range []string{"/api/users", "/api/config", "/api/secret", "/api/admin/whatever"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", path, nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// 先跑 JWT 中间件注入 user context
		middleware.JWTAuth(jwtMgr, db)(c)
		if c.IsAborted() {
			t.Fatalf("JWT 认证未通过: %s", path)
		}
		// 再跑 RBAC
		middleware.RBAC(db)(c)
		if c.IsAborted() {
			t.Errorf("超级管理员应放行 %s，但被拒绝", path)
		}
	}
}

// TestRBAC_NoRole_UserHasNoPermission
// 没有分配任何角色的用户，不能访问任何受保护路径。
func TestRBAC_NoRole_UserHasNoPermission(t *testing.T) {
	db := setupTestDB(t)
	jwtMgr := jwt.New("secret", "issuer", 30*time.Minute, 7*24*time.Hour, nil)
	// 创建一个菜单但不分配给任何人
	menu := seedMenu(t, db, "secret_menu", "秘密菜单", "/api/secret")
	menu.AllowedPaths = mustJSONBytes(`["/api/secret/*"]`)
	db.Save(menu)

	user := seedUser(t, db, "norole_user", "pass123")

	gin.SetMode(gin.TestMode)
	token, _ := jwtMgr.GenerateAccessToken(user.ID, user.Username)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/secret/data", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	middleware.JWTAuth(jwtMgr, db)(c)
	if c.IsAborted() {
		t.Fatal("JWT 认证未通过")
	}
	middleware.RBAC(db)(c)
	if !c.IsAborted() {
		t.Error("无角色用户应被拒绝")
	}
}

// TestRBAC_EmptyAllowedPaths_MenuAllowsNothing
// 菜单的 allowed_paths 为空数组时，不授权任何路径。
func TestRBAC_EmptyAllowedPaths_MenuAllowsNothing(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "空路径角色", false)
	menu := seedMenu(t, db, "empty_paths", "空路径菜单", "/api/empty")
	// 故意不设 allowed_paths（默认空数组）
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "empty_user", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	if checkPermissionWithDB(db, user.ID, "GET", "/api/empty/data") {
		t.Error("空 allowed_paths 不应授权任何路径")
	}
}

// TestRBAC_FullFlow_CreateAssignVerify
// 全流程: 创建→分配→验证→修改→再验证。
func TestRBAC_FullFlow_CreateAssignVerify(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)
	userSvc := NewUserService(db, userRepo)
	roleSvc := NewRoleService(db, roleRepo)

	// Step 1: 创建两个角色
	rViewer, _ := roleSvc.Create(RoleCreateInput{Name: "查看者"})
	rEditor, _ := roleSvc.Create(RoleCreateInput{Name: "编辑者"})

	// Step 2: 创建菜单并分配
	mUser := seedMenu(t, db, "m_user", "用户页", "/api/users")
	mUser.AllowedPaths = mustJSONBytes(`["GET:/api/users/*"]`)
	db.Save(mUser)
	roleRepo.AssignMenus(rViewer.ID, []int64{mUser.ID})

	mPost := seedMenu(t, db, "m_post", "文章页", "/api/posts")
	mPost.AllowedPaths = mustJSONBytes(`["GET:/api/posts/*", "POST:/api/posts", "PUT:/api/posts/*", "DELETE:/api/posts/*"]`)
	db.Save(mPost)
	roleRepo.AssignMenus(rEditor.ID, []int64{mPost.ID})

	// Step 3: 创建用户，先只给 viewer 角色
	user, _ := userSvc.Create(CreateInput{
		Username: "integration_user",
		Password: "test123",
		Roles:    []string{rViewer.ID},
	})

	// Step 4: 验证 viewer 权限
	assertHasAccess(t, db, user.ID, "GET", "/api/users/list")
	assertNoAccess(t, db, user.ID, "GET", "/api/posts/list")  // viewer 不能看文章
	assertNoAccess(t, db, user.ID, "POST", "/api/posts")       // viewer 不能发文

	// Step 5: 追加 editor 角色
	userRepo.AssignRoles(user.ID, []string{rViewer.ID, rEditor.ID})

	// Step 6: 验证 viewer + editor 权限并集
	assertHasAccess(t, db, user.ID, "GET", "/api/users/list")
	assertHasAccess(t, db, user.ID, "GET", "/api/posts/list")
	assertHasAccess(t, db, user.ID, "POST", "/api/posts")
	assertHasAccess(t, db, user.ID, "DELETE", "/api/posts/1")

	// Step 7: 收回 editor 角色
	userRepo.AssignRoles(user.ID, []string{rViewer.ID})

	// Step 8: 验证退回到只读
	assertHasAccess(t, db, user.ID, "GET", "/api/users/list")
	assertNoAccess(t, db, user.ID, "GET", "/api/posts/list")

	// Step 9: 禁用 viewer 角色
	db.Model(&model.Role{}).Where("id = ?", rViewer.ID).Update("is_active", false)

	// Step 10: 用户完全失去所有权限（roles.is_active=false 被校验）
	assertNoAccess(t, db, user.ID, "GET", "/api/users/list")
	assertNoAccess(t, db, user.ID, "GET", "/api/posts/list")
}

// TestRBAC_RoleNameChange_DoesNotAffectPermissions
// 角色改名不影响已关联菜单和用户。
func TestRBAC_RoleNameChange_DoesNotAffectPermissions(t *testing.T) {
	db := setupTestDB(t)
	roleRepo := repository.NewRoleRepo(db)
	userRepo := repository.NewUserRepo(db)

	role := seedRole(t, db, "旧名称", false)
	menu := seedMenu(t, db, "stable", "稳定菜单", "/api/stable")
	menu.AllowedPaths = mustJSONBytes(`["GET:/api/stable/*"]`)
	db.Save(menu)
	roleRepo.AssignMenus(role.ID, []int64{menu.ID})

	user := seedUser(t, db, "stable_user", "pass123")
	userRepo.AssignRoles(user.ID, []string{role.ID})

	// 改名
	role.Name = "新名称"
	db.Save(role)

	// 验证权限不受影响
	if !checkPermissionWithDB(db, user.ID, "GET", "/api/stable/data") {
		t.Error("角色改名后权限不应受影响")
	}
}

// ── 辅助函数 ──

func assertHasAccess(t *testing.T, db any, userID, method, path string) {
	t.Helper()
	if !checkPermissionWithDB(db, userID, method, path) {
		t.Errorf("%s %s: 应有权限但被拒绝", method, path)
	}
}

func assertNoAccess(t *testing.T, db any, userID, method, path string) {
	t.Helper()
	if checkPermissionWithDB(db, userID, method, path) {
		t.Errorf("%s %s: 不应有权限但被放行", method, path)
	}
}

// checkPermissionWithDB 直接调 middleware 的 checkPermission 验证权限。
// db 参数类型为 any 以绕过 middleware 包对 *gorm.DB 的导入限制。
func checkPermissionWithDB(db any, userID, method, requestPath string) bool {
	ok, err := middleware.CheckPermissionForTest(db, userID, method, requestPath)
	if err != nil {
		return false
	}
	return ok
}

// mustJSONBytes 将 JSON 字符串转为 datatypes.JSON 格式。
func mustJSONBytes(s string) []byte {
	return []byte(s)
}
