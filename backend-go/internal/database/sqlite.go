package database

import (
	"fmt"

	"gorm.io/gorm"
)

// SQLite 本地开发模式的建表语句（显式 DDL）。
//
// 为什么不用 AutoMigrate：生产模型带 PostgreSQL 专有类型与默认值
// （type:uuid + default:gen_random_uuid()、jsonb + default:'[]'::jsonb），
// SQLite 无法执行这些 DDL。因此 SQLite 用下面这份显式 DDL 建表。
//
// 该 DDL 与 postgres 的 AutoMigrate 结果必须保持一致：
//   - 新增模型 / 字段时同步更新这里，否则 SQLite 模式运行时报 "no such column"；
//   - sqlite_test.go 的守卫测试会校验覆盖完整性（含 M2M 中间表）。
//
// 测试代码（internal/service）也复用这份 DDL，避免两处 schema 漂移。
func SQLiteStatements() []string {
	return []string{
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
			is_active INTEGER NOT NULL DEFAULT 1,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS password_histories (
			id TEXT PRIMARY KEY,
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
		`CREATE TABLE IF NOT EXISTS file_records (
			id TEXT PRIMARY KEY,
			original_name TEXT NOT NULL,
			size INTEGER NOT NULL,
			mime_type TEXT NOT NULL DEFAULT '',
			storage_backend TEXT NOT NULL DEFAULT 'local',
			storage_path TEXT NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			uploaded_by TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS scheduler_heartbeats (
			id TEXT PRIMARY KEY,
			last_heartbeat DATETIME,
			reload_pending INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS monitor_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cpu_percent REAL NOT NULL DEFAULT 0,
			memory_percent REAL NOT NULL DEFAULT 0,
			disk_percent REAL NOT NULL DEFAULT 0,
			disk_read_m_bps REAL NOT NULL DEFAULT 0,
			disk_write_m_bps REAL NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
}

// migrateSQLite 执行 SQLite 建表语句（CREATE TABLE IF NOT EXISTS，幂等）。
func migrateSQLite(db *gorm.DB) error {
	for _, stmt := range SQLiteStatements() {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("执行 SQLite 建表语句失败: %w", err)
		}
	}
	return nil
}
