package service

import (
	"context"
	"encoding/json"
	"testing"

	"adminx/internal/repository"
)

func newThemeServiceWithDB(t *testing.T) (*ThemeService, *ConfigService) {
	t.Helper()
	db := setupTestDB(t)
	cfg := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)
	return NewThemeService(cfg), cfg
}

// 没有任何记录时返回默认主题。
func TestTheme_DefaultWhenMissing(t *testing.T) {
	svc, _ := newThemeServiceWithDB(t)

	cfg, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if cfg != defaultThemeConfig() {
		t.Errorf("默认主题不对: %+v", cfg)
	}
}

// 局部更新只改传入的字段，其余保持原值，并且真正落库（换一个 service 实例读回来一致）。
func TestTheme_PartialUpdatePersists(t *testing.T) {
	svc, cfgSvc := newThemeServiceWithDB(t)
	ctx := context.Background()

	if _, err := svc.Update(ctx, []byte(`{"layout":"top","isDark":true}`)); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	reloaded, err := NewThemeService(cfgSvc).Get(ctx)
	if err != nil {
		t.Fatalf("重新读取失败: %v", err)
	}
	want := defaultThemeConfig()
	want.Layout = "top"
	want.IsDark = true
	if reloaded != want {
		t.Errorf("持久化结果 = %+v, want %+v", reloaded, want)
	}
}

// 非法值与未知字段都不能写入，且已有设置保持不变。
func TestTheme_RejectsInvalidAndKeepsStored(t *testing.T) {
	svc, _ := newThemeServiceWithDB(t)
	ctx := context.Background()
	if _, err := svc.Update(ctx, []byte(`{"layout":"top"}`)); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	bad := []string{
		`{"layout":"left"}`,
		`{"primaryColor":"red"}`,
		`{"navTextColor":"#fff"}`,
		`{"borderRadius":99}`,
		`{"fontSize":"big"}`,
		`{"colour":"#ffffff"}`,
	}
	for _, body := range bad {
		if _, err := svc.Update(ctx, []byte(body)); err == nil {
			t.Errorf("%s：应当被拒绝", body)
		}
	}

	cfg, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if cfg.Layout != "top" || cfg.PrimaryColor != "#1890ff" {
		t.Errorf("非法更新污染了已保存设置: %+v", cfg)
	}
}

// 导出的文件再导入到另一个库：字段一致；错误类型的文件被拒绝。
func TestTheme_ExportImportRoundTrip(t *testing.T) {
	ctx := context.Background()
	src, _ := newThemeServiceWithDB(t)
	if _, err := src.Update(ctx, []byte(`{"fontSize":16,"navTextAuto":false,"navTextColor":"#000000"}`)); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	file, err := src.ExportFile(ctx)
	if err != nil {
		t.Fatalf("ExportFile 失败: %v", err)
	}
	raw, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	dst, _ := newThemeServiceWithDB(t)
	got, err := dst.ImportFile(ctx, raw)
	if err != nil {
		t.Fatalf("ImportFile 失败: %v", err)
	}
	if got.FontSize != 16 || got.NavTextAuto || got.NavTextColor != "#000000" {
		t.Errorf("导入结果不对: %+v", got)
	}

	if _, err := dst.ImportFile(ctx, []byte(`{"kind":"adminx.config","version":1,"theme":{}}`)); err == nil {
		t.Error("字典文件不能导入为主题")
	}
	if _, err := dst.ImportFile(ctx, []byte(`{"kind":"adminx.theme","version":1}`)); err == nil {
		t.Error("缺少 theme 字段应当被拒绝")
	}
}
