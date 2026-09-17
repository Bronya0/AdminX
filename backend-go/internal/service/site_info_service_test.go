package service

import (
	"context"
	"testing"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// newSiteInfoService 建一个不带 Redis / AES 的站点信息服务（配置缺失时走默认值分支）。
func newSiteInfoService(t *testing.T) *SiteInfoService {
	t.Helper()
	db := setupTestDB(t)
	return NewSiteInfoService(NewConfigService(db, repository.NewConfigRepo(db), nil, nil))
}

// TestSiteInfo_Defaults 没有配置中心记录时返回兜底默认值。
func TestSiteInfo_Defaults(t *testing.T) {
	svc := newSiteInfoService(t)

	info, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if info.SiteName != "AdminX" || info.SiteDesc != "企业级 Admin 框架" {
		t.Errorf("默认站点名/描述不对: %+v", info)
	}
	if info.IdleTimeout != 30 {
		t.Errorf("默认 idle_timeout = %d, want 30", info.IdleTimeout)
	}
	if info.AppVersion != "1.0.0" {
		t.Errorf("默认 app_version = %s, want 1.0.0", info.AppVersion)
	}
}

// TestSiteInfo_UpdatePartial 只写入传入字段，其余保持默认/原值。
func TestSiteInfo_UpdatePartial(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSiteInfoService(NewConfigService(db, repository.NewConfigRepo(db), nil, nil))
	ctx := context.Background()

	name := "运维平台"
	timeout := 0 // 0 = 不超时，是合法值，不能被当成"未传"
	updated, err := svc.Update(ctx, SiteInfoUpdateInput{SiteName: &name, IdleTimeout: &timeout})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.SiteName != name || updated.IdleTimeout != 0 {
		t.Errorf("更新未生效: %+v", updated)
	}
	if updated.SiteDesc != "企业级 Admin 框架" {
		t.Errorf("未传入的字段应保持默认值，实际: %s", updated.SiteDesc)
	}

	// 只改描述：站点名与超时值不变
	desc := "内部运维门户"
	again, err := svc.Update(ctx, SiteInfoUpdateInput{SiteDesc: &desc})
	if err != nil {
		t.Fatalf("二次 Update 失败: %v", err)
	}
	if again.SiteDesc != desc || again.SiteName != name || again.IdleTimeout != 0 {
		t.Errorf("部分更新互相影响: %+v", again)
	}

	// 同 key 反复写入不能产生重复配置行（应为 3 行：name / idle_timeout / desc）
	var count int64
	if err := db.Model(&model.Config{}).Where(`"group" = ?`, siteConfigGroup).Count(&count).Error; err != nil {
		t.Fatalf("统计配置行失败: %v", err)
	}
	if count != 3 {
		t.Errorf("site 分组配置行数 = %d, want 3", count)
	}

	// idle_timeout 落库为 int 类型，读回仍是 int（不是字符串）
	var cfg model.Config
	if err := db.Where("key = ?", siteKeyIdleTimeout).First(&cfg).Error; err != nil {
		t.Fatalf("查询 idle_timeout 配置失败: %v", err)
	}
	if cfg.ValueType != "int" || cfg.Group != siteConfigGroup {
		t.Errorf("idle_timeout 配置元信息不对: type=%s group=%s", cfg.ValueType, cfg.Group)
	}
}

// TestSiteInfo_UpdateValidation 非法入参直接拒绝（不落库）。
func TestSiteInfo_UpdateValidation(t *testing.T) {
	db := setupTestDB(t)
	svc := NewSiteInfoService(NewConfigService(db, repository.NewConfigRepo(db), nil, nil))
	ctx := context.Background()

	blank := "   "
	if _, err := svc.Update(ctx, SiteInfoUpdateInput{SiteName: &blank}); err == nil {
		t.Error("站点名称为空白时应报错")
	}

	negative := -1
	if _, err := svc.Update(ctx, SiteInfoUpdateInput{IdleTimeout: &negative}); err == nil {
		t.Error("负数空闲超时应报错")
	}

	tooLarge := maxIdleTimeoutMinutes + 1
	if _, err := svc.Update(ctx, SiteInfoUpdateInput{IdleTimeout: &tooLarge}); err == nil {
		t.Error("超出上限的空闲超时应报错")
	}

	var count int64
	if err := db.Model(&model.Config{}).Where(`"group" = ?`, siteConfigGroup).Count(&count).Error; err != nil {
		t.Fatalf("统计配置行失败: %v", err)
	}
	if count != 0 {
		t.Errorf("非法入参不应落库，实际写入 %d 行", count)
	}
}
