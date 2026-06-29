package service

import (
	"context"
	"encoding/base64"
	"testing"

	"gorm.io/gorm"

	"adminx/pkg/crypto"

	"adminx/internal/model"
	"adminx/internal/repository"
)

func newTestAES() *crypto.AESGCM {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	aes, _ := crypto.NewAESGCM(base64.StdEncoding.EncodeToString(key))
	return aes
}

func TestConfigService_Create(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, err := svc.Create(ConfigCreateInput{
		Key:       "SITE_NAME",
		Value:     "",
		ValueType: "string",
		Group:     "system",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if cfg.Key != "SITE_NAME" {
		t.Errorf("Key = %s", cfg.Key)
	}
	if cfg.Group != "system" {
		t.Errorf("Group = %s", cfg.Group)
	}
	if cfg.ID == 0 {
		t.Error("ID 为 0")
	}
}

func TestConfigService_Create_DefaultGroup(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, err := svc.Create(ConfigCreateInput{
		Key:   "TEST_CONFIG",
		Value: "test",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if cfg.Group != "default" {
		t.Errorf("默认 group = %s, want default", cfg.Group)
	}
}

func TestConfigService_Create_Encrypted(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, err := svc.Create(ConfigCreateInput{
		Key:         "SECRET_KEY",
		Value:       "my-secret-value",
		IsEncrypted: true,
	})
	if err != nil {
		t.Fatalf("Create encrypted 失败: %v", err)
	}
	if !cfg.IsEncrypted {
		t.Error("IsEncrypted 应为 true")
	}
	if cfg.Value == "my-secret-value" {
		t.Error("加密后 value 不应是明文")
	}
}

func TestConfigService_Create_Encrypted_NoAES(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)

	_, err := svc.Create(ConfigCreateInput{
		Key:         "SECRET",
		Value:       "secret",
		IsEncrypted: true,
	})
	if err == nil {
		t.Fatal("无 AES 密钥时加密创建应返回 error")
	}
}

func TestConfigService_GetValue(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	svc.Create(ConfigCreateInput{Key: "MAX_CONNECTIONS", Value: "100", ValueType: "int"})

	val, err := svc.GetValue(context.Background(), "MAX_CONNECTIONS", 50)
	if err != nil {
		t.Fatalf("GetValue 失败: %v", err)
	}
	if num, ok := val.(int); !ok || num != 100 {
		t.Errorf("GetValue = %v, want 100", val)
	}
}

func TestConfigService_GetValue_Default(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)

	val, err := svc.GetValue(context.Background(), "NON_EXIST", "fallback")
	if err != nil {
		t.Fatalf("GetValue 失败: %v", err)
	}
	if val != "fallback" {
		t.Errorf("GetValue = %v, want fallback", val)
	}
}

func TestConfigService_GetByGroup(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	svc.Create(ConfigCreateInput{Key: "OPT_A", Value: "1", Group: "testing"})
	svc.Create(ConfigCreateInput{Key: "OPT_B", Value: "2", Group: "testing"})

	result, err := svc.GetByGroup(context.Background(), "testing")
	if err != nil {
		t.Fatalf("GetByGroup 失败: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("result count = %d, want 2", len(result))
	}
}

func TestConfigService_Update(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, _ := svc.Create(ConfigCreateInput{Key: "UPDATE_ME", Value: "old", Group: "system"})

	updated, err := svc.Update(cfg.ID, ConfigUpdateInput{Value: "new", Group: "updated_group"})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Value != "new" {
		t.Errorf("Value = %s", updated.Value)
	}
	if updated.Group != "updated_group" {
		t.Errorf("Group = %s", updated.Group)
	}
}

func TestConfigService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, nil)

	_, err := svc.Update(99999, ConfigUpdateInput{Value: "x"})
	if err == nil {
		t.Fatal("更新不存在的配置应返回 error")
	}
}

func TestConfigService_Delete(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, _ := svc.Create(ConfigCreateInput{Key: "DELETE_ME", Value: "bye"})

	if err := svc.Delete(cfg.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	_, err := svc.GetValue(context.Background(), "DELETE_ME", nil)
	if err != nil {
		t.Fatalf("GetValue 不应报错（返回默认值）: %v", err)
	}
}

func TestConfigService_ParseValue(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := &ConfigService{db: db, aes: aes}

	t.Run("int", func(t *testing.T) {
		c := &model.Config{Value: "42", ValueType: "int"}
		v := svc.parseValue(c)
		if v != 42 {
			t.Errorf("int parse = %v, want 42", v)
		}
	})
	t.Run("int empty", func(t *testing.T) {
		c := &model.Config{Value: "", ValueType: "int"}
		v := svc.parseValue(c)
		if v != 0 {
			t.Errorf("empty int = %v, want 0", v)
		}
	})
	t.Run("bool true", func(t *testing.T) {
		c := &model.Config{Value: "true", ValueType: "bool"}
		if svc.parseValue(c) != true {
			t.Error("true parse 应为 true")
		}
	})
	t.Run("bool 1", func(t *testing.T) {
		c := &model.Config{Value: "1", ValueType: "bool"}
		if svc.parseValue(c) != true {
			t.Error("'1' parse 应为 true")
		}
	})
	t.Run("bool false", func(t *testing.T) {
		c := &model.Config{Value: "false", ValueType: "bool"}
		if svc.parseValue(c) != false {
			t.Error("'false' parse 应为 false")
		}
	})
	t.Run("json", func(t *testing.T) {
		c := &model.Config{Value: `{"a":1}`, ValueType: "json"}
		v := svc.parseValue(c)
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatal("json parse 应为 map")
		}
		if m["a"].(float64) != 1 {
			t.Errorf("a = %v", m["a"])
		}
	})
	t.Run("options", func(t *testing.T) {
		c := &model.Config{Value: `[{"label":"A","value":"a"}]`, ValueType: "options"}
		v := svc.parseValue(c)
		arr, ok := v.([]interface{})
		if !ok || len(arr) != 1 {
			t.Fatalf("options parse 应为长度为1的数组: %v", v)
		}
	})
	t.Run("string", func(t *testing.T) {
		c := &model.Config{Value: "hello", ValueType: "string"}
		if svc.parseValue(c) != "hello" {
			t.Error("string parse 失败")
		}
	})
}

func TestConfigService_Encrypted_GetValue(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	_, err := svc.Create(ConfigCreateInput{
		Key:         "ENC_VAL",
		Value:       "plain-secret",
		IsEncrypted: true,
	})
	if err != nil {
		t.Fatalf("Create encrypted 失败: %v", err)
	}

	val, err := svc.GetValue(context.Background(), "ENC_VAL", nil)
	if err != nil {
		t.Fatalf("GetValue 失败: %v", err)
	}
	if val != "plain-secret" {
		t.Errorf("加密配置读取应解密为明文: got %v", val)
	}
}

func TestConfigService_List(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	svc.Create(ConfigCreateInput{Key: "A", Value: "1"})
	svc.Create(ConfigCreateInput{Key: "B", Value: "2", Group: "system"})

	dtos, count, err := svc.List(0, 10, "", "")
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(dtos) != 2 {
		t.Errorf("len = %d, want 2", len(dtos))
	}
}

func TestConfigService_Groups(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	svc.Create(ConfigCreateInput{Key: "A", Value: "1", Group: "g1"})
	svc.Create(ConfigCreateInput{Key: "B", Value: "2", Group: "g2"})

	groups, err := svc.Groups()
	if err != nil {
		t.Fatalf("Groups 失败: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("groups count = %d, want 2", len(groups))
	}
}

func TestConfigService_Enrypted_DisplayValue(t *testing.T) {
	db := setupTestDB(t)
	aes := newTestAES()
	svc := NewConfigService(db, repository.NewConfigRepo(db), nil, aes)

	cfg, _ := svc.Create(ConfigCreateInput{
		Key:         "DISPLAY_TEST",
		Value:       "secret",
		IsEncrypted: true,
	})

	dto, err := svc.GetByID(cfg.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if dto.Value != "" {
		t.Errorf("加密配置的 value 字段应为空, got %s", dto.Value)
	}
	if dto.DisplayValue != "********" {
		t.Errorf("DisplayValue = %s, want ********", dto.DisplayValue)
	}
}

// 确保 gorm import 被使用
var _ = gorm.ErrRecordNotFound
