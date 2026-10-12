package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// 字典配置导入导出文件格式（版本 1）。
// 导入为 key 覆盖式：key 已存在则整条覆盖，不存在则新建，文件之外的配置保持不变。
const (
	configFileKind       = "adminx.config"
	configFileVersion    = 1
	maxConfigImportItems = 1000
)

// ConfigFile 字典配置导入导出文件。
type ConfigFile struct {
	Kind       string           `json:"kind"`
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Items      []ConfigFileItem `json:"items"`
}

// ConfigFileItem 文件中的一条配置。加密配置的明文无法导出，因此文件里不存在加密项。
type ConfigFileItem struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	ValueType string `json:"value_type"`
	Group     string `json:"group"`
	Desc      string `json:"desc"`
	IsActive  *bool  `json:"is_active"`
}

// ConfigImportResult 导入结果。
type ConfigImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

var intValuePattern = regexp.MustCompile(`^-?\d+$`)

// serviceOwnedGroups 由专用服务维护的配置分组，值为该分组对应的维护入口（只用于报错提示）。
// 这些分组的整份值有各自的 schema 与校验（主题走 ThemeService、站点信息走 SiteInfoService），
// 字典导入导出一律绕开：字典只校验「值是合法 JSON」，一份手改过的文件就能把主题写成
// 非法值（例如 layout 取到未知布局，前端所有布局分支都不成立，页面直接白屏）。
// 新增专用分组时必须登记到这里。
var serviceOwnedGroups = map[string]string{
	themeConfigGroup: "主题设置",
	siteConfigGroup:  "站点信息",
}

// ExportFile 导出全部非加密配置（含停用项），跳过由专用服务维护的分组。
func (s *ConfigService) ExportFile() (*ConfigFile, error) {
	cfgs, err := s.repo.FindPlain()
	if err != nil {
		return nil, apperr.ErrInternal
	}
	items := make([]ConfigFileItem, 0, len(cfgs))
	for _, c := range cfgs {
		if _, owned := serviceOwnedGroups[c.Group]; owned {
			continue
		}
		isActive := c.IsActive
		items = append(items, ConfigFileItem{
			Key: c.Key, Value: c.Value, ValueType: c.ValueType,
			Group: c.Group, Desc: c.Desc, IsActive: &isActive,
		})
	}
	return &ConfigFile{
		Kind: configFileKind, Version: configFileVersion,
		ExportedAt: time.Now().UTC(), Items: items,
	}, nil
}

// ImportFile 按 key 覆盖式导入 raw（文件 JSON）。
// 整批先校验，全部通过后才在一个事务内写入：任一条不合法、属于专用服务分组
// 或与加密配置冲突，整批都不写入。
func (s *ConfigService) ImportFile(ctx context.Context, raw []byte) (*ConfigImportResult, error) {
	var file ConfigFile
	if err := decodeStrictJSON(raw, &file); err != nil {
		return nil, apperr.New(400, "文件格式无效："+err.Error())
	}
	if err := checkSettingsFileHeader(file.Kind, configFileKind, file.Version, configFileVersion); err != nil {
		return nil, err
	}
	if len(file.Items) == 0 {
		return nil, apperr.New(400, "文件中没有可导入的配置")
	}
	if len(file.Items) > maxConfigImportItems {
		return nil, apperr.New(400, fmt.Sprintf("单次最多导入 %d 条配置", maxConfigImportItems))
	}

	records := make([]model.Config, 0, len(file.Items))
	seen := make(map[string]bool, len(file.Items))
	for i, item := range file.Items {
		rec, err := item.toRecord()
		if err != nil {
			return nil, apperr.New(400, fmt.Sprintf("第 %d 条配置（%s）无效：%s", i+1, strings.TrimSpace(item.Key), err.Error()))
		}
		if seen[rec.Key] {
			return nil, apperr.New(400, "文件中配置键重复："+rec.Key)
		}
		seen[rec.Key] = true
		if label, owned := serviceOwnedGroups[rec.Group]; owned {
			return nil, apperr.New(400, fmt.Sprintf("第 %d 条配置（%s）属于「%s」，不能用字典导入", i+1, rec.Key, label))
		}
		records = append(records, rec)
	}

	result := &ConfigImportResult{}
	// 被覆盖前所在的分组，用于事务提交后清掉旧分组的缓存
	oldGroups := make(map[string]string)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repository.NewConfigRepo(tx)
		for i := range records {
			rec := &records[i]
			existing, err := repo.FindAnyByKey(rec.Key)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := repo.Create(rec); err != nil {
					return apperr.ErrInternal
				}
				result.Created++
				continue
			}
			if err != nil {
				return apperr.ErrInternal
			}
			if existing.IsEncrypted {
				return apperr.New(400, "配置 "+rec.Key+" 是加密配置，不能被导入覆盖")
			}
			// 文件里改了分组也绕不过去：不管文件怎么写，专用分组的存量配置都不许被字典导入改写
			if label, owned := serviceOwnedGroups[existing.Group]; owned {
				return apperr.New(400, "配置 "+rec.Key+" 由「"+label+"」维护，不能用字典导入覆盖")
			}
			oldGroups[rec.Key] = existing.Group
			existing.Value = rec.Value
			existing.ValueType = rec.ValueType
			existing.Desc = rec.Desc
			existing.Group = rec.Group
			existing.IsActive = rec.IsActive
			if err := repo.Update(existing); err != nil {
				return apperr.ErrInternal
			}
			result.Updated++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, rec := range records {
		s.invalidateCache(rec.Key, rec.Group)
		if oldGroup, ok := oldGroups[rec.Key]; ok {
			s.invalidateCache(rec.Key, oldGroup)
		}
	}
	return result, nil
}

