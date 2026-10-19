package service

import (
	"errors"
	"fmt"
	"testing"

	"adminx/internal/model"
	"adminx/internal/repository"
	apperr "adminx/pkg/errors"
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

func TestRoleService_InvalidMenuIDsRollback(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))
	menu := seedMenu(t, db, "valid", "有效菜单", "/valid")
	invalidIDs := []int64{menu.ID, menu.ID + 1000}
	if _, err := svc.Create(RoleCreateInput{Name: "invalid", MenuIDs: invalidIDs}); !errors.Is(err, apperr.ErrBadRequest) {
		t.Fatalf("Create 应拒绝部分不存在的菜单，实际为 %v", err)
	}
	var count int64
	if err := db.Model(&model.Role{}).Where("name = ?", "invalid").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("创建失败不应留下角色: count=%d, err=%v", count, err)
	}
	role, err := svc.Create(RoleCreateInput{Name: "original", MenuIDs: []int64{menu.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(role.ID, RoleUpdateInput{Name: "changed", MenuIDs: invalidIDs}); !errors.Is(err, apperr.ErrBadRequest) {
		t.Fatalf("Update 应拒绝部分不存在的菜单，实际为 %v", err)
	}
	stored, err := svc.GetByID(role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "original" || len(stored.Menus) != 1 || stored.Menus[0].ID != menu.ID {
		t.Fatalf("授权校验失败应回滚资料且保留原授权: %+v", stored)
	}
}

func TestRoleService_UpdateRollsBackOnAssociationWriteFailure(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))
	oldMenu := seedMenu(t, db, "old", "原菜单", "/old")
	newMenu := seedMenu(t, db, "new", "新菜单", "/new")
	role, err := svc.Create(RoleCreateInput{Name: "original", MenuIDs: []int64{oldMenu.ID}})
	if err != nil {
		t.Fatal(err)
	}
	// 菜单均存在，但实际关联写入失败：验证事务而非仅验证参数校验。
	if err := db.Exec(fmt.Sprintf(`CREATE TRIGGER fail_role_menu_insert BEFORE INSERT ON role_menus
		WHEN NEW.menu_id = %d BEGIN SELECT RAISE(ABORT, 'injected association failure'); END`, newMenu.ID)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(role.ID, RoleUpdateInput{Name: "changed", MenuIDs: []int64{newMenu.ID}}); !errors.Is(err, apperr.ErrInternal) {
		t.Fatalf("预期关联写入失败，实际为 %v", err)
	}
	stored, err := svc.GetByID(role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "original" || len(stored.Menus) != 1 || stored.Menus[0].ID != oldMenu.ID {
		t.Fatalf("关联失败后必须完整回滚: %+v", stored)
	}
}

func TestRoleService_MenuAssignmentSetSemantics(t *testing.T) {
	db := setupTestDB(t)
	svc := NewRoleService(db, repository.NewRoleRepo(db))
	menu := seedMenu(t, db, "menu", "菜单", "/menu")
	role, err := svc.Create(RoleCreateInput{Name: "role", MenuIDs: []int64{menu.ID, menu.ID}})
	if err != nil || len(role.Menus) != 1 {
		t.Fatalf("重复菜单 ID 应按集合处理: role=%+v, err=%v", role, err)
	}
	updated, err := svc.Update(role.ID, RoleUpdateInput{Name: "renamed"})
	if err != nil || updated.Name != "renamed" || len(updated.Menus) != 1 {
		t.Fatalf("未传 menu_ids 应保留授权: role=%+v, err=%v", updated, err)
	}
	updated, err = svc.Update(role.ID, RoleUpdateInput{MenuIDs: []int64{menu.ID, menu.ID}})
	if err != nil || len(updated.Menus) != 1 {
		t.Fatalf("更新时重复菜单 ID 应按集合处理: role=%+v, err=%v", updated, err)
	}
	updated, err = svc.Update(role.ID, RoleUpdateInput{MenuIDs: []int64{}})
	if err != nil || len(updated.Menus) != 0 {
		t.Fatalf("显式空数组应清空授权: role=%+v, err=%v", updated, err)
	}
}
