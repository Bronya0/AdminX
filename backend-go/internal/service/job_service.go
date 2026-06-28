package service

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"

	"djangoadminx/internal/model"
	"djangoadminx/internal/repository"
)

var _ = time.Second // 保留 time 引用（executeShell 超时用到）

// JobService 定时任务业务逻辑。
type JobService struct {
	db   *gorm.DB
	repo *repository.JobRepo
	log  *slog.Logger
}

func NewJobService(db *gorm.DB, repo *repository.JobRepo, logger *slog.Logger) *JobService {
	return &JobService{db: db, repo: repo, log: logger}
}

func (s *JobService) List(offset, limit int, search string) ([]model.ScheduleJob, int64, error) {
	return s.repo.List(offset, limit, search)
}

func (s *JobService) ListLogs(offset, limit int, jobID, status string) ([]model.JobLog, int64, error) {
	return s.repo.ListLogs(offset, limit, jobID, status)
}

func (s *JobService) GetByID(id string) (*model.ScheduleJob, error) {
	j, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	return j, nil
}

// CreateInput 创建任务入参。
type JobCreateInput struct {
	Name          string `json:"name" binding:"required"`
	CommandType   string `json:"command_type"` // python/shell
	Handler       string `json:"handler"`
	Command       string `json:"command"`
	TriggerType   string `json:"trigger_type"` // cron/interval/date
	TriggerConfig string `json:"trigger_config"`
	Args          string `json:"args"`
	Kwargs        string `json:"kwargs"`
	IsActive      *bool  `json:"is_active"`
}

func (s *JobService) Create(in JobCreateInput) (*model.ScheduleJob, error) {
	ct := in.CommandType
	if ct == "" {
		ct = "python"
	}
	tt := in.TriggerType
	if tt == "" {
		tt = "interval"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	kwargs := in.Kwargs
	if kwargs == "" {
		kwargs = "{}"
	}
	job := &model.ScheduleJob{
		Name:          in.Name,
		CommandType:   ct,
		Handler:       in.Handler,
		Command:       in.Command,
		TriggerType:   tt,
		TriggerConfig: in.TriggerConfig,
		Args:          in.Args,
		Kwargs:        kwargs,
		IsActive:      isActive,
	}
	if err := s.repo.Create(job); err != nil {
		return nil, apperr.ErrInternal
	}
	return job, nil
}

// UpdateInput 更新任务入参。
type JobUpdateInput struct {
	Name          string `json:"name"`
	CommandType   string `json:"command_type"`
	Handler       string `json:"handler"`
	Command       string `json:"command"`
	TriggerType   string `json:"trigger_type"`
	TriggerConfig string `json:"trigger_config"`
	Args          string `json:"args"`
	Kwargs        string `json:"kwargs"`
	IsActive      *bool  `json:"is_active"`
}

func (s *JobService) Update(id string, in JobUpdateInput) (*model.ScheduleJob, error) {
	job, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	if in.Name != "" {
		job.Name = in.Name
	}
	if in.CommandType != "" {
		job.CommandType = in.CommandType
	}
	job.Handler = in.Handler
	job.Command = in.Command
	if in.TriggerType != "" {
		job.TriggerType = in.TriggerType
	}
	job.TriggerConfig = in.TriggerConfig
	job.Args = in.Args
	if in.Kwargs != "" {
		job.Kwargs = in.Kwargs
	}
	if in.IsActive != nil {
		job.IsActive = *in.IsActive
	}
	if err := s.repo.Update(job); err != nil {
		return nil, apperr.ErrInternal
	}
	return job, nil
}

func (s *JobService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return apperr.ErrInternal
	}
	return nil
}

// RunOnce 手动触发一次任务执行（不受调度器控制）。
func (s *JobService) RunOnce(ctx context.Context, id string) (string, error) {
	job, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", apperr.ErrNotFound
		}
		return "", apperr.ErrInternal
	}

	result, errMsg := s.executeJob(ctx, job)
	return result, errMsg
}