// toRecord 把文件条目规范化为可落库的配置，并校验字段长度与值类型。
func (it ConfigFileItem) toRecord() (model.Config, error) {
	key := strings.TrimSpace(it.Key)
	if key == "" {
		return model.Config{}, errors.New("配置键不能为空")
	}
	if utf8.RuneCountInString(key) > 255 {
		return model.Config{}, errors.New("配置键不能超过 255 个字符")
	}
	group := strings.TrimSpace(it.Group)
	if group == "" {
		group = "default"
	}
	if utf8.RuneCountInString(group) > 128 {
		return model.Config{}, errors.New("分组名不能超过 128 个字符")
	}
	if utf8.RuneCountInString(it.Desc) > 500 {
		return model.Config{}, errors.New("描述不能超过 500 个字符")
	}
	valueType := it.ValueType
	if valueType == "" {
		valueType = "string"
	}
	if err := validateConfigValue(valueType, it.Value); err != nil {
		return model.Config{}, err
	}
	isActive := true
	if it.IsActive != nil {
		isActive = *it.IsActive
	}
	return model.Config{
		Key: key, Value: it.Value, ValueType: valueType,
		Group: group, Desc: it.Desc, IsActive: isActive,
	}, nil
}

// validateConfigValue 按值类型校验配置值，规则与前端配置编辑表单保持一致。
func validateConfigValue(valueType, value string) error {
	switch valueType {
	case "string":
		return nil
	case "int":
		if !intValuePattern.MatchString(value) {
			return errors.New("整数类型的值必须是整数")
		}
		if _, err := strconv.Atoi(value); err != nil {
			return errors.New("整数超出范围")
		}
		return nil
	case "bool":
		if lower := strings.ToLower(value); lower != "true" && lower != "false" {
			return errors.New("布尔类型的值只能是 true 或 false")
		}
		return nil
	case "json":
		var v interface{}
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			return errors.New("JSON 格式无效")
		}
		switch v.(type) {
		case map[string]interface{}, []interface{}:
			return nil
		}
		return errors.New("JSON 类型的值必须是对象或数组")
	case "options":
		if value == "" {
			return nil
		}
		var opts []struct {
			Label *string `json:"label"`
			Value *string `json:"value"`
		}
		if err := json.Unmarshal([]byte(value), &opts); err != nil {
			return errors.New(`选项列表格式无效，应为 [{"label":"显示名","value":"值"}]`)
		}
		for _, o := range opts {
			if o.Label == nil || o.Value == nil {
				return errors.New(`选项列表每项都需包含 label 与 value 字符串`)
			}
		}
		return nil
	default:
		return fmt.Errorf("不支持的值类型 %q", valueType)
	}
}
