// Package main 是 init-data 命令：初始化超级用户、默认角色、默认菜单。
//
// 用法:
//   go run ./cmd/init-data --username admin --password 'YourStr0ng!Pass'
//
// --password 未提供时自动生成随机密码并打印一次（不再提供 admin123 等默认弱口令）。
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gorm.io/gorm"

	"adminx/internal/config"
	"adminx/internal/database"
	"adminx/internal/model"
	"adminx/pkg/crypto"
	"gorm.io/datatypes"
)

func main() {
	username := flag.String("username", "admin", "超级管理员用户名")
	password := flag.String("password", "", "超级管理员密码（不提供则随机生成并打印一次）")
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 未提供密码时生成随机密码（打印一次；不再内置 admin123 等弱口令）
	if *password == "" {
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			fmt.Fprintf(os.Stderr, "生成随机密码失败: %v\n", err)
			os.Exit(1)
		}
		pw := "Ax" + base64.RawURLEncoding.EncodeToString(b) + "!9"
		*password = pw
		fmt.Printf("已生成随机超级管理员密码（仅显示这一次，请立即保存）:\n  %s\n\n", pw)
	}

	// 密码策略校验，弱口令直接拒绝初始化
	if err := validatePasswordStrength(*password); err != nil {
		fmt.Fprintf(os.Stderr, "密码不满足安全要求: %v\n", err)
		os.Exit(1)
	}

	db, err := database.Init(cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "数据库初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 1. 创建超级管理员
	if err := createSuperuser(db, *username, *password); err != nil {
		fmt.Fprintf(os.Stderr, "创建超级用户失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 超级用户已创建: %s\n", *username)

	// 2. 创建默认角色
	if err := createDefaultRoles(db); err != nil {
		fmt.Fprintf(os.Stderr, "创建默认角色失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 默认角色已创建: super_admin, security_admin, audit_admin, user")

	// 3. 创建默认菜单
	if err := createDefaultMenus(db); err != nil {
		fmt.Fprintf(os.Stderr, "创建默认菜单失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 默认菜单已创建")

	// 4. 内置角色绑定默认菜单（幂等）
	if err := bindDefaultRoleMenus(db); err != nil {
		fmt.Fprintf(os.Stderr, "绑定角色菜单失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 内置角色菜单已绑定")

	// 5. 默认密码策略（单例行，幂等）
	if err := createDefaultPasswordPolicy(db); err != nil {
		fmt.Fprintf(os.Stderr, "创建默认密码策略失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 默认密码策略已就绪")

	fmt.Println("\n初始化完成。请使用配置的凭证登录。")
}

func createSuperuser(db *gorm.DB, username, password string) error {
	var count int64
	db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil // 已存在，跳过
	}

	hashed, err := crypto.HashPassword(password)
	if err != nil {
		return err
	}

	user := &model.User{
		Username:    username,
		Password:    hashed,
		IsActive:    true,
		IsSuperuser: true,
		IsStaff:     true,
		Desc:        "超级管理员",
	}
	return db.Create(user).Error
}

func createDefaultRoles(db *gorm.DB) error {
	roles := []model.Role{
		{Name: "super_admin", Desc: "超级管理员（全部权限）", IsActive: true, IsSystem: true},
		{Name: "security_admin", Desc: "安全管理员（配置/监控/集群/审计）", IsActive: true, IsSystem: true},
		{Name: "audit_admin", Desc: "审计管理员（审计/登录日志只读）", IsActive: true, IsSystem: true},
		{Name: "user", Desc: "普通用户（基础访问）", IsActive: true, IsSystem: true},
	}
	for _, r := range roles {
		var count int64
		db.Model(&model.Role{}).Where("name = ?", r.Name).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&r).Error; err != nil {
			return err
		}
	}
	return nil
}

func createDefaultMenus(db *gorm.DB) error {
	seeds := defaultMenuSeeds()

	// 改名：历史库里的旧 Go 种子 code → 现行 code（保留菜单 ID 与角色绑定，避免重复菜单）
	renamed, err := migrateLegacyMenuCodes(db, seeds)
	if err != nil {
		return err
	}
	if renamed > 0 {
		fmt.Printf("  （旧菜单 code 已迁移: %d 条）\n", renamed)
	}

	// 第一遍：按 code 建立/更新菜单（骨架 code 以种子为准，见 ensureMenu）
	byCode := make(map[string]*model.Menu, len(seeds))
	for _, s := range seeds {
		m, err := ensureMenu(db, s)
		if err != nil {
			return err
		}
		byCode[s.Code] = m
	}

	// 第二遍：按 ParentCode 回填 parent_id / depth（根=1，子=父+1）。
	// 幂等：已存在的扁平菜单（parent_id 为空、depth=1）会被修正为正确层级。
	seedByCode := make(map[string]menuSeed, len(seeds))
	for _, s := range seeds {
		seedByCode[s.Code] = s
	}
	for _, s := range seeds {
		menu := byCode[s.Code]
		depth, err := seedDepth(seedByCode, s)
		if err != nil {
			return err
		}
		var parentID *int64
		if s.ParentCode != "" {
			parent, ok := byCode[s.ParentCode]
			if !ok {
				return fmt.Errorf("菜单种子 %s 的父菜单 %s 不在种子列表中", s.Code, s.ParentCode)
			}
			id := parent.ID
			parentID = &id
		}
		if menu.ParentID != nil && parentID != nil && *menu.ParentID == *parentID && menu.Depth == depth {
			continue
		}
		if menu.ParentID == nil && parentID == nil && menu.Depth == depth {
			continue
		}
		// Select 强制写入零值/nil（否则 GORM 会跳过空值，无法把子菜单改回根）
		hierarchy := menuHierarchy{ParentID: parentID, Depth: depth}
		if err := db.Model(&model.Menu{}).Where("id = ?", menu.ID).
			Select("parent_id", "depth").Updates(hierarchy).Error; err != nil {
			return err
		}
	}

	return nil
}

// legacyMenuCodeRenames 旧版 Go 种子的菜单 code → 现行 code。
// 现行 code 与 Django 版 init_data 对齐（如 system:user），改名可保留菜单 ID 与已有角色绑定。
var legacyMenuCodeRenames = map[string]string{
	"system-user":     "system:user",
	"system-role":     "system:role",
	"system-menu":     "system:permission",
	"system-config":   "system:config",
	"audit-login-log": "audit:login-log",
}

// migrateLegacyMenuCodes 把历史库里旧 code 的种子菜单改名到现行 code，返回改名条数。
// 现行 code 已存在（或旧 code 本身还在骨架里）时跳过，保证幂等。
func migrateLegacyMenuCodes(db *gorm.DB, seeds []menuSeed) (int, error) {
	seeded := make(map[string]bool, len(seeds))
	for _, s := range seeds {
		seeded[s.Code] = true
	}
	renamed := 0
	for oldCode, newCode := range legacyMenuCodeRenames {
		if seeded[oldCode] {
			continue
		}
		var count int64
		if err := db.Model(&model.Menu{}).Where("code = ?", newCode).Count(&count).Error; err != nil {
			return renamed, err
		}
		if count > 0 {
			continue
		}
		res := db.Model(&model.Menu{}).Where("code = ?", oldCode).Update("code", newCode)
		if res.Error != nil {
			return renamed, res.Error
		}
		renamed += int(res.RowsAffected)
	}
	return renamed, nil
}

// bindDefaultRoleMenus 为内置角色绑定默认菜单（三权分立开箱可用，菜单码对齐 Django 版 init_data）。
func bindDefaultRoleMenus(db *gorm.DB) error {
	seeds := defaultMenuSeeds()
	allCodes := make([]string, 0, len(seeds))
	for _, s := range seeds {
		allCodes = append(allCodes, s.Code)
	}
	bindings := []struct {
		roleName  string
		menuCodes []string
	}{
		{"super_admin", allCodes},
		{"security_admin", []string{"system:permission", "system:config", "system:resource", "system:cluster", "audit", "audit:log", "audit:login-log"}},
		{"audit_admin", []string{"audit", "audit:log", "audit:login-log"}},
	}
	for _, b := range bindings {
		var role model.Role
		if err := db.Where("name = ?", b.roleName).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		var menus []model.Menu
		if err := db.Where("code IN ?", b.menuCodes).Find(&menus).Error; err != nil {
			return err
		}
		if len(menus) > 0 {
			if err := db.Model(&role).Association("Menus").Append(menus); err != nil {
				return err
			}
		}
	}
	return nil
}

// menuSeed 菜单种子。ParentCode 指向同批种子里的父菜单 code（父先于子写入）。
// 用 code 而非 ID 引用父级：种子可以重复执行、且不依赖自增 ID 具体取值。
type menuSeed struct {
	Code           string
	Name           string
	Icon           string
	Path           string
	Component      string
	PermissionCode string
	MenuType       string
	SortOrder      int
	ParentCode     string
	AllowedPaths   []string // RBAC glob 规则列表，空表示不配置
}

// defaultMenuSeeds 内置菜单骨架。
//
// 对齐 Django 版 init_data 的 DEFAULT_MENUS（Go 版之前只移植了 8 条，丢了系统资源/组件管理/
// 节点管理/定时任务/通知中心/操作审计/权限管理），并按前端 router/index.ts 的实际路由校准：
//   - code 沿用 Django 命名（`system:user` 这种带冒号的层级码）
//   - permission_code 必须等于前端 route 的 meta.permission，否则非超管能看菜单却进不去页面
//   - allowed_paths 按 Go 后端真实路由（/api/v1/...，不带 base_path）书写
func defaultMenuSeeds() []menuSeed {
	return []menuSeed{
		{Code: "dashboard", Name: "仪表盘", Icon: "DashboardOutlined", Path: "/dashboard", Component: "dashboard/DashboardView", SortOrder: 0, AllowedPaths: []string{"GET:/api/v1/common/dashboard/*"}},

		{Code: "system", Name: "系统管理", Icon: "SettingOutlined", Path: "/system", PermissionCode: "system:view", SortOrder: 1},
		{Code: "system:user", Name: "用户管理", Icon: "UserOutlined", Path: "/system/users", Component: "system/UserList", PermissionCode: "accounts:user:list", SortOrder: 1, ParentCode: "system", AllowedPaths: []string{"/api/v1/accounts/users/*"}},
		{Code: "system:role", Name: "角色管理", Icon: "TeamOutlined", Path: "/system/roles", Component: "system/RoleList", PermissionCode: "accounts:role:list", SortOrder: 2, ParentCode: "system", AllowedPaths: []string{"/api/v1/accounts/roles/*"}},
		{Code: "system:permission", Name: "权限管理", Icon: "SafetyOutlined", Path: "/system/permissions", Component: "system/PermissionList", PermissionCode: "accounts:permission:list", SortOrder: 3, ParentCode: "system", AllowedPaths: []string{"/api/v1/menu/*"}},
		{Code: "system:config", Name: "配置中心", Icon: "ControlOutlined", Path: "/system/config", Component: "config/ConfigCenter", PermissionCode: "config_center:config:list", SortOrder: 4, ParentCode: "system", AllowedPaths: []string{"/api/v1/config/*", "/api/v1/common/site-info/*"}},
		{Code: "system:resource", Name: "系统资源", Icon: "MonitorOutlined", Path: "/system/resources", Component: "monitor/SystemMonitor", PermissionCode: "monitor:resource:list", SortOrder: 5, ParentCode: "system", AllowedPaths: []string{"/api/v1/monitor/*"}},
		{Code: "system:component", Name: "组件管理", Icon: "AppstoreOutlined", Path: "/system/components", Component: "monitor/ComponentStatus", PermissionCode: "monitor:resource:list", SortOrder: 6, ParentCode: "system", AllowedPaths: []string{"/api/v1/cluster/components/*", "/api/v1/files/*", "/api/v1/common/health/*", "/api/v1/common/components/*"}},
		{Code: "system:cluster", Name: "节点管理", Icon: "ClusterOutlined", Path: "/system/nodes", Component: "cluster/ClusterNodes", PermissionCode: "cluster:node:list", SortOrder: 7, ParentCode: "system", AllowedPaths: []string{"/api/v1/cluster/*"}},
		{Code: "system:scheduler", Name: "定时任务", Icon: "ClockCircleOutlined", Path: "/system/scheduler", Component: "scheduler/ScheduleJobList", PermissionCode: "webservice:schedulejob:list", SortOrder: 8, ParentCode: "system", AllowedPaths: []string{"/api/v1/jobs/*"}},
		{Code: "system:notification", Name: "通知中心", Icon: "BellOutlined", Path: "/system/notification", Component: "notification/NotificationCenter", PermissionCode: "notification:notification:list", SortOrder: 9, ParentCode: "system", AllowedPaths: []string{"/api/v1/notification/*"}},

		{Code: "audit", Name: "安全审计", Icon: "AuditOutlined", Path: "/audit", PermissionCode: "audit:view", SortOrder: 2},
		{Code: "audit:log", Name: "操作审计", Icon: "FileSearchOutlined", Path: "/audit/log", Component: "audit/AuditLogList", PermissionCode: "audit:auditlog:list", SortOrder: 1, ParentCode: "audit", AllowedPaths: []string{"GET:/api/v1/audit/*"}},
		{Code: "audit:login-log", Name: "登录日志", Icon: "LoginOutlined", Path: "/audit/login-log", Component: "audit/LoginLogList", PermissionCode: "accounts:userloginlog:list", SortOrder: 2, ParentCode: "audit", AllowedPaths: []string{"GET:/api/v1/accounts/login-logs/*"}},
	}
}

// ensureMenu 按 code 建立或更新菜单骨架。
//
// 对骨架里的 code，种子是权威来源（对齐 Django 版 update_or_create 语义）：覆盖名称/路径/
// 组件/权限码/图标/排序/RBAC 规则，因此重跑 init-data 能修复历史库里过时的菜单定义；
// 管理员自建的菜单（code 不在骨架里）完全不受影响。
func ensureMenu(db *gorm.DB, s menuSeed) (*model.Menu, error) {
	var m model.Menu
	err := db.Where("code = ?", s.Code).First(&m).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	menuType := s.MenuType
	if menuType == "" {
		menuType = "menu"
	}

	m.Code = s.Code
	m.Name = s.Name
	m.Icon = s.Icon
	m.Path = s.Path
	m.Component = s.Component
	m.PermissionCode = s.PermissionCode
	m.MenuType = menuType
	m.SortOrder = s.SortOrder
	m.AllowedPaths = allowedPathsJSON(s.AllowedPaths)
	m.IsActive = true
	m.IsVisible = true

	if err == gorm.ErrRecordNotFound {
		m.Depth = 1 // 层级统一在 createDefaultMenus 第二遍回填
		if err := db.Create(&m).Error; err != nil {
			return nil, err
		}
		return &m, nil
	}

	// Save 写入全部字段（含空值）：种子把 permission_code / allowed_paths 置空时也要生效
	if err := db.Save(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// seedDepth 沿 ParentCode 回溯计算种子菜单深度（根=1）。
func seedDepth(seedByCode map[string]menuSeed, s menuSeed) (int, error) {
	depth := 1
	seen := map[string]bool{s.Code: true}
	for pc := s.ParentCode; pc != ""; pc = seedByCode[pc].ParentCode {
		parent, ok := seedByCode[pc]
		if !ok {
			return 0, fmt.Errorf("菜单种子 %s 的父菜单 %s 不在种子列表中", s.Code, pc)
		}
		if seen[pc] {
			return 0, fmt.Errorf("菜单种子 %s 的父级链存在环", s.Code)
		}
		seen[pc] = true
		depth++
		pc = parent.ParentCode
	}
	return depth, nil
}

// menuHierarchy 菜单层级字段（用于 Select 强制写入，含 nil parent_id）。
type menuHierarchy struct {
	ParentID *int64
	Depth    int
}

// createDefaultPasswordPolicy 建立密码策略单例行（ID=1，幂等）。
// 不建行时读取策略会走 GORM 的 record not found 分支，接口虽然仍返回 200，
// 但每次前端加载都会打一行错误级日志（看着像报错）。
func createDefaultPasswordPolicy(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.PasswordPolicy{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	policy := model.DefaultPasswordPolicy()
	return db.Create(&policy).Error
}

// allowedPathsJSON 把 glob 规则列表转为 allowed_paths JSON（空列表 = 空数组）。
func allowedPathsJSON(patterns []string) datatypes.JSON {
	if len(patterns) == 0 {
		return datatypes.JSON([]byte(`[]`))
	}
	quoted := make([]string, 0, len(patterns))
	for _, p := range patterns {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, p))
	}
	return datatypes.JSON([]byte("[" + strings.Join(quoted, ",") + "]"))
}

// validatePasswordStrength 超管密码最低强度要求（长度≥10 且含大小写/数字/特殊字符）。
func validatePasswordStrength(password string) error {
	if subtle.ConstantTimeCompare([]byte(password), []byte("admin123")) == 1 {
		return fmt.Errorf("禁止使用默认弱口令 admin123")
	}
	if len(password) < 10 {
		return fmt.Errorf("长度至少 10 位")
	}
	p := model.PasswordPolicy{MinLength: 10, RequireUpper: true, RequireLower: true, RequireDigit: true, RequireSpecial: true, IsActive: true}
	return p.Validate(password)
}
