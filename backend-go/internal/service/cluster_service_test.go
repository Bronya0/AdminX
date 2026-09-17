package service

import (
	"testing"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// seedNode 造一个集群节点。
func seedNode(t *testing.T, svc *ClusterService, name, host, status string) {
	t.Helper()
	if _, err := svc.CreateNodeInput(NodeCreateInput{Name: name, Host: host, Status: status}); err != nil {
		t.Fatalf("创建节点 %s 失败: %v", name, err)
	}
}

// TestClusterService_NodesOverview 节点概览统计与列表过滤。
func TestClusterService_NodesOverview(t *testing.T) {
	db := setupTestDB(t)
	svc := NewClusterService(db, repository.NewClusterRepo(db))

	seedNode(t, svc, "node-1", "10.0.0.1", "online")
	seedNode(t, svc, "node-2", "10.0.0.2", "online")
	seedNode(t, svc, "node-3", "10.0.0.3", "offline")

	overview, err := svc.NodesOverview()
	if err != nil {
		t.Fatalf("NodesOverview 失败: %v", err)
	}
	if overview.Total != 3 || overview.Online != 2 || overview.Offline != 1 {
		t.Errorf("统计不对: %+v", overview)
	}
	if len(overview.Nodes) != 3 {
		t.Errorf("概览应带全部节点，实际 %d", len(overview.Nodes))
	}

	// status 过滤
	if _, count, err := svc.ListNodes(0, 10, "", "online"); err != nil || count != 2 {
		t.Errorf("status=online 应命中 2 条: count=%d err=%v", count, err)
	}
	// search 过滤（按 name/host）
	if _, count, err := svc.ListNodes(0, 10, "node-1", ""); err != nil || count != 1 {
		t.Errorf("search=node-1 应命中 1 条: count=%d err=%v", count, err)
	}
	if _, count, err := svc.ListNodes(0, 10, "10.0.0.3", ""); err != nil || count != 1 {
		t.Errorf("按 host 搜索应命中 1 条: count=%d err=%v", count, err)
	}

	// 单节点查询与不存在
	node, err := svc.GetNode(overview.Nodes[0].ID)
	if err != nil || node.Name == "" {
		t.Errorf("GetNode 失败: %v", err)
	}
	if _, err := svc.GetNode("00000000-0000-0000-0000-0000000000ff"); err == nil {
		t.Error("不存在的节点应返回 404 错误")
	}
}

// TestClusterService_ComponentLifecycle 组件查询/删除/确认升级。
func TestClusterService_ComponentLifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewClusterService(db, repository.NewClusterRepo(db))

	component, err := svc.Register(RegisterInput{AppLabel: "demo-app", Name: "演示业务", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("注册组件失败: %v", err)
	}

	got, err := svc.GetComponent(component.ID)
	if err != nil || got.AppLabel != "demo-app" {
		t.Fatalf("GetComponent 失败: %+v err=%v", got, err)
	}
	if got.Status() != "online" {
		t.Errorf("刚注册的组件应在线，实际 %s", got.Status())
	}

	// 下发升级指令 → 确认升级后清空
	if err := svc.SetUpgrade(component.ID, "1.1.0", "http://example.com/pkg.zip", "sha256:abc"); err != nil {
		t.Fatalf("SetUpgrade 失败: %v", err)
	}
	if err := svc.ConfirmUpgrade(component.ID); err != nil {
		t.Fatalf("ConfirmUpgrade 失败: %v", err)
	}
	after, err := svc.GetComponent(component.ID)
	if err != nil {
		t.Fatalf("确认升级后查询失败: %v", err)
	}
	if after.UpgradeVersion != "" || after.UpgradeURL != "" {
		t.Errorf("确认升级应清空指令，实际 version=%s url=%s", after.UpgradeVersion, after.UpgradeURL)
	}

	// 删除组件；重复删除应报 404
	if err := svc.DeleteComponent(component.ID); err != nil {
		t.Fatalf("DeleteComponent 失败: %v", err)
	}
	if err := svc.DeleteComponent(component.ID); err == nil {
		t.Error("删除不存在的组件应报错")
	}

	var count int64
	if err := db.Model(&model.ServiceComponent{}).Count(&count).Error; err != nil {
		t.Fatalf("统计组件失败: %v", err)
	}
	if count != 0 {
		t.Errorf("组件应已删除，剩余 %d 条", count)
	}
}
