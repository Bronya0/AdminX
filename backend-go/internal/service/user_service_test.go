package service

import (
	"errors"
	"testing"

	"adminx/internal/repository"
	apperr "adminx/pkg/errors"
)

func TestUserService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))
	role := seedRole(t, db, "admin", false)

	user, err := svc.Create(CreateInput{
		Username: "newuser",
		Password: "Pass123!",
		Email:    "new@test.com",
		Roles:    []string{role.ID},
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if user.ID == "" {
		t.Error("用户 ID 为空")
	}
	if user.Username != "newuser" {
		t.Errorf("Username = %s", user.Username)
	}
	if len(user.Roles) != 1 {
		t.Errorf("Roles count = %d, want 1", len(user.Roles))
	}
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	_, _ = createPlainUser(t, db, svc, "dup", "Pass123!")
	_, err := createPlainUser(t, db, svc, "dup", "Pass456!")
	if err == nil {
		t.Fatal("重复用户名应返回 error")
	}
}

func TestUserService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	created, _ := createPlainUser(t, db, svc, "findme", "Pass123!")

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID 失败: %v", err)
	}
	if got.Username != "findme" {
		t.Errorf("Username = %s", got.Username)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	_, err := svc.GetByID("non-existent-id")
	if err == nil {
		t.Fatal("不存在的用户应返回 error")
	}
}

func TestUserService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	created, _ := createPlainUser(t, db, svc, "updateme", "Pass123!")

	newEmail := "updated@test.com"
	updated, err := svc.Update(created.ID, UpdateInput{
		Email: &newEmail,
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("Email = %s, want %s", updated.Email, newEmail)
	}
}

func TestUserService_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	created, _ := createPlainUser(t, db, svc, "pwuser", "OldPass1!")

	updated, err := svc.Update(created.ID, UpdateInput{
		Password: "NewPass456!",
	})
	if err != nil {
		t.Fatalf("Update password 失败: %v", err)
	}
	if updated.Password == "" {
		t.Error("密码不应为空")
	}
}

func TestUserService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	created, _ := createPlainUser(t, db, svc, "deleteme", "Pass123!")

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 软删除后不应查到
	_, err := svc.GetByID(created.ID)
	if err == nil {
		t.Fatal("删除后应查不到用户")
	}
}

func TestUserService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	_, _ = createPlainUser(t, db, svc, "alice", "Pass123!")
	_, _ = createPlainUser(t, db, svc, "bob", "Pass456!")

	users, count, err := svc.List(0, 10, "", "", nil)
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(users) != 2 {
		t.Errorf("len = %d, want 2", len(users))
	}
}

func TestUserService_List_Search(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	aliceRole := seedRole(t, db, "role_alice", false)
	_, _ = svc.Create(CreateInput{Username: "alice", Password: "Pass123!", Email: "alice@test.com", Roles: []string{aliceRole.ID}})
	bobRole := seedRole(t, db, "role_bob", false)
	_, _ = svc.Create(CreateInput{Username: "bob", Password: "Pass456!", Email: "bob@test.com", Roles: []string{bobRole.ID}})

	_, count, err := svc.List(0, 10, "alice", "", nil)
	if err != nil {
		t.Fatalf("List search 失败: %v", err)
	}
	if count != 1 {
		t.Errorf("search 'alice' count = %d, want 1", count)
	}
}

// 最后一个超管不能被降级、停用或删除：一旦成功，后台再没人能管，只能去改库。
func TestUserService_LastSuperuserGuard(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	root, err := svc.Create(CreateInput{Username: "root", Password: "Pass123!", IsSuperuser: true})
	if err != nil {
		t.Fatalf("创建超管失败: %v", err)
	}

	assertForbidden := func(name string, err error) {
		t.Helper()
		var appErr *apperr.AppError
		if !errors.As(err, &appErr) || appErr.Code != 403 {
			t.Errorf("%s 应返回 403，实际 %v", name, err)
		}
	}

	_, err = svc.Update(root.ID, UpdateInput{IsSuperuser: boolPtr(false)})
	assertForbidden("唯一超管降级", err)
	_, err = svc.Update(root.ID, UpdateInput{IsActive: boolPtr(false)})
	assertForbidden("唯一超管停用", err)
	assertForbidden("唯一超管删除", svc.Delete(root.ID))

	// 补上第二个超管后，降级其中一个应被允许
	second, err := svc.Create(CreateInput{Username: "root2", Password: "Pass123!", IsSuperuser: true})
	if err != nil {
		t.Fatalf("创建第二个超管失败: %v", err)
	}
	if _, err := svc.Update(second.ID, UpdateInput{IsSuperuser: boolPtr(false)}); err != nil {
		t.Errorf("还有其它超管时降级应允许，实际 %v", err)
	}
}

// 超管的角色不参与授权：更新时静默忽略 roles，避免前端提交整个表单被拒。
func TestUserService_Update_IgnoresSuperuserRoles(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	root, err := svc.Create(CreateInput{Username: "root", Password: "Pass123!", IsSuperuser: true})
	if err != nil {
		t.Fatalf("创建超管失败: %v", err)
	}
	role := seedRole(t, db, "ops", false)

	updated, err := svc.Update(root.ID, UpdateInput{Roles: []string{role.ID}})
	if err != nil {
		t.Fatalf("更新超管失败: %v", err)
	}
	if len(updated.Roles) != 0 {
		t.Errorf("超管角色变更应被忽略，实际 %v", updated.Roles)
	}
}

// 强制下线：写入 last_logout，使该用户已签发的 token 失效。
func TestUserService_Logout(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	u, err := createPlainUser(t, db, svc, "kickme", "Pass123!")
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if before, _ := svc.GetByID(u.ID); before.LastLogout != nil {
		t.Fatalf("新用户不该有 last_logout")
	}

	if err := svc.Logout(u.ID); err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}
	after, err := svc.GetByID(u.ID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if after.LastLogout == nil {
		t.Error("下线后应写入 last_logout，否则旧 token 不会被吊销")
	}
}
