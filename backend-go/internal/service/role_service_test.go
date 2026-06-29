package service

import (
	"testing"

	"djangoadminx/internal/repository"
)

func TestRoleService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))
	menu := seedMenu(t, db, "dashboard", "仪表盘", "/dashboard")

	active := true
	role, err := svc.Create(RoleCreateInput{
		Name:     "test-role",
		Desc:     "测试角色",
		IsActive: &active,
		MenuIDs:  []int64{menu.ID},
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if role.Name != "test-role" {
		t.Errorf("Name = %s", role.Name)
	}
}

func TestRoleService_Create_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))

	svc.Create(RoleCreateInput{Name: "dup-role"})
	_, err := svc.Create(RoleCreateInput{Name: "dup-role"})
	if err == nil {
		t.Fatal("重复角色名应返回 error")
	}
}

func TestRoleService_Update_SystemRole(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))

	role := seedRole(t, db, "system-role", true)

	_, err := svc.Update(role.ID, RoleUpdateInput{Name: "new-name"})
	if err == nil {
		t.Fatal("修改系统角色名称应返回 error")
	}
}

func TestRoleService_Delete_SystemRole(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))

	role := seedRole(t, db, "system-role", true)

	err := svc.Delete(role.ID)
	if err == nil {
		t.Fatal("删除系统角色应返回 error")
	}
}

func TestRoleService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))

	seedRole(t, db, "admin", false)
	seedRole(t, db, "user", false)

	roles, count, err := svc.List(0, 10, "")
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(roles) != 2 {
		t.Errorf("len = %d, want 2", len(roles))
	}
}
