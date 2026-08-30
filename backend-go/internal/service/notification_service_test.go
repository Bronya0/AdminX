package service

import (
	"context"
	"log/slog"
	"testing"

	"adminx/internal/repository"
)

func TestNotificationService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	n, err := svc.Create(context.Background(), NotificationCreateInput{
		Title:   "测试通知",
		Content: "这是一条测试通知",
		Type:    "info",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if n.Title != "测试通知" {
		t.Errorf("Title = %s", n.Title)
	}
	if n.NotificationType != "info" {
		t.Errorf("NotificationType = %s", n.NotificationType)
	}
}

func TestNotificationService_Create_DefaultType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	n, err := svc.Create(context.Background(), NotificationCreateInput{
		Title: "无类型通知",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if n.NotificationType != "info" {
		t.Errorf("默认 type = %s, want info", n.NotificationType)
	}
}

func TestNotificationService_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	// 广播通知
	svc.Create(context.Background(), NotificationCreateInput{
		Title: "系统公告",
		Type:  "warning",
	})
	// 用户通知
	user := seedUser(t, db, "notifyuser", "pass123")
	svc.Create(context.Background(), NotificationCreateInput{
		Title:  "个人消息",
		UserID: user.ID,
	})

	notifs, count, err := svc.ListByUser(0, 10, user.ID, false)
	if err != nil {
		t.Fatalf("ListByUser 失败: %v", err)
	}
	if count < 2 {
		t.Errorf("count = %d, want >= 2", count)
	}
	if len(notifs) < 2 {
		t.Errorf("len = %d, want >= 2", len(notifs))
	}
}

func TestNotificationService_UnreadCount(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	user := seedUser(t, db, "countuser", "pass123")
	svc.Create(context.Background(), NotificationCreateInput{
		Title:  "未读消息",
		UserID: user.ID,
	})

	count, err := svc.UnreadCount(user.ID)
	if err != nil {
		t.Fatalf("UnreadCount 失败: %v", err)
	}
	if count < 1 {
		t.Errorf("count = %d, want >= 1", count)
	}
}

func TestNotificationService_MarkRead(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	user := seedUser(t, db, "readuser", "pass123")
	n, _ := svc.Create(context.Background(), NotificationCreateInput{
		Title:  "可标记已读",
		UserID: user.ID,
	})

	if err := svc.MarkRead(n.ID, user.ID); err != nil {
		t.Fatalf("MarkRead 失败: %v", err)
	}
}

func TestNotificationService_MarkRead_NotOwned(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	owner := seedUser(t, db, "owner", "pass123")
	other := seedUser(t, db, "other", "pass456")

	n, _ := svc.Create(context.Background(), NotificationCreateInput{
		Title:  "所有者的通知",
		UserID: owner.ID,
	})

	err := svc.MarkRead(n.ID, other.ID)
	if err == nil {
		t.Fatal("标记他人通知应返回 error")
	}
}

func TestNotificationService_MarkAllRead(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	user := seedUser(t, db, "alluser", "pass123")
	svc.Create(context.Background(), NotificationCreateInput{Title: "m1", UserID: user.ID})
	svc.Create(context.Background(), NotificationCreateInput{Title: "m2", UserID: user.ID})

	if err := svc.MarkAllRead(user.ID); err != nil {
		t.Fatalf("MarkAllRead 失败: %v", err)
	}
}

func TestNotificationService_WebhookCRUD(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNotificationService(db, repository.NewNotificationRepo(db), slog.Default())

	w, err := svc.CreateWebhook(
		WebhookInput{Name: "test-webhook", URL: "http://example.com/hook", IsActive: boolPtr(true)},
	)
	if err != nil {
		t.Fatalf("CreateWebhook 失败: %v", err)
	}
	if w.Name != "test-webhook" {
		t.Errorf("Name = %s", w.Name)
	}

	// list
	webhooks, count, err := svc.ListWebhooks(0, 10)
	if err != nil {
		t.Fatalf("ListWebhooks 失败: %v", err)
	}
	if count != 1 {
		t.Errorf("webhooks count = %d, want 1", count)
	}
	if len(webhooks) != 1 {
		t.Errorf("webhooks len = %d", len(webhooks))
	}

	// delete
	if err := svc.DeleteWebhook(w.ID); err != nil {
		t.Fatalf("DeleteWebhook 失败: %v", err)
	}
}

func TestMatchEvent(t *testing.T) {
	tests := []struct {
		eventsCSV string
		eventType string
		want      bool
	}{
		{"", "info", true},
		{"", "warning", true},
		{"info,warning,error", "info", true},
		{"info,warning,error", "warning", true},
		{"info,warning,error", "success", false},
		{" info , warning ", "info", true}, // 前后空格
	}
	for _, tt := range tests {
		got := matchEvent(tt.eventsCSV, tt.eventType)
		if got != tt.want {
			t.Errorf("matchEvent(%q, %q) = %v, want %v", tt.eventsCSV, tt.eventType, got, tt.want)
		}
	}
}
