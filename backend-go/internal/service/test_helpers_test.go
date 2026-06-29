package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"adminx/internal/model"
)

// setupTestDB 初始化 SQLite :memory: 数据库。
// 由于生产模型使用 PostgreSQL 特有类型（jsonb, gen_random_uuid()），
// 测试中用 raw SQL 创建 SQLite 兼容的表。
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

	// 用 raw SQL 创建 SQLite 兼容的表
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			"desc" TEXT NOT NULL DEFAULT '',
			home_page TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			is_staff INTEGER NOT NULL DEFAULT 0,
			is_superuser INTEGER NOT NULL DEFAULT 0,
			last_login DATETIME,
			last_logout DATETIME,
			last_activity DATETIME,
			date_joined DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			"desc" TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 1,
			is_system INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			PRIMARY KEY (user_id, role_id)
		)`,
		`CREATE TABLE IF NOT EXISTS menus (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			icon TEXT NOT NULL DEFAULT '',
			path TEXT NOT NULL DEFAULT '',
			component TEXT NOT NULL DEFAULT '',
			permission_code TEXT NOT NULL DEFAULT '',
			menu_type TEXT NOT NULL DEFAULT 'menu',
			is_active INTEGER NOT NULL DEFAULT 1,
			is_visible INTEGER NOT NULL DEFAULT 1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			parent_id INTEGER,
			allowed_paths TEXT NOT NULL DEFAULT '[]',
			depth INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS role_menus (
			role_id TEXT NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (role_id, menu_id)
		)`,
		`CREATE TABLE IF NOT EXISTS login_locks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			failed_count INTEGER NOT NULL DEFAULT 0,
			locked_at DATETIME,
			unlocked_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_login_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			username TEXT NOT NULL,
			ip TEXT DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			success INTEGER NOT NULL DEFAULT 1,
			message TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			value TEXT NOT NULL DEFAULT '',
			value_type TEXT NOT NULL DEFAULT 'string',
			encrypted_value TEXT NOT NULL DEFAULT '',
			is_encrypted INTEGER NOT NULL DEFAULT 0,
			"desc" TEXT NOT NULL DEFAULT '',
			"group" TEXT NOT NULL DEFAULT 'default',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS schedule_jobs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			command_type TEXT NOT NULL DEFAULT 'python',
			handler TEXT NOT NULL DEFAULT '',
			command TEXT NOT NULL DEFAULT '',
			trigger_type TEXT NOT NULL DEFAULT 'interval',
			trigger_config TEXT NOT NULL DEFAULT '',
			args TEXT NOT NULL DEFAULT '',
			kwargs TEXT NOT NULL DEFAULT '{}',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS job_logs (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'running',
			result TEXT NOT NULL DEFAULT '',
			started_at DATETIME,
			finished_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			model_name TEXT NOT NULL,
			object_id TEXT NOT NULL DEFAULT '',
			object_repr TEXT NOT NULL DEFAULT '',
			operator TEXT NOT NULL DEFAULT '',
			operator_ip TEXT DEFAULT '',
			old_values TEXT DEFAULT '{}',
			new_values TEXT DEFAULT '{}',
			diff_summary TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			notification_type TEXT NOT NULL DEFAULT 'info',
			is_read INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_configs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			secret TEXT NOT NULL DEFAULT '',
			events TEXT NOT NULL DEFAULT 'info,success,warning,error',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS webhook_logs (
			id TEXT PRIMARY KEY,
			webhook_id TEXT NOT NULL,
			notification_id TEXT,
			status TEXT NOT NULL DEFAULT 'success',
			response_status INTEGER,
			response_body TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS password_policies (
			id INTEGER PRIMARY KEY,
			min_length INTEGER NOT NULL DEFAULT 8,
			require_upper INTEGER NOT NULL DEFAULT 1,
			require_lower INTEGER NOT NULL DEFAULT 1,
			require_digit INTEGER NOT NULL DEFAULT 1,
			require_special INTEGER NOT NULL DEFAULT 1,
			expire_days INTEGER NOT NULL DEFAULT 90,
			history_count INTEGER NOT NULL DEFAULT 5,
			is_active INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS password_histories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cluster_nodes (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL DEFAULT 8000,
			role TEXT NOT NULL DEFAULT 'slave',
			status TEXT NOT NULL DEFAULT 'offline',
			version TEXT NOT NULL DEFAULT '',
			last_heartbeat DATETIME,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS service_components (
			id TEXT PRIMARY KEY,
			app_label TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '',
			host TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			last_heartbeat DATETIME,
			upgrade_version TEXT NOT NULL DEFAULT '',
			upgrade_url TEXT NOT NULL DEFAULT '',
			upgrade_checksum TEXT NOT NULL DEFAULT '',
			uninstall_pending INTEGER NOT NULL DEFAULT 0,
			extra_info TEXT NOT NULL DEFAULT '{}',
			registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS scheduler_heartbeats (
			id TEXT PRIMARY KEY,
			last_heartbeat DATETIME,
			reload_pending INTEGER NOT NULL DEFAULT 0
		)`,
	}

	for _, stmt := range statements {
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
