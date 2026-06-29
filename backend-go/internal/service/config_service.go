package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/datatypes"

	apperr "adminx/pkg/errors"
	"adminx/pkg/crypto"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// ConfigDTO 配置列表/详情返回给前端的 DTO（隐藏加密值）。
type ConfigDTO struct {
	ID           int64     `json:"id"`
	Key          string    `json:"key"`
	Value        string    `json:"value"`
	DisplayValue string    `json:"display_value"`
	ValueType    string    `json:"value_type"`
	IsEncrypted  bool      `json:"is_encrypted"`
	Desc         string    `json:"desc"`
	Group        string    `json:"group"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toConfigDTO(c *model.Config) ConfigDTO {
	dto := ConfigDTO{
		ID: c.ID, Key: c.Key, ValueType: c.ValueType,
		IsEncrypted: c.IsEncrypted, Desc: c.Desc, Group: c.Group,
		IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.IsEncrypted {
		dto.Value = ""
		dto.DisplayValue = "********"
	} else {
		dto.Value = c.Value
		dto.DisplayValue = c.Value
	}
	return dto
}

func toConfigDTOs(cfgs []model.Config) []ConfigDTO {
	dtos := make([]ConfigDTO, len(cfgs))
	for i := range cfgs {
		dtos[i] = toConfigDTO(&cfgs[i])
	}
	return dtos
}

// ConfigService 配置中心业务（AES-GCM 加密 + Redis 缓存）。
type ConfigService struct {
	db       *gorm.DB
	repo     *repository.ConfigRepo
	rdb      *redis.Client
	aes      *crypto.AESGCM // 可为 nil（未配置加密时）
}

// NewConfigService 构造。aesGCM 为 nil 时加密配置不可用。
func NewConfigService(db *gorm.DB, repo *repository.ConfigRepo, rdb *redis.Client, aesGCM *crypto.AESGCM) *ConfigService {
	return &ConfigService{db: db, repo: repo, rdb: rdb, aes: aesGCM}
}

const (
	configCacheTTL    = time.Hour
	configCachePrefix = "config:"
	configGroupPrefix = "config_group:"
)

// List 分页查询配置列表。
func (s *ConfigService) List(offset, limit int, search, group string) ([]ConfigDTO, int64, error) {
	cfgs, count, err := s.repo.List(offset, limit, search, group)
	if err != nil {
		return nil, 0, err
	}
	return toConfigDTOs(cfgs), count, nil
}

// Groups 查询所有分组。
func (s *ConfigService) Groups() ([]string, error) {
	return s.repo.Groups()
}

// GetByID 按 ID 查询。
func (s *ConfigService) GetByID(id int64) (*ConfigDTO, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	dto := toConfigDTO(c)
	return &dto, nil
}

// CreateInput 创建配置入参。
type ConfigCreateInput struct {
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	IsEncrypted bool   `json:"is_encrypted"`
	Desc        string `json:"desc"`
	Group       string `json:"group"`
	IsActive    *bool  `json:"is_active"`
}

func (s *ConfigService) Create(in ConfigCreateInput) (*model.Config, error) {
	valueType := in.ValueType
	if valueType == "" {
		valueType = "string"
	}
	group := in.Group
	if group == "" {
		group = "default"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	cfg := &model.Config{
		Key:       in.Key,
		Value:     in.Value,
		ValueType: valueType,
		Desc:      in.Desc,
		Group:     group,
		IsActive:  isActive,
	}

	// 加密处理
	if in.IsEncrypted {
		if s.aes == nil {
			return nil, apperr.New(500, "加密功能未配置（缺少 AES 密钥）")
		}
		encrypted, err := s.aes.Encrypt(in.Value)
		if err != nil {
			return nil, apperr.Wrap(500, "加密失败", err)
		}
		cfg.Value = encrypted
		cfg.IsEncrypted = true
	}

	if err := s.repo.Create(cfg); err != nil {
		return nil, apperr.ErrInternal
	}
	s.invalidateCache(cfg.Key, cfg.Group)
	return cfg, nil
}

// UpdateInput 更新配置入参。
type ConfigUpdateInput struct {
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Desc        string `json:"desc"`
	Group       string `json:"group"`
	IsActive    *bool  `json:"is_active"`
}

func (s *ConfigService) Update(id int64, in ConfigUpdateInput) (*model.Config, error) {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}

	oldKey, oldGroup := cfg.Key, cfg.Group

	// 加密配置：加密后不允许改回明文
	if cfg.IsEncrypted && in.Value != "" {
		if s.aes == nil {
			return nil, apperr.New(500, "加密功能未配置")
		}
		encrypted, err := s.aes.Encrypt(in.Value)
		if err != nil {
			return nil, apperr.Wrap(500, "加密失败", err)
		}
		cfg.Value = encrypted
	} else if in.Value != "" {
		cfg.Value = in.Value
	}

	if in.ValueType != "" {
		cfg.ValueType = in.ValueType
	}
	cfg.Desc = in.Desc
	if in.Group != "" {
		cfg.Group = in.Group
	}
	if in.IsActive != nil {
		cfg.IsActive = *in.IsActive
	}

	if err := s.repo.Update(cfg); err != nil {
		return nil, apperr.ErrInternal
	}
	// 清除新旧缓存（key/group 可能变更）
	s.invalidateCache(cfg.Key, cfg.Group)
	if oldKey != cfg.Key || oldGroup != cfg.Group {
		s.invalidateCache(oldKey, oldGroup)
	}
	return cfg, nil
}

func (s *ConfigService) Delete(id int64) error {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(id); err != nil {
		return apperr.ErrInternal
	}
	s.invalidateCache(cfg.Key, cfg.Group)
	return nil
}

// GetValue 按 key 读取配置值（带缓存 + 解密 + 类型解析）。
func (s *ConfigService) GetValue(ctx context.Context, key string, defaultVal interface{}) (interface{}, error) {
	cacheKey := configCachePrefix + key
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			return jsonDecode(cached)
		}
	}

	cfg, err := s.repo.FindByKey(key)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultVal, nil
		}
		return nil, err
	}

	val := s.parseValue(cfg)
	// 仅缓存非加密配置；加密配置的明文不写 Redis（避免敏感信息泄漏）
	if s.rdb != nil && !cfg.IsEncrypted {
		if b, err := jsonEncode(val); err == nil {
			_ = s.rdb.Set(ctx, cacheKey, b, configCacheTTL).Err()
		}
	}
	return val, nil
}

// GetByGroup 按 group 批量读取（带缓存）。返回 {key: value} 字典。
func (s *ConfigService) GetByGroup(ctx context.Context, group string) (map[string]interface{}, error) {
	cacheKey := configGroupPrefix + group
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var out map[string]interface{}
			if json.Unmarshal([]byte(cached), &out) == nil {
				return out, nil
			}
		}
	}

	cfgs, err := s.repo.FindByGroup(group)
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(cfgs))
	for _, c := range cfgs {
		out[c.Key] = s.parseValue(&c)
	}
	// 仅缓存不含加密配置的 group（避免敏感信息泄漏到 Redis）
	hasEncrypted := false
	for _, c := range cfgs {
		if c.IsEncrypted {
			hasEncrypted = true
			break
		}
	}
	if s.rdb != nil && !hasEncrypted {
		if b, err := json.Marshal(out); err == nil {
			_ = s.rdb.Set(ctx, cacheKey, b, configCacheTTL).Err()
		}
	}
	return out, nil
}

// parseValue 按 valueType 解析配置值。
// 加密配置自动解密；解密失败返回 "<<解密失败>>"。
func (s *ConfigService) parseValue(c *model.Config) interface{} {
	raw := c.Value

	// 加密配置解密
	if c.IsEncrypted {
		if s.aes == nil {
			slog.Default().Warn("AES 密钥未配置，无法解密", "key", c.Key)
			return "<<解密失败>>"
		}
		decrypted, err := s.aes.Decrypt(raw)
		if err != nil {
			slog.Default().Warn("配置解密失败", "key", c.Key, "error", err)
			return "<<解密失败>>"
		}
		raw = decrypted
	}

	switch c.ValueType {
	case "int":
		if raw == "" {
			return 0
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0
		}
		return n
	case "bool":
		lower := toLower(raw)
		return lower == "true" || lower == "1" || lower == "yes"
	case "json":
		if raw == "" {
			return nil
		}
		var v interface{}
		if json.Unmarshal([]byte(raw), &v) != nil {
			return nil
		}
		return v
	case "options":
		if raw == "" {
			return []interface{}{}
		}
		var v []interface{}
		if json.Unmarshal([]byte(raw), &v) != nil {
			return []interface{}{}
		}
		return v
	default: // string
		return raw
	}
}

// invalidateCache 清除指定 key/group 的缓存。
func (s *ConfigService) invalidateCache(key, group string) {
	if s.rdb == nil {
		return
	}
	ctx := context.Background()
	_ = s.rdb.Del(ctx, configCachePrefix+key).Err()
	_ = s.rdb.Del(ctx, configGroupPrefix+group).Err()
}

// ── 审计辅助 ──

// RecordAudit 记录审计日志（service 层显式调用，不依赖 signal）。
func (s *ConfigService) RecordAudit(repo *repository.AuditRepo, action, modelName, objectID, objectRepr, operator, ip string, oldVals, newVals datatypes.JSON, diff string) {
	log := &model.AuditLog{
		Action:      action,
		ModelName:   modelName,
		ObjectID:    objectID,
		ObjectRepr:  objectRepr,
		Operator:    operator,
		OperatorIP:  ip,
		OldValues:   oldVals,
		NewValues:   newVals,
		DiffSummary: truncate(diff, 500),
	}
	_ = repo.Create(log)
}

// ── 辅助函数 ──

func jsonEncode(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func jsonDecode(s string) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

var _ = fmt.Sprintf
