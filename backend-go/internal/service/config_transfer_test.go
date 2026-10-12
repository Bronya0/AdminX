package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/gorm"

	"adminx/internal/model"
	"adminx/internal/repository"
)

func mustCreateConfig(t *testing.T, svc *ConfigService, in ConfigCreateInput) {
	t.Helper()
	if _, err := svc.Create(in); err != nil {
		t.Fatalf("Create(%s) 失败: %v", in.Key, err)
	}
}

func mustFindConfig(t *testing.T, repo *repository.ConfigRepo, key string) *model.Config {
	t.Helper()
	cfg, err := repo.FindAnyByKey(key)
	if err != nil {
		t.Fatalf("查询 %s 失败: %v", key, err)
	}
	return cfg
}

func TestConfigImport_OverwritesByKeyAndCreatesNew(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewConfigRepo(db)
	svc := NewConfigService(db, repo, nil, newTestAES())
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "site.title", Value: "old", Group: "g1", Desc: "旧描述"})
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "keep.me", Value: "x"})

	result, err := svc.ImportFile(context.Background(), []byte(`{
		"kind": "adminx.config", "version": 1,
		"items": [
			{"key": "site.title", "value": "new", "value_type": "string", "group": "g2", "desc": "新描述", "is_active": false},
			{"key": "page.size", "value": "20", "value_type": "int"}
		]
	}`))
	if err != nil {
		t.Fatalf("ImportFile 失败: %v", err)
	}
	if result.Created != 1 || result.Updated != 1 {
		t.Fatalf("结果 = %+v, want created=1 updated=1", result)
	}

	title := mustFindConfig(t, repo, "site.title")
	if title.Value != "new" || title.Group != "g2" || title.Desc != "新描述" || title.IsActive {
		t.Errorf("site.title 未被整条覆盖: %+v", title)
	}
	size := mustFindConfig(t, repo, "page.size")
	if size.Value != "20" || size.ValueType != "int" || size.Group != "default" || !size.IsActive {
		t.Errorf("page.size 新建结果不对: %+v", size)
	}
	if keep := mustFindConfig(t, repo, "keep.me"); keep.Value != "x" {
		t.Errorf("文件之外的配置被改动了: %+v", keep)
	}
}

// 新建的停用项必须保持停用（GORM 的 default:true 不能把 false 吞掉）。
func TestConfigImport_NewInactiveItemStaysInactive(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewConfigRepo(db)
	svc := NewConfigService(db, repo, nil, nil)

	if _, err := svc.ImportFile(context.Background(), []byte(`{
		"kind": "adminx.config", "version": 1,
		"items": [{"key": "off.flag", "value": "true", "value_type": "bool", "is_active": false}]
	}`)); err != nil {
		t.Fatalf("ImportFile 失败: %v", err)
	}
	if cfg := mustFindConfig(t, repo, "off.flag"); cfg.IsActive {
		t.Error("停用项导入后变成了启用")
	}
}

func TestConfigImport_RejectsWholeBatchWhenAnyItemInvalid(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewConfigRepo(db)
	svc := NewConfigService(db, repo, nil, nil)

	_, err := svc.ImportFile(context.Background(), []byte(`{
		"kind": "adminx.config", "version": 1,
		"items": [
			{"key": "good.one", "value": "a", "value_type": "string"},
			{"key": "bad.num", "value": "abc", "value_type": "int"}
		]
	}`))
	if err == nil {
		t.Fatal("非法整数应当导入失败")
	}
	if _, err := repo.FindAnyByKey("good.one"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("整批失败后不应留下部分写入，good.one 查询结果: %v", err)
	}
}

func TestConfigImport_RejectsOverwritingEncryptedConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewConfigRepo(db)
	svc := NewConfigService(db, repo, nil, newTestAES())
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "secret.token", Value: "cipher-src", IsEncrypted: true})
	before := mustFindConfig(t, repo, "secret.token").Value

	_, err := svc.ImportFile(context.Background(), []byte(`{
		"kind": "adminx.config", "version": 1,
		"items": [
			{"key": "plain.new", "value": "x", "value_type": "string"},
			{"key": "secret.token", "value": "hijack", "value_type": "string"}
		]
	}`))
	if err == nil {
		t.Fatal("覆盖加密配置应当被拒绝")
	}
	secret := mustFindConfig(t, repo, "secret.token")
	if !secret.IsEncrypted || secret.Value != before {
		t.Errorf("加密配置被改动: %+v", secret)
	}
	if _, err := repo.FindAnyByKey("plain.new"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("冲突时整批应回滚，plain.new 查询结果: %v", err)
	}
}

