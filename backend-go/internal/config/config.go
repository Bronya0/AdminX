// Package config 定义应用配置结构与加载逻辑。
//
// 当前为骨架，仅定义结构体，加载逻辑（从 yaml/env 读取）待实现。
// 配置示例见 configs/config.example.yaml。
package config

// Config 是应用顶层配置。
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	DB      DBConfig      `yaml:"database"`
	Redis   RedisConfig   `yaml:"redis"`
	JWT     JWTConfig     `yaml:"jwt"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	BasePath  string `yaml:"base_path"`  // 对应 Django 的 FORCE_SCRIPT_NAME，如 /djangoadminx
	TimeoutSec int   `yaml:"timeout_sec"`
}

// DBConfig 数据库配置。
type DBConfig struct {
	Driver string `yaml:"driver"` // postgres | sqlite | mysql
	DSN    string `yaml:"dsn"`    // 连接串
}

// RedisConfig Redis 配置（缓存、限流、调度器锁、Channel Layer）。
type RedisConfig struct {
	URL string `yaml:"url"`
}

// JWTConfig JWT 认证配置。
type JWTConfig struct {
	Secret          string `yaml:"secret"`
	AccessExpire    string `yaml:"access_expire"`  // 如 "30m"
	RefreshExpire   string `yaml:"refresh_expire"` // 如 "168h"
	Issuer          string `yaml:"issuer"`
}

// LoggingConfig 日志配置。
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug | info | warn | error
	Format string `yaml:"format"` // text | json
	Dir    string `yaml:"dir"`    // 日志目录
}
