package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	apperr "adminx/pkg/errors"
)

// 全局主题设置（不分用户）：整份 JSON 存在配置中心 group=theme、key=theme.settings。
const (
	themeConfigGroup = "theme"
	themeConfigKey   = "theme.settings"
	themeConfigDesc  = "全局主题设置（JSON，由主题设置页维护）"

	themeFileKind    = "adminx.theme"
	themeFileVersion = 1
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ThemeConfig 全局主题设置。字段名与前端 ThemeConfig 一致（camelCase），前后端直接互传，无需映射。
type ThemeConfig struct {
	PrimaryColor   string `json:"primaryColor"`
	IsDark         bool   `json:"isDark"`
	Layout         string `json:"layout"`
	Collapsed      bool   `json:"collapsed"`
	ShowLogo       bool   `json:"showLogo"`
	ShowBreadcrumb bool   `json:"showBreadcrumb"`
	ShowTabs       bool   `json:"showTabs"`
	ShowFooter     bool   `json:"showFooter"`
	ShowVersion    bool   `json:"showVersion"`
	BorderRadius   int    `json:"borderRadius"`
	FontSize       int    `json:"fontSize"`
	NavHeight      int    `json:"navHeight"`
	HeaderHeight   int    `json:"headerHeight"`
	NavTopBgColor  string `json:"navTopBgColor"`
	NavSideBgColor string `json:"navSideBgColor"`
	NavTextSize    int    `json:"navTextSize"`
	NavTextAuto    bool   `json:"navTextAuto"`
	NavTextColor   string `json:"navTextColor"`
}

// ThemeFile 主题导入导出文件。theme 可以只含部分字段：导入时只覆盖文件里出现的字段。
type ThemeFile struct {
	Kind       string          `json:"kind"`
	Version    int             `json:"version"`
	ExportedAt time.Time       `json:"exported_at"`
	Theme      json.RawMessage `json:"theme"`
}

// ThemeService 全局主题设置（基于配置中心，复用缓存与 JSON 解析）。
type ThemeService struct {
	cfg *ConfigService
}

func NewThemeService(cfg *ConfigService) *ThemeService {
	return &ThemeService{cfg: cfg}
}

// defaultThemeConfig 主题兜底值，与前端 stores/user.ts 的 defaultTheme 保持一致。
func defaultThemeConfig() ThemeConfig {
	return ThemeConfig{
		PrimaryColor: "#1890ff", IsDark: false, Layout: "side", Collapsed: false,
		ShowLogo: true, ShowBreadcrumb: true, ShowTabs: true, ShowFooter: true, ShowVersion: true,
		BorderRadius: 6, FontSize: 14, NavHeight: 56, HeaderHeight: 64,
		NavTopBgColor: "#001529", NavSideBgColor: "#001529", NavTextSize: 14,
		NavTextAuto: true, NavTextColor: "#ffffff",
	}
}

// Get 读取全局主题；配置中心没有记录时返回默认值。
func (s *ThemeService) Get(ctx context.Context) (ThemeConfig, error) {
	cfg := defaultThemeConfig()
	stored, err := s.cfg.GetValue(ctx, themeConfigKey, nil)
	if err != nil {
		return cfg, apperr.Wrap(500, "读取主题设置失败", err)
	}
	if stored == nil {
		return cfg, nil
	}
	// GetValue 已把 JSON 解析成通用结构，这里转回强类型；存量数据里的未知字段直接忽略。
	raw, err := json.Marshal(stored)
	if err != nil {
		return cfg, apperr.ErrInternal
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, apperr.Wrap(500, "主题设置数据损坏", err)
	}
	return cfg, nil
}

// Update 覆盖式更新：raw 里出现的字段覆盖当前值，未出现的字段保持不变。
// 合并后整体校验，通过才落库；未知字段直接报错，避免拼写错误被静默吞掉。
func (s *ThemeService) Update(ctx context.Context, raw []byte) (ThemeConfig, error) {
	cfg, err := s.Get(ctx)
	if err != nil {
		return ThemeConfig{}, err
	}
	if err := decodeStrictJSON(raw, &cfg); err != nil {
		return ThemeConfig{}, apperr.New(400, "主题设置格式无效："+err.Error())
	}
	if err := cfg.validate(); err != nil {
		return ThemeConfig{}, err
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return ThemeConfig{}, apperr.ErrInternal
	}
	if err := s.cfg.SetValue(themeConfigKey, string(encoded), "json", themeConfigDesc, themeConfigGroup); err != nil {
		return ThemeConfig{}, err
	}
	return cfg, nil
}

// ExportFile 导出当前全局主题。
func (s *ThemeService) ExportFile(ctx context.Context) (*ThemeFile, error) {
	cfg, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, apperr.ErrInternal
	}
	return &ThemeFile{
		Kind: themeFileKind, Version: themeFileVersion,
		ExportedAt: time.Now().UTC(), Theme: raw,
	}, nil
}

// ImportFile 导入主题文件（raw 为文件 JSON），按字段覆盖式合并到当前全局主题，返回合并后的完整主题。
func (s *ThemeService) ImportFile(ctx context.Context, raw []byte) (ThemeConfig, error) {
	var file ThemeFile
	if err := decodeStrictJSON(raw, &file); err != nil {
		return ThemeConfig{}, apperr.New(400, "文件格式无效："+err.Error())
	}
	if err := checkSettingsFileHeader(file.Kind, themeFileKind, file.Version, themeFileVersion); err != nil {
		return ThemeConfig{}, err
	}
	if len(file.Theme) == 0 {
		return ThemeConfig{}, apperr.New(400, "文件缺少 theme 字段")
	}
	return s.Update(ctx, file.Theme)
}

// validate 校验主题取值，范围与前端设置页的控件范围一致。
func (c ThemeConfig) validate() error {
	switch c.Layout {
	case "side", "top", "mix":
	default:
		return apperr.New(400, "布局模式只能是 side、top 或 mix")
	}

	colors := []struct {
		name  string
		value string
	}{
		{"主题色", c.PrimaryColor},
		{"顶栏底色", c.NavTopBgColor},
		{"侧栏底色", c.NavSideBgColor},
		{"导航文字颜色", c.NavTextColor},
	}
	for _, col := range colors {
		if !hexColorPattern.MatchString(col.value) {
			return apperr.New(400, fmt.Sprintf("%s必须是 #RRGGBB 格式的颜色", col.name))
		}
	}

	ranges := []struct {
		name          string
		value, lo, hi int
	}{
		{"圆角大小", c.BorderRadius, 0, 16},
		{"字体大小", c.FontSize, 12, 20},
		{"顶栏高度", c.NavHeight, 40, 88},
		{"页头高度", c.HeaderHeight, 40, 88},
		{"导航文字大小", c.NavTextSize, 12, 18},
	}
	for _, r := range ranges {
		if r.value < r.lo || r.value > r.hi {
			return apperr.New(400, fmt.Sprintf("%s需在 %d~%d 之间", r.name, r.lo, r.hi))
		}
	}
	return nil
}
