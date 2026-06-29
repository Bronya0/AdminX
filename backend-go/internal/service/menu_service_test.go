package service

import (
	"testing"

	"adminx/internal/repository"
)

func TestMenuService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, err := svc.Create(MenuCreateInput{
		Code:      "dashboard",
		Name:      "仪表盘",
		Path:      "/dashboard",
		Component: "DashboardView",
		MenuType:  "menu",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if menu.Name != "仪表盘" {
		t.Errorf("Name = %s", menu.Name)
	}
	if menu.ID == 0 {
		t.Error("ID 为 0")
	}
}

func TestMenuService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, _ := svc.Create(MenuCreateInput{
		Code: "old",
		Name: "旧名称",
		Path: "/old",
	})

	updated, err := svc.Update(menu.ID, MenuUpdateInput{
		Name: "新名称",
	})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("Name = %s", updated.Name)
	}
}

func TestMenuService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	menu, _ := svc.Create(MenuCreateInput{
		Code: "todelete",
		Name: "待删除",
		Path: "/todelete",
	})

	if err := svc.Delete(menu.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	_, err := svc.GetByID(menu.ID)
	if err == nil {
		t.Fatal("删除后应查不到菜单")
	}
}

func TestMenuService_All(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMenuService(db, repository.NewMenuRepo(db))

	seedMenu(t, db, "m1", "菜单1", "/m1")
	seedMenu(t, db, "m2", "菜单2", "/m2")

	menus, err := svc.All()
	if err != nil {
		t.Fatalf("All 失败: %v", err)
	}
	if len(menus) != 2 {
		t.Errorf("len = %d, want 2", len(menus))
	}
}