// ExecuteJob 调度器调用的执行入口（带 JobLog 记录）。
func (s *JobService) ExecuteJob(ctx context.Context, jobID string) {
	job, err := s.repo.FindByID(jobID)
	if err != nil {
		s.log.Warn("执行任务时找不到 job", "id", jobID, "error", err)
		return
	}

	now := time.Now()
	logEntry := &model.JobLog{
		JobID:     job.ID,
		Status:    "running",
		StartedAt: &now,
	}
	if err := s.repo.CreateLog(logEntry); err != nil {
		s.log.Error("写任务日志失败", "error", err)
	}

	result, execErr := s.executeJob(ctx, job)
	finished := time.Now()
	logEntry.FinishedAt = &finished

	if execErr != nil {
		logEntry.Status = "failed"
		logEntry.Result = truncateStr(execErr.Error(), 500)
		s.log.Error("任务执行失败", "job", job.Name, "error", execErr)
	} else {
		logEntry.Status = "success"
		logEntry.Result = truncateStr(result, 500)
	}
	_ = s.repo.UpdateLog(logEntry)
}

// executeJob 执行单个任务（python handler 反射 / shell 命令）。
func (s *JobService) executeJob(ctx context.Context, job *model.ScheduleJob) (string, error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("任务 panic", "job", job.Name, "panic", r, "stack", string(debug.Stack()))
		}
	}()

	if job.CommandType == "shell" {
		return s.executeShell(ctx, job)
	}
	return s.executePython(ctx, job)
}

// executePython 执行 Go 函数（通过 handler 路径反射查找已注册的 handler）。
// handler 格式: "jobs.SampleTask" → 在 HandlerRegistry 中查找。
func (s *JobService) executePython(ctx context.Context, job *model.ScheduleJob) (string, error) {
	if job.Handler == "" {
		return "", fmt.Errorf("handler 为空")
	}
	handler := GetHandler(job.Handler)
	if handler == nil {
		return "", fmt.Errorf("未注册的 handler: %s", job.Handler)
	}

	// 当前 handler 注册体系不支持参数传递，记录提示
	if job.Args != "" || job.Kwargs != "" {
		s.log.Warn("当前 handler 不支持参数传递，args/kwargs 被忽略",
			"job", job.Name, "handler", job.Handler)
	}

	// 反射调用前校验 handler 签名兼容性
	ht := reflect.TypeOf(handler)
	if ht.NumIn() != 0 || ht.NumOut() != 2 {
		return "", fmt.Errorf("handler %s 签名不兼容，期望 func() (string, error)，实际 %v", job.Handler, ht)
	}
	result := reflect.ValueOf(handler).Call([]reflect.Value{})
	if len(result) > 0 {
		if err, ok := result[0].Interface().(error); ok && err != nil {
			return "", err
		}
		str, ok := result[0].Interface().(string)
		if ok {
			return str, nil
		}
	}
	return "ok", nil
}

// executeShell 执行 shell 命令。
// 安全: 用最小化的 strings.Fields 分割（不支持 shell 元字符，避免注入），
// 调用方应避免传入含 ; | & $ ` 等元字符的命令。
// 超时: 使用执行 context（带 deadline，避免僵尸进程）。
func (s *JobService) executeShell(ctx context.Context, job *model.ScheduleJob) (string, error) {
	if job.Command == "" {
		return "", fmt.Errorf("shell 命令为空")
	}
	cmdStr := strings.TrimSpace(job.Command)
	// 阻断明显的 shell 元字符注入（命令拼接）
	for _, ch := range ";|&`$\n\r<>{}()#~" {
		if strings.ContainsRune(cmdStr, ch) {
			return "", fmt.Errorf("shell 命令包含禁用字符 %q", string(ch))
		}
	}
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}
	// 若 context 无 deadline，补一个默认上限，防止任务挂死
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("命令退出: %w", err)
	}
	return string(out), nil
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
