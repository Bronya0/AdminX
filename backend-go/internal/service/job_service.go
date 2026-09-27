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

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
	"adminx/internal/trigger"
)

var _ = time.Second // 保留 time 引用（executeShell 超时用到）

// JobService 定时任务业务逻辑。
type JobService struct {
	db   *gorm.DB
	repo *repository.JobRepo
	log  *slog.Logger

	// onJobsChanged 任务增删改后的回调（main.go 注入）：
	// 标记 reload_pending（跨进程通知调度器）+ 触发本进程内调度器立即重载。
	onJobsChanged func()
}

func NewJobService(db *gorm.DB, repo *repository.JobRepo, logger *slog.Logger) *JobService {
	return &JobService{db: db, repo: repo, log: logger}
}

// SetOnJobsChanged 注入任务变更回调（幂等，可重复调用覆盖）。
func (s *JobService) SetOnJobsChanged(fn func()) { s.onJobsChanged = fn }

// notifyJobsChanged 任务变更后通知调度器重载。
func (s *JobService) notifyJobsChanged() {
	if s.onJobsChanged != nil {
		s.onJobsChanged()
	}
}

// Reload 手动触发调度器重载（POST /jobs/reload/）。
// 与任务 CRUD 走同一条路径：标记 reload_pending（供独立调度器进程轮询消费）+ 本进程内立即重载。
func (s *JobService) Reload() {
	s.notifyJobsChanged()
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
	// 写入前校验触发配置（解析实现与调度器共用 trigger 包）：
	// 否则接口返回 201，而调度器侧解析失败只打一行日志，任务永远不注册。
	if _, err := trigger.Parse(tt, in.TriggerConfig); err != nil {
		return nil, apperr.New(400, err.Error())
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
	s.notifyJobsChanged()
	return job, nil
}

// UpdateInput 更新任务入参（PATCH 语义：nil = 请求未携带该字段，保持原值）。
// 指针不能改回值类型：前端“启用/禁用”开关只提交 is_active，
// 零值语义会把 handler/command/trigger_config/args 一并清空，
// 任务就地停跑且配置不可恢复。
type JobUpdateInput struct {
	Name          *string `json:"name"`
	CommandType   *string `json:"command_type"`
	Handler       *string `json:"handler"`
	Command       *string `json:"command"`
	TriggerType   *string `json:"trigger_type"`
	TriggerConfig *string `json:"trigger_config"`
	Args          *string `json:"args"`
	Kwargs        *string `json:"kwargs"`
	IsActive      *bool   `json:"is_active"`
}

func (s *JobService) Update(id string, in JobUpdateInput) (*model.ScheduleJob, error) {
	job, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.ErrInternal
	}
	if in.Name != nil {
		job.Name = *in.Name
	}
	if in.CommandType != nil {
		job.CommandType = *in.CommandType
	}
	if in.Handler != nil {
		job.Handler = *in.Handler
	}
	if in.Command != nil {
		job.Command = *in.Command
	}
	if in.TriggerType != nil {
		job.TriggerType = *in.TriggerType
	}
	if in.TriggerConfig != nil {
		job.TriggerConfig = *in.TriggerConfig
	}
	if in.Args != nil {
		job.Args = *in.Args
	}
	if in.Kwargs != nil {
		job.Kwargs = *in.Kwargs
	}
	if in.IsActive != nil {
		job.IsActive = *in.IsActive
	}
	// 合并后的触发配置必须可用（PATCH 可能只改了 trigger_type/trigger_config）
	if _, err := trigger.Parse(job.TriggerType, job.TriggerConfig); err != nil {
		return nil, apperr.New(400, err.Error())
	}
	if err := s.repo.Update(job); err != nil {
		return nil, apperr.ErrInternal
	}
	s.notifyJobsChanged()
	return job, nil
}

func (s *JobService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return apperr.ErrInternal
	}
	s.notifyJobsChanged()
	return nil
}

// RunOnce 手动触发一次任务执行（不受调度器控制，同样写 job_logs）。
// 手动执行必须留痕：否则管理员跑过的副作用在日志里查不到。
func (s *JobService) RunOnce(ctx context.Context, id string) (string, error) {
	job, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", apperr.ErrNotFound
		}
		return "", apperr.ErrInternal
	}
	return s.executeWithLog(ctx, job)
}

// ExecuteJob 调度器调用的执行入口（带 JobLog 记录）。
func (s *JobService) ExecuteJob(ctx context.Context, jobID string) {
	job, err := s.repo.FindByID(jobID)
	if err != nil {
		s.log.Warn("执行任务时找不到 job", "id", jobID, "error", err)
		return
	}
	if _, err := s.executeWithLog(ctx, job); err != nil {
		// 详情已写入 job_logs，这里补一条进程日志
		s.log.Error("任务执行失败", "job", job.Name, "error", err)
	}
}

// RecordSkippedLock 记录"因未取得分布式锁而跳过本次执行"。
// gocron 拿不到锁时直接 Skip（不调用任务函数、不写任何日志），
// 必须由 AfterLockError 监听器补一条 failed 记录，否则漏跑完全无痕。
func (s *JobService) RecordSkippedLock(jobID string, cause error) {
	now := time.Now()
	entry := &model.JobLog{
		JobID:      jobID,
		Status:     "failed",
		Result:     truncateStr("未取得分布式锁，本次执行被跳过: "+cause.Error(), 500),
		StartedAt:  &now,
		FinishedAt: &now,
	}
	if err := s.repo.CreateLog(entry); err != nil {
		s.log.Warn("写跳过执行日志失败", "job_id", jobID, "error", err)
	}
}

// executeWithLog 执行任务并写 job_logs（running → success/failed）。
// 调度器执行与手动执行共用同一条路径。
func (s *JobService) executeWithLog(ctx context.Context, job *model.ScheduleJob) (string, error) {
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
	} else {
		logEntry.Status = "success"
		logEntry.Result = truncateStr(result, 500)
	}
	_ = s.repo.UpdateLog(logEntry)
	return result, execErr
}

// executeJob 执行单个任务（python handler 反射 / shell 命令）。
// panic 必须转成 error 返回：recover 后若直接返回零值，调用方会把“执行崩溃”记成 success。
func (s *JobService) executeJob(ctx context.Context, job *model.ScheduleJob) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("任务 panic", "job", job.Name, "panic", r, "stack", string(debug.Stack()))
			result = ""
			err = fmt.Errorf("任务执行 panic: %v", r)
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
	if len(result) > 1 {
		if err, ok := result[1].Interface().(error); ok && err != nil {
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
// 安全: 不经 shell 解释（exec.CommandContext 直 exec），禁用拼接元字符；
// 分词支持引号（shlex 语义的最小子集）：双/单引号包裹的参数可含空格，引号内
// 的反斜杠仅在双引号中对 " \ 两个字符转义。
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
	parts, err := splitCommand(cmdStr)
	if err != nil {
		return "", err
	}
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

// splitCommand 按空格分词，支持单/双引号包裹含空格的参数（shlex 最小子集）。
// 引号必须成对，否则报错。
func splitCommand(s string) ([]string, error) {
	var parts []string
	var cur strings.Builder
	inWord := false
	var quote byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			} else if quote == '"' && ch == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
				i++
				cur.WriteByte(s[i])
			} else {
				cur.WriteByte(ch)
			}
		case ch == '"' || ch == '\'':
			quote = ch
			inWord = true
		case ch == ' ' || ch == '\t':
			if inWord {
				parts = append(parts, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(ch)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("命令存在未闭合的引号")
	}
	if inWord {
		parts = append(parts, cur.String())
	}
	return parts, nil
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
