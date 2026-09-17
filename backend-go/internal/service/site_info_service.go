package service

import (
	"context"
	"strconv"
	"strings"

	apperr "adminx/pkg/errors"
)

// 站点信息存储约定：统一存在配置中心 group=site、key 固定为 site.*。
// 缺失的 key 回落默认值（与前端 store 的初始值一致），因此不需要初始化播种。
const (
	siteConfigGroup = "site"

	siteKeyName        = "site.name"
	siteKeyDesc        = "site.desc"
	siteKeyLogo        = "site.logo"
	siteKeyThemeColor  = "site.theme_color"
	siteKeyIdleTimeout = "site.idle_timeout"
	siteKeyLoginBg     = "site.login_bg_image"
	siteKeyFavicon     = "site.favicon"
	siteKeyAppVersion  = "site.app_version"

	// maxIdleTimeoutMinutes 空闲超时上限（24 小时；0 = 不超时）。
	maxIdleTimeoutMinutes = 1440
)

// SiteInfo 站点信息（对齐前端 adminx-ui/src/api/common.ts 的 SiteInfo 契约）。
type SiteInfo struct {
	SiteName       string `json:"site_name"`
	SiteDesc       string `json:"site_desc"`
	SiteLogo       string `json:"site_logo"`
	SiteThemeColor string `json:"site_theme_color"`
	IdleTimeout    int    `json:"idle_timeout"`
	LoginBgImage   string `json:"login_bg_image"`
	AppVersion     string `json:"app_version"`
	Favicon        string `json:"favicon"`
}

// SiteInfoUpdateInput 站点信息更新入参（指针字段：nil = 不修改）。
type SiteInfoUpdateInput struct {
	SiteName       *string `json:"site_name"`
	SiteDesc       *string `json:"site_desc"`
	SiteLogo       *string `json:"site_logo"`
	SiteThemeColor *string `json:"site_theme_color"`
	IdleTimeout    *int    `json:"idle_timeout"`
	LoginBgImage   *string `json:"login_bg_image"`
	AppVersion     *string `json:"app_version"`
	Favicon        *string `json:"favicon"`
}

// SiteInfoService 站点信息（基于配置中心，读写都走 ConfigService 以复用缓存与类型解析）。
type SiteInfoService struct {
	cfg *ConfigService
}

func NewSiteInfoService(cfg *ConfigService) *SiteInfoService {
	return &SiteInfoService{cfg: cfg}
}

// 默认值：配置中心里没有对应 key 时的兜底。
const (
	defaultSiteName        = "AdminX"
	defaultSiteDesc        = "企业级 Admin 框架"
	defaultSiteThemeColor  = "#1677ff"
	defaultSiteIdleTimeout = 30
	defaultSiteAppVersion  = "1.0.0"
)

// Get 读取站点信息（缺失字段回落默认值）。
func (s *SiteInfoService) Get(ctx context.Context) (*SiteInfo, error) {
	values, err := s.cfg.GetByGroup(ctx, siteConfigGroup)
	if err != nil {
		return nil, apperr.Wrap(500, "读取站点信息失败", err)
	}
	return &SiteInfo{
		SiteName:       configString(values, siteKeyName, defaultSiteName),
		SiteDesc:       configString(values, siteKeyDesc, defaultSiteDesc),
		SiteLogo:       configString(values, siteKeyLogo, ""),
		SiteThemeColor: configString(values, siteKeyThemeColor, defaultSiteThemeColor),
		IdleTimeout:    configInt(values, siteKeyIdleTimeout, defaultSiteIdleTimeout),
		LoginBgImage:   configString(values, siteKeyLoginBg, ""),
		Favicon:        configString(values, siteKeyFavicon, ""),
		AppVersion:     configString(values, siteKeyAppVersion, defaultSiteAppVersion),
	}, nil
}

// Update 保存站点信息：只写入本次传入的字段（其余保持不变），返回保存后的完整值。
func (s *SiteInfoService) Update(ctx context.Context, in SiteInfoUpdateInput) (*SiteInfo, error) {
	// 站点名称为空会让登录页/侧边栏标题变空，明确拒绝
	if in.SiteName != nil {
		name := strings.TrimSpace(*in.SiteName)
		if name == "" {
			return nil, apperr.New(400, "站点名称不能为空")
		}
		in.SiteName = &name
	}

	stringFields := []struct {
		key   string
		desc  string
		value *string
	}{
		{siteKeyName, "站点名称", in.SiteName},
		{siteKeyDesc, "站点描述", in.SiteDesc},
		{siteKeyLogo, "站点 Logo URL", in.SiteLogo},
		{siteKeyThemeColor, "主题色", in.SiteThemeColor},
		{siteKeyLoginBg, "登录页背景图 URL", in.LoginBgImage},
		{siteKeyFavicon, "浏览器标签图标 URL", in.Favicon},
		{siteKeyAppVersion, "站点版本号", in.AppVersion},
	}
	for _, f := range stringFields {
		if f.value == nil {
			continue
		}
		if err := s.cfg.SetValue(f.key, *f.value, "string", f.desc, siteConfigGroup); err != nil {
			return nil, err
		}
	}

	if in.IdleTimeout != nil {
		timeout := *in.IdleTimeout
		if timeout < 0 || timeout > maxIdleTimeoutMinutes {
			return nil, apperr.New(400, "会话空闲超时需在 0~1440 分钟之间（0 = 不超时）")
		}
		if err := s.cfg.SetValue(siteKeyIdleTimeout, strconv.Itoa(timeout), "int",
			"会话空闲超时（分钟，0=不超时）", siteConfigGroup); err != nil {
			return nil, err
		}
	}

	return s.Get(ctx)
}

// configString 取字符串配置（缺失或类型不符时回落默认值）。
func configString(values map[string]interface{}, key, fallback string) string {
	v, ok := values[key]
	if !ok {
		return fallback
	}
	str, ok := v.(string)
	if !ok {
		return fallback
	}
	return str
}

// configInt 取 int 配置。走缓存时 JSON 解码回来是 float64，这里统一归一化。
func configInt(values map[string]interface{}, key string, fallback int) int {
	v, ok := values[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if parsed, err := strconv.Atoi(n); err == nil {
			return parsed
		}
	}
	return fallback
}