func TestConfigImport_RejectsEncryptedFieldAndUnknownFields(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)

	cases := map[string]string{
		"加密项":     `{"kind":"adminx.config","version":1,"items":[{"key":"a","value":"b","is_encrypted":true}]}`,
		"未知字段":    `{"kind":"adminx.config","version":1,"items":[{"key":"a","value":"b"}],"extra":1}`,
		"布尔非法":    `{"kind":"adminx.config","version":1,"items":[{"key":"a","value":"yes","value_type":"bool"}]}`,
		"JSON 非法": `{"kind":"adminx.config","version":1,"items":[{"key":"a","value":"{","value_type":"json"}]}`,
	}
	for name, body := range cases {
		if _, err := svc.ImportFile(context.Background(), []byte(body)); err == nil {
			t.Errorf("%s：应当被拒绝", name)
		}
	}
}

func TestConfigImport_RejectsWrongKindVersionAndDuplicates(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)

	cases := map[string]string{
		"主题文件": `{"kind":"adminx.theme","version":1,"items":[{"key":"a","value":"b"}]}`,
		"版本不符": `{"kind":"adminx.config","version":2,"items":[{"key":"a","value":"b"}]}`,
		"键重复":  `{"kind":"adminx.config","version":1,"items":[{"key":"a","value":"1"},{"key":"a","value":"2"}]}`,
		"没有条目": `{"kind":"adminx.config","version":1,"items":[]}`,
	}
	for name, body := range cases {
		if _, err := svc.ImportFile(context.Background(), []byte(body)); err == nil {
			t.Errorf("%s：应当被拒绝", name)
		}
	}
}

func TestConfigExport_SkipsEncryptedAndRoundTrips(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, newTestAES())
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "plain.key", Value: "v1", ValueType: "string", Group: "g"})
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "secret.key", Value: "hidden", IsEncrypted: true})

	file, err := svc.ExportFile()
	if err != nil {
		t.Fatalf("ExportFile 失败: %v", err)
	}
	if len(file.Items) != 1 || file.Items[0].Key != "plain.key" {
		t.Fatalf("导出应只含非加密项，实际: %+v", file.Items)
	}

	raw, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	otherDB := setupTestDB(t)
	otherRepo := repository.NewConfigRepo(otherDB)
	other := NewConfigService(otherDB, otherRepo, nil, nil)
	result, err := other.ImportFile(context.Background(), raw)
	if err != nil {
		t.Fatalf("导入导出文件失败: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("结果 = %+v, want created=1", result)
	}
	if got := mustFindConfig(t, otherRepo, "plain.key"); got.Value != "v1" || got.Group != "g" {
		t.Errorf("往返后数据不一致: %+v", got)
	}
}

// 主题、站点信息由专用服务维护：导出要跳过，手工构造的文件也不能写进去（否则能写坏主题）。
func TestConfigTransfer_SkipsServiceOwnedGroups(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewConfigRepo(db)
	svc := NewConfigService(db, repo, nil, nil)
	mustCreateConfig(t, svc, ConfigCreateInput{Key: themeConfigKey, Value: `{"layout":"top"}`, ValueType: "json", Group: themeConfigGroup})
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "site.name", Value: "平台", Group: siteConfigGroup})
	mustCreateConfig(t, svc, ConfigCreateInput{Key: "plain.key", Value: "v1"})

	file, err := svc.ExportFile()
	if err != nil {
		t.Fatalf("ExportFile 失败: %v", err)
	}
	if len(file.Items) != 1 || file.Items[0].Key != "plain.key" {
		t.Fatalf("导出应跳过专用分组，实际: %+v", file.Items)
	}

	bad := map[string]string{
		"写进专用分组": `{"kind":"adminx.config","version":1,"items":[{"key":"theme.injected","value":"x","group":"theme"}]}`,
		"改写存量配置": `{"kind":"adminx.config","version":1,"items":[{"key":"site.name","value":"劫持","group":"default"}]}`,
	}
	for name, body := range bad {
		if _, err := svc.ImportFile(context.Background(), []byte(body)); err == nil {
			t.Errorf("%s：应当被拒绝", name)
		}
	}

	if got := mustFindConfig(t, repo, "site.name"); got.Value != "平台" || got.Group != siteConfigGroup {
		t.Errorf("专用分组的配置被改动了: %+v", got)
	}
}
