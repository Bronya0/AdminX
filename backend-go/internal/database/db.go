// Package database 负责 GORM 初始化、连接池配置、AutoMigrate。
package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"djangoadminx/internal/config"
	// 注册模型包以便 AutoMigrate 能发现所有表
	"djangoadminx/internal/model"
)

// Init 初始化 GORM 连接并返回 *gorm.DB。
func Init(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	logLevel := parseLogLevel(cfg.Database.LogLevel)

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 ping 失败: %w", err)
	}

	logger.Info("数据库连接成功",
		"max_open", cfg.Database.MaxOpenConns,
		"max_idle", cfg.Database.MaxIdleConns,
	)

	// AutoMigrate（生产环境也可用，GORM 的 migrate 是幂等的，只增不删）
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	logger.Info("数据库迁移完成")

	return db, nil
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

// autoMigrate 自动建表（GORM 只增列、加索引，不删列）。
func autoMigrate(db *gorm.DB) error {
	models := []interface{}{
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
	}
	return db.AutoMigrate(models...)
}

// 用于避免 time 包未使用的编译错误（占位，后续迁移逻辑可能用到）
var _ = time.Now
