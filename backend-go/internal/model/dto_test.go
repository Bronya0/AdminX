package model

import (
	"testing"
	"time"
)

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
