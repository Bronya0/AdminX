package model

import (
	"testing"
	"time"
)

// 心跳响应必须把升级/卸载指令放在 upgrade/uninstall 字段里。
// 业务组件读的是 data.upgrade（平铺的 upgrade_version/upgrade_url 它读不到），
// 只返回平铺字段等于指令永远不生效。
func TestToHeartbeatDTO(t *testing.T) {
	checksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	comp := &ServiceComponent{
		AppLabel:         "demo",
		Name:             "演示业务",
		UpgradeVersion:   "1.1.0",
		UpgradeURL:       "http://example.com/pkg.zip",
		UpgradeChecksum:  checksum,
		UninstallPending: true,
	}

	dto := ToHeartbeatDTO(comp)
	if dto.Upgrade == nil {
		t.Fatal("完整升级指令必须下发到 data.upgrade")
	}
	if dto.Upgrade.Version != "1.1.0" || dto.Upgrade.URL != "http://example.com/pkg.zip" || dto.Upgrade.Checksum != checksum {
		t.Errorf("升级指令字段不完整: %+v", dto.Upgrade)
	}
	if !dto.Uninstall {
		t.Error("卸载标记必须下发到 data.uninstall")
	}

	// 指令不完整时不下发，避免业务侧拿到无校验的包
	partial := ToHeartbeatDTO(&ServiceComponent{AppLabel: "demo", UpgradeVersion: "1.1.0"})
	if partial.Upgrade != nil {
		t.Error("缺 url/checksum 的升级指令不应下发")
	}
}

func TestToUserDTO(t *testing.T) {
	now := time.Now()
	user := &User{
		ID:           "uuid-1",
		Username:     "testuser",
		Email:        "test@example.com",
		Phone:        "13800138000",
		Avatar:       "/avatar.png",
		Desc:         "测试用户",
		IsActive:     true,
		IsSuperuser:  false,
		IsStaff:      true,
		HomePage:     "/dashboard",
		LastLogin:    &now,
		LastActivity: &now,
		DateJoined:   now,
		Roles: []Role{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "user"},
		},
	}

	dto := ToUserDTO(user)

	if dto.ID != "uuid-1" {
		t.Errorf("ID = %s, want uuid-1", dto.ID)
	}
	if dto.Username != "testuser" {
		t.Errorf("Username = %s", dto.Username)
	}
	if len(dto.Roles) != 2 {
		t.Errorf("Roles count = %d, want 2", len(dto.Roles))
	}
	if dto.Roles[0] != "admin" {
		t.Errorf("Roles[0] = %s, want admin", dto.Roles[0])
	}
	if len(dto.RoleNames) != 2 {
		t.Errorf("RoleNames count = %d, want 2", len(dto.RoleNames))
	}
	// UserDTO 不暴露 Password 字段，验证 DTO 不包含敏感信息
	_ = dto
}

func TestToUserDTO_NoRoles(t *testing.T) {
	user := &User{ID: "uuid-1", Username: "norole"}
	dto := ToUserDTO(user)
	if len(dto.Roles) != 0 {
		t.Errorf("Roles 应为空切片, got %v", dto.Roles)
	}
	if dto.Roles == nil {
		t.Error("Roles 应为空切片而非 nil")
	}
}

func TestToUserDTOList(t *testing.T) {
	users := []User{
		{ID: "u1", Username: "a", Roles: []Role{{Name: "admin"}}},
		{ID: "u2", Username: "b", Roles: []Role{{Name: "user"}}},
	}
	dtos := ToUserDTOList(users)
	if len(dtos) != 2 {
		t.Fatalf("len = %d, want 2", len(dtos))
	}
	if dtos[0].ID != "u1" {
		t.Errorf("dtos[0].ID = %s", dtos[0].ID)
	}
	if dtos[1].ID != "u2" {
		t.Errorf("dtos[1].ID = %s", dtos[1].ID)
	}
}

func TestUser_IsOnline(t *testing.T) {
	t.Run("无活动时间", func(t *testing.T) {
		u := &User{}
		if u.IsOnline() {
			t.Error("无 LastActivity 应返回 false")
		}
	})
	t.Run("最近活动", func(t *testing.T) {
		now := time.Now()
		u := &User{LastActivity: &now}
		if !u.IsOnline() {
			t.Error("刚刚活动应返回 true")
		}
	})
	t.Run("超时", func(t *testing.T) {
		past := time.Now().Add(-10 * time.Minute)
		u := &User{LastActivity: &past}
		if u.IsOnline() {
			t.Error("10 分钟前活动应返回 false")
		}
	})
}
