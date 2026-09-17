// Package database 负责 GORM 初始化、连接池配置、AutoMigrate。
package database

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"adminx/internal/config"
	// 注册模型包以便 AutoMigrate 能发现所有表
	"adminx/internal/model"
)

// sqliteMaxConns SQLite 连接数上限（单写入者：只允许一条连接，避免写锁冲突）。
const sqliteMaxConns = 1

// Init 初始化 GORM 连接并返回 *gorm.DB。
func Init(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	db, err := open(cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}

	// SQLite 是单写入者模型：连接池固定为 1，避免并发写报 "database is locked"
	maxOpen, maxIdle := cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns
	if cfg.Database.Driver == config.DriverSQLite {
		maxOpen, maxIdle = sqliteMaxConns, sqliteMaxConns
	} else {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}

	logger.Info("数据库连接成功",
		"driver", cfg.Database.Driver,
		"max_open", maxOpen,
		"max_idle", maxIdle,
	)

	if err := migrate(cfg.Database.Driver, db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	logger.Info("数据库迁移完成", "driver", cfg.Database.Driver)

	return db, nil
}

// open 按 database.driver 选择 GORM 方言（driver 取值已由 config.validate 校验）。
func open(cfg *config.Config) (*gorm.DB, error) {
	opts := &gorm.Config{Logger: gormlogger.Default.LogMode(parseLogLevel(cfg.Database.LogLevel))}

	var dialector gorm.Dialector
	if cfg.Database.Driver == config.DriverSQLite {
		// 对齐 postgres 的错误语义（唯一索引冲突 → gorm.ErrDuplicatedKey）
		opts.TranslateError = true
		dialector = sqlite.Open(cfg.Database.DSN)
	} else {
		dialector = postgres.Open(cfg.Database.DSN)
	}

	db, err := gorm.Open(dialector, opts)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return db, nil
}

// migrate 建表：postgres 走 AutoMigrate，sqlite 走显式 DDL（见 sqlite.go）。
func migrate(driver string, db *gorm.DB) error {
	if driver == config.DriverSQLite {
		return migrateSQLite(db)
	}
	return autoMigrate(db)
}

// Close 关闭数据库连接。
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func parseLogLevel(level string) gormlogger.LogLevel {
	switch level {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn":
		return gormlogger.Warn
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}

// Models 返回需要建表的全部模型（AutoMigrate 与 SQLite DDL 守卫测试共用，新增模型只改这里）。
func Models() []interface{} {
	return []interface{}{
		// accounts
		&model.User{},
		&model.Role{},
		&model.LoginLock{},
		&model.UserLoginLog{},
		// menu
		&model.Menu{},
		// config_center
		&model.Config{},
		// audit
		&model.AuditLog{},
		// jobs
		&model.ScheduleJob{},
		&model.JobLog{},
		// notification
		&model.Notification{},
		&model.WebhookConfig{},
		&model.WebhookLog{},
		// cluster
		&model.ClusterNode{},
		&model.ServiceComponent{},
		// file_center
		&model.FileRecord{},
		// policy
		&model.PasswordPolicy{},
		&model.PasswordHistory{},
		// monitor
		&model.MonitorSample{},
	}
}

// autoMigrate 自动建表（GORM 只增列、加索引，不删列）。
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
