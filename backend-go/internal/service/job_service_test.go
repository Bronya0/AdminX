package service

import (
	"context"
	"log/slog"
	"testing"

	"djangoadminx/internal/repository"
)

func TestJobService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	isActive := true
	job, err := svc.Create(JobCreateInput{
		Name:        "测试任务",
		CommandType: "python",
		Handler:     "jobs.SampleTask",
		TriggerType: "interval",
		IsActive:    &isActive,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if job.Name != "测试任务" {
		t.Errorf("Name = %s", job.Name)
	}
	if job.ID == "" {
		t.Error("ID 为空")
	}
	if job.Kwargs != "{}" {
		t.Errorf("默认 kwargs = %s, want {}", job.Kwargs)
	}
}

func TestJobService_Create_DefaultCommandType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, err := svc.Create(JobCreateInput{
		Name: "默认类型任务",
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if job.CommandType != "python" {
		t.Errorf("CommandType = %s, want python", job.CommandType)
	}
	if job.TriggerType != "interval" {
		t.Errorf("TriggerType = %s, want interval", job.TriggerType)
	}
}

func TestJobService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{Name: "old", Handler: "jobs.SampleTask"})

	updated, err := svc.Update(job.ID, JobUpdateInput{Name: "new"})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("Name = %s", updated.Name)
	}
}

func TestJobService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	_, err := svc.Update("non-existent", JobUpdateInput{Name: "x"})
	if err == nil {
		t.Fatal("更新不存在的任务应返回 error")
	}
}

func TestJobService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{Name: "to-delete"})

	if err := svc.Delete(job.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
}

func TestJobService_RunOnce(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{
		Name:    "sample-job",
		Handler: "jobs.SampleTask",
	})

	result, err := svc.RunOnce(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("RunOnce 失败: %v", err)
	}
	if result == "" {
		t.Error("执行结果为空")
	}
}

func TestJobService_RunOnce_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	_, err := svc.RunOnce(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("执行不存在的任务应返回 error")
	}
}

func TestJobService_RunOnce_EmptyHandler(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{Name: "no-handler"})

	_, err := svc.RunOnce(context.Background(), job.ID)
	if err == nil {
		t.Fatal("无 handler 的任务应返回 error")
	}
}

func TestJobService_RunOnce_UnregisteredHandler(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{
		Name:    "bad-handler",
		Handler: "jobs.NotRegistered",
	})

	_, err := svc.RunOnce(context.Background(), job.ID)
	if err == nil {
		t.Fatal("未注册的 handler 应返回 error")
	}
}

func TestJobService_RunOnce_ShellBlockedChars(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{
		Name:        "bad-cmd",
		CommandType: "shell",
		Command:     "echo hello; rm -rf /",
	})

	result, err := svc.RunOnce(context.Background(), job.ID)
	if err == nil {
		// 可能 echo 没被阻止，但 ; rm -rf 应被阻止
		t.Logf("result: %s", result)
	}
	// 预期应因为含 `;` 被拒绝
	if err == nil {
		t.Fatal("含禁用字符的 shell 命令应返回 error")
	}
}

func TestJobService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	svc.Create(JobCreateInput{Name: "任务A"})
	svc.Create(JobCreateInput{Name: "任务B"})

	jobs, count, err := svc.List(0, 10, "")
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(jobs) != 2 {
		t.Errorf("len = %d, want 2", len(jobs))
	}
}

func TestJobService_ListLogs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	logs, count, err := svc.ListLogs(0, 10, "", "")
	if err != nil {
		t.Fatalf("ListLogs 失败: %v", err)
	}
	if count != 0 {
		t.Logf("count = %d", count)
	}
	if logs == nil {
		t.Error("logs 不应为 nil")
	}
}
