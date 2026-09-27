package service

import (
	"context"
	"log/slog"
	"testing"

	"adminx/internal/repository"
)

func TestJobService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	isActive := true
	job, err := svc.Create(JobCreateInput{
		Name:          "测试任务",
		CommandType:   "python",
		Handler:       "jobs.SampleTask",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
		IsActive:      &isActive,
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
		Name:          "默认类型任务",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
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

	job, _ := svc.Create(JobCreateInput{Name: "old", Handler: "jobs.SampleTask", TriggerConfig: `{"minutes":5}`})

	name := "new"
	updated, err := svc.Update(job.ID, JobUpdateInput{Name: &name})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Name != "new" {
		t.Errorf("Name = %s", updated.Name)
	}
}

// PATCH 语义：只传 is_active（前端启用/禁用开关）时不得清空任务配置。
// 回归背景：UpdateInput 曾用值类型，零值覆盖会把 handler/command/trigger_config
// 一并写空，任务就地停跑且配置不可恢复。
func TestJobService_Update_PartialKeepsConfig(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, err := svc.Create(JobCreateInput{
		Name:          "keep-config",
		CommandType:   "shell",
		Command:       "echo hi",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
		Args:          `["a"]`,
	})
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}

	inactive := false
	updated, err := svc.Update(job.ID, JobUpdateInput{IsActive: &inactive})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Command != "echo hi" || updated.TriggerConfig != `{"minutes":5}` || updated.Args != `["a"]` {
		t.Fatalf("只传 is_active 不应清空任务配置: command=%q trigger_config=%q args=%q",
			updated.Command, updated.TriggerConfig, updated.Args)
	}
	if updated.IsActive {
		t.Error("IsActive 应已更新为 false")
	}
}

func TestJobService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	_, err := svc.Update("non-existent", JobUpdateInput{Name: strPtr("x")})
	if err == nil {
		t.Fatal("更新不存在的任务应返回 error")
	}
}

// strPtr 测试用字符串指针。
func strPtr(s string) *string { return &s }

func TestJobService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{Name: "to-delete", TriggerType: "interval", TriggerConfig: `{"minutes":5}`})

	if err := svc.Delete(job.ID); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
}

func TestJobService_RunOnce(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{
		Name:          "sample-job",
		Handler:       "jobs.SampleTask",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
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

	job, _ := svc.Create(JobCreateInput{Name: "no-handler", TriggerType: "interval", TriggerConfig: `{"minutes":5}`})

	_, err := svc.RunOnce(context.Background(), job.ID)
	if err == nil {
		t.Fatal("无 handler 的任务应返回 error")
	}
}

func TestJobService_RunOnce_UnregisteredHandler(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db, repository.NewJobRepo(db), slog.Default())

	job, _ := svc.Create(JobCreateInput{
		Name:          "bad-handler",
		Handler:       "jobs.NotRegistered",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
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
		Name:          "bad-cmd",
		CommandType:   "shell",
		Command:       "echo hello; rm -rf /",
		TriggerType:   "interval",
		TriggerConfig: `{"minutes":5}`,
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

	svc.Create(JobCreateInput{Name: "任务A", TriggerType: "interval", TriggerConfig: `{"minutes":5}`})
	svc.Create(JobCreateInput{Name: "任务B", TriggerType: "interval", TriggerConfig: `{"minutes":5}`})

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
