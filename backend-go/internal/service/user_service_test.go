package service

import (
	"testing"

	"adminx/internal/repository"
)

func TestUserService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))
	role := seedRole(t, db, "admin", false)

	user, err := svc.Create(CreateInput{
		Username: "newuser",
		Password: "pass123",
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

	svc.Create(CreateInput{Username: "dup", Password: "pass123"})
	_, err := svc.Create(CreateInput{Username: "dup", Password: "pass456"})
	if err == nil {
		t.Fatal("重复用户名应返回 error")
	}
}

func TestUserService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, repository.NewUserRepo(db))

	created, _ := svc.Create(CreateInput{Username: "findme", Password: "pass123"})

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

	created, _ := svc.Create(CreateInput{Username: "updateme", Password: "pass123"})

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

	created, _ := svc.Create(CreateInput{Username: "pwuser", Password: "oldpass"})

	updated, err := svc.Update(created.ID, UpdateInput{
		Password: "newpass456",
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

	created, _ := svc.Create(CreateInput{Username: "deleteme", Password: "pass123"})

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

	svc.Create(CreateInput{Username: "alice", Password: "pass123"})
	svc.Create(CreateInput{Username: "bob", Password: "pass456"})

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

	svc.Create(CreateInput{Username: "alice", Password: "pass123", Email: "alice@test.com"})
	svc.Create(CreateInput{Username: "bob", Password: "pass456", Email: "bob@test.com"})

	_, count, err := svc.List(0, 10, "alice", "", nil)
	if err != nil {
		t.Fatalf("List search 失败: %v", err)
	}
	if count != 1 {
		t.Errorf("search 'alice' count = %d, want 1", count)
	}
}
