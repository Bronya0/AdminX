package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"adminx/internal/database"
	"adminx/internal/model"
)

// setupTestDB 初始化 SQLite :memory: 数据库。
// 生产模型使用 PostgreSQL 特有类型（jsonb, gen_random_uuid()），测试库改用
// internal/database 的 SQLite DDL 建表（与运行时 sqlite 模式同一份定义，避免 schema 漂移）。
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	// SQLite :memory: 每条连接独立，必须限制连接池为 1
	// 否则 GORM 多连接 → 多个独立内存库 → "no such table"
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}
	if err != nil {
		t.Fatalf("连接 SQLite 失败: %v", err)
	}

	for _, stmt := range database.SQLiteStatements() {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表失败: %v\nSQL: %s", err, stmt)
		}
	}

	return db
}

// fastHash 快速哈希密码（bcrypt cost=4 加速测试）。
func fastHash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		panic("fastHash failed: " + err.Error())
	}
	return string(hash)
}

// seedRole 创建测试角色。
func seedRole(t *testing.T, db *gorm.DB, name string, isSystem bool) *model.Role {
	role := &model.Role{
		Name:     name,
		IsActive: true,
		IsSystem: isSystem,
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	return role
}

// seedMenu 创建测试菜单。
func seedMenu(t *testing.T, db *gorm.DB, code, name, path string) *model.Menu {
	menu := &model.Menu{
		Code:     code,
		Name:     name,
		Path:     path,
		MenuType: "menu",
		IsActive: true,
		Depth:    1,
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("创建菜单失败: %v", err)
	}
	return menu
}

// seedUser 创建测试用户（含角色关联）。
func seedUser(t *testing.T, db *gorm.DB, username, password string, roleIDs ...string) *model.User {
	hashed := fastHash(password)
	user := &model.User{
		Username:  username,
		Password:  hashed,
		Email:     username + "@test.com",
		IsActive:  true,
		IsStaff:   true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if len(roleIDs) > 0 {
		var roles []model.Role
		db.Where("id IN ?", roleIDs).Find(&roles)
		_ = db.Model(user).Association("Roles").Replace(roles)
	}
	return user
}

// boolPtr 测试辅助：bool 指针。
func boolPtr(b bool) *bool { return &b }
