package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

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

// ToConfigDTO 转换配置为响应 DTO（加密值只显示 ********）。
// 写操作也返回它：模型一旦被直接序列化就会漏出密文列。
func ToConfigDTO(c *model.Config) ConfigDTO {
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
		dtos[i] = ToConfigDTO(&cfgs[i])
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
	dto := ToConfigDTO(c)
	return &dto, nil
}

// assertGroupWritable 拒绝通过通用配置接口改写由专用服务维护的分组（见 serviceOwnedGroups）。
// 这些分组的整份值有各自的 schema 与校验，被写成“合法 JSON 但非法布局”会让前端整站白屏。
func assertGroupWritable(groups ...string) error {
	for _, g := range groups {
		if label, owned := serviceOwnedGroups[g]; owned {
			return apperr.New(400, "分组 "+g+" 由"+label+"维护，请在对应设置页修改")
		}
	}
	return nil
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
	if err := assertGroupWritable(group); err != nil {
		return nil, err
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

// UpdateInput 更新配置入参。Desc 用指针区分「未传」（保持原值）与「传空串」（清空）：
// 非指针字段被无条件赋值时，只改 is_active 的 PATCH 会把描述一并清掉。
type ConfigUpdateInput struct {
	Value     string  `json:"value"`
	ValueType string  `json:"value_type"`
	Desc      *string `json:"desc"`
	Group     string  `json:"group"`
	IsActive  *bool   `json:"is_active"`
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
	// 新旧分组都要判：把分组改走同样会让专用服务读不到自己的配置
	if err := assertGroupWritable(oldGroup, in.Group); err != nil {
		return nil, err
	}

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
	if in.Desc != nil {
		cfg.Desc = *in.Desc
	}
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
	if err := assertGroupWritable(cfg.Group); err != nil {
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return apperr.ErrInternal
	}
	s.invalidateCache(cfg.Key, cfg.Group)
	return nil
}

// SetValue 按固定 key 写入明文配置（不存在则创建，存在则覆盖值/类型/描述）。
// 用于站点信息这类"按约定 key 键值化"的场景；加密配置只能走 Create/Update，
// 避免误把密文当明文覆盖。
func (s *ConfigService) SetValue(key, value, valueType, desc, group string) error {
	if valueType == "" {
		valueType = "string"
	}
	if group == "" {
		group = "default"
	}

	cfg, err := s.repo.FindAnyByKey(key)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return apperr.ErrInternal
		}
		created := &model.Config{
			Key: key, Value: value, ValueType: valueType,
			Desc: desc, Group: group, IsActive: true,
		}
		if err := s.repo.Create(created); err != nil {
			return apperr.ErrInternal
		}
		s.invalidateCache(created.Key, created.Group)
		return nil
	}
	if cfg.IsEncrypted {
		return apperr.New(400, "加密配置不支持按键覆盖写入")
	}

	oldGroup := cfg.Group
	cfg.Value = value
	cfg.ValueType = valueType
	cfg.Desc = desc
	cfg.Group = group
	cfg.IsActive = true
	if err := s.repo.Update(cfg); err != nil {
		return apperr.ErrInternal
	}
	s.invalidateCache(cfg.Key, cfg.Group)
	if oldGroup != cfg.Group {
		s.invalidateCache(cfg.Key, oldGroup)
	}
	return nil
}

// ConfigValue 待写入的一条键值（含元信息）。
type ConfigValue struct {
	Value     string
	ValueType string
	Desc      string
	Group     string
}

// SetValues 单事务批量写入多条明文配置（不存在则创建，存在则覆盖）。
// 用于站点信息这类“一次改多个 key”的场景：逐条 SetValue 时中途失败会留下半份配置。
func (s *ConfigService) SetValues(ctx context.Context, kvs map[string]ConfigValue) error {
	if len(kvs) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewConfigRepo(tx)
		for key, kv := range kvs {
			valueType := kv.ValueType
			if valueType == "" {
				valueType = "string"
			}
			group := kv.Group
			if group == "" {
				group = "default"
			}
			cfg, err := repo.FindAnyByKey(key)
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return apperr.ErrInternal
				}
				created := &model.Config{
					Key: key, Value: kv.Value, ValueType: valueType,
					Desc: kv.Desc, Group: group, IsActive: true,
				}
				if err := repo.Create(created); err != nil {
					return apperr.ErrInternal
				}
				continue
			}
			if cfg.IsEncrypted {
				return apperr.New(400, "加密配置不支持按键覆盖写入")
			}
			cfg.Value = kv.Value
			cfg.ValueType = valueType
			cfg.Desc = kv.Desc
			cfg.Group = group
			cfg.IsActive = true
			if err := repo.Update(cfg); err != nil {
				return apperr.ErrInternal
			}
		}
		return nil
	}); err != nil {
		return err
	}
	for key, kv := range kvs {
		group := kv.Group
		if group == "" {
			group = "default"
		}
		s.invalidateCache(key, group)
	}
	// 批量写入可能跨分组：顺带清掉整组缓存（分组 key 变更同样失效）
	seenGroups := map[string]bool{}
	for _, kv := range kvs {
		group := kv.Group
		if group == "" {
			group = "default"
		}
		if !seenGroups[group] {
			seenGroups[group] = true
			if s.rdb != nil {
				_ = s.rdb.Del(ctx, configGroupPrefix+group).Err()
			}
		}
	}
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
