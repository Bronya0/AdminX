// Package config 负责应用配置的加载与校验。
//
// 配置来源优先级（高 → 低）:
//  1. 环境变量（12-factor，生产部署友好）
//  2. configs/config.yaml（本地开发默认值）
//  3. 结构体默认值
//
// 环境变量前缀 DJA_（缩写），如 DJA_SERVER_PORT。
// 配置文件示例见 configs/config.example.yaml。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用顶层配置。
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DBConfig        `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Security  SecurityConfig  `mapstructure:"security"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	BasePath   string `mapstructure:"base_path"` // 对应 FORCE_SCRIPT_NAME，如 /adminx
	Mode       string `mapstructure:"mode"`      // debug / release / test
	TimeoutSec int    `mapstructure:"timeout_sec"`
}

// DBConfig 数据库配置。
type DBConfig struct {
	Driver          string `mapstructure:"driver"` // postgres (当前仅支持 postgres)
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"` // 如 "1h"
	LogLevel        string `mapstructure:"log_level"`         // silent / error / warn / info
}

// RedisConfig Redis 配置（缓存、限流、调度器锁、JWT 黑名单、Channel Layer）。
type RedisConfig struct {
	URL string `mapstructure:"url"`
}

// JWTConfig JWT 认证配置。
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessExpire  string `mapstructure:"access_expire"`  // 如 "30m"
	RefreshExpire string `mapstructure:"refresh_expire"` // 如 "168h"
	Issuer        string `mapstructure:"issuer"`
}

// LoggingConfig 日志配置。
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // text / json
	Dir    string `mapstructure:"dir"`    // 日志目录，空则输出 stdout
}

// SecurityConfig 安全相关配置。
type SecurityConfig struct {
	// AES-GCM 密钥（32 字节 base64 编码）—— 用于配置中心加密字段。
	// 生成: openssl rand -base64 32
	AESKey string `mapstructure:"aes_key"`

	// CORS 允许的来源，逗号分隔。
	CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`

	// 是否信任代理头（X-Forwarded-For 等）。
	TrustProxyHeaders bool `mapstructure:"trust_proxy_headers"`
}

// SchedulerConfig 调度器配置。
type SchedulerConfig struct {
	// 是否启用调度器（独立进程或同进程运行）。
	Enabled bool `mapstructure:"enabled"`
	// 分布式锁 TTL（秒）。
	LockTTLSec int `mapstructure:"lock_ttl_sec"`
}

// Load 加载配置。configPath 为空时按默认路径查找。
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 默认值
	setDefaults(v)

	// 配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 环境变量覆盖（DJA_ 前缀，下划线分隔，如 DJA_SERVER_PORT）
	v.SetEnvPrefix("DJA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 读取配置文件（不存在不报错，靠环境变量+默认值兜底）
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时静默跳过（环境变量和默认值兜底）
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8000)
	v.SetDefault("server.base_path", "/adminx")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.timeout_sec", 120)

	// Database
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.dsn", "postgres://postgres:postgres@localhost:5432/adminx?sslmode=disable")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.log_level", "warn")

	// Redis
	v.SetDefault("redis.url", "redis://localhost:6379/0")

	// JWT
	v.SetDefault("jwt.secret", "change-me-in-production")
	v.SetDefault("jwt.access_expire", "30m")
	v.SetDefault("jwt.refresh_expire", "168h")
	v.SetDefault("jwt.issuer", "adminx")

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.dir", "")

	// Scheduler
	v.SetDefault("scheduler.enabled", false)
	v.SetDefault("scheduler.lock_ttl_sec", 30)
}

// validate 校验关键配置项。
func (c *Config) validate() error {
	// JWT 密钥不允许默认值、空值或示例值（任何模式都禁止，防止误用公开密钥伪造 token）
	if c.JWT.Secret == "" || c.JWT.Secret == "change-me-in-production" || c.JWT.Secret == "dev-only-change-me-9f8e7d6c5b4a3210" {
		return fmt.Errorf("必须配置安全的 jwt.secret（不允许使用默认值/示例值，可通过 DJA_JWT_SECRET 环境变量或 config.yaml 设置）")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn 不能为空")
	}
	// 解析 JWT 过期时间，确保格式正确
	if _, err := time.ParseDuration(c.JWT.AccessExpire); err != nil {
		return fmt.Errorf("jwt.access_expire 格式错误（应为如 30m）: %w", err)
	}
	if _, err := time.ParseDuration(c.JWT.RefreshExpire); err != nil {
		return fmt.Errorf("jwt.refresh_expire 格式错误（应为如 168h）: %w", err)
	}
	if _, err := time.ParseDuration(c.Database.ConnMaxLifetime); err != nil {
		return fmt.Errorf("database.conn_max_lifetime 格式错误: %w", err)
	}
	return nil
}

// AccessExpireDuration 返回 access token 过期时间的 time.Duration。
func (c *Config) AccessExpireDuration() time.Duration {
	d, _ := time.ParseDuration(c.JWT.AccessExpire)
	return d
}

// RefreshExpireDuration 返回 refresh token 过期时间的 time.Duration。
func (c *Config) RefreshExpireDuration() time.Duration {
	d, _ := time.ParseDuration(c.JWT.RefreshExpire)
	return d
}

// ConnMaxLifetimeDuration 返回数据库连接最大生命周期的 time.Duration。
func (c *Config) ConnMaxLifetimeDuration() time.Duration {
	d, _ := time.ParseDuration(c.Database.ConnMaxLifetime)
	return d
}
