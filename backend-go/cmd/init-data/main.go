// Package main 是 init-data 命令：初始化超级用户、默认角色、默认菜单。
//
// 用法:
//   go run ./cmd/init-data --username admin --password admin123
//
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"gorm.io/gorm"

	"adminx/internal/config"
	"adminx/internal/database"
	"adminx/internal/model"
	"adminx/pkg/crypto"
	"gorm.io/datatypes"
)

func main() {
	username := flag.String("username", "admin", "超级管理员用户名")
	password := flag.String("password", "admin123", "超级管理员密码")
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
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
	// 基础菜单骨架（完整菜单可在 UI 中配置）
	menus := []model.Menu{
		{Code: "dashboard", Name: "仪表盘", Icon: "DashboardOutlined", Path: "/dashboard", Component: "dashboard/DashboardView", MenuType: "menu", SortOrder: 1, AllowedPaths: mustJSONPath("/api/v1/common/dashboard/*")},
		{Code: "system", Name: "系统管理", Icon: "SettingOutlined", Path: "/system", MenuType: "menu", SortOrder: 2},
		{Code: "system-user", Name: "用户管理", Path: "/system/users", Component: "system/UserList", PermissionCode: "user", MenuType: "menu", SortOrder: 1, AllowedPaths: mustJSONPath("/api/v1/accounts/users/*")},
		{Code: "system-role", Name: "角色管理", Path: "/system/roles", Component: "system/RoleList", PermissionCode: "role", MenuType: "menu", SortOrder: 2, AllowedPaths: mustJSONPath("/api/v1/accounts/roles/*")},
		{Code: "system-menu", Name: "菜单管理", Path: "/system/menus", Component: "system/MenuList", PermissionCode: "menu", MenuType: "menu", SortOrder: 3, AllowedPaths: mustJSONPath("/api/v1/menu/*")},
	}
	for _, m := range menus {
		var count int64
		db.Model(&model.Menu{}).Where("code = ?", m.Code).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func mustJSONPath(pathPattern string) datatypes.JSON {
	return datatypes.JSON([]byte(fmt.Sprintf(`["%s"]`, pathPattern)))
}
