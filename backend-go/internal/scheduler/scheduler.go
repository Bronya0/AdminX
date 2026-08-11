// Package scheduler 封装 gocron v2 调度器。
//
// HA 方案: gocron.WithDistributedLocker + gocron-redis-lock
// —— 任务级锁，多实例并行跑不同任务，同一任务同时只跑一次。
// 无 Redis 时降级为单机调度（多实例会重复执行，需告警）。
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/redis/go-redis/v9"

	"adminx/internal/model"
	"adminx/internal/repository"
	"adminx/internal/service"
)

// Manager 调度器管理器。
type Manager struct {
	scheduler gocron.Scheduler
	jobSvc    *service.JobService
	jobRepo   *repository.JobRepo
	rdb       *redis.Client
	logger    *slog.Logger
	jobIDs    map[string]gocron.Job // job.UUID → gocron.Job（用于 reload 移除）
	mu        sync.Mutex            // 保护 jobIDs 并发访问
}

// New 构造调度器。
func New(jobSvc *service.JobService, jobRepo *repository.JobRepo, rdb *redis.Client, logger *slog.Logger) (*Manager, error) {
	opts := []gocron.SchedulerOption{
		gocron.WithLogger(gocron.NewLogger(gocron.LogLevelInfo)),
		gocron.WithStopTimeout(10 * time.Second),
	}

	if rdb != nil {
		locker, err := redislock.NewRedisLocker(rdb)
		if err != nil {
			logger.Warn("创建 Redis 分布式锁失败，降级单机模式", "error", err)
		} else {
			opts = append(opts, gocron.WithDistributedLocker(locker))
			logger.Info("调度器启用 Redis 分布式锁（任务级 HA）")
		}
	} else {
		logger.Warn("无 Redis，调度器降级单机模式（多实例会重复执行任务）")
	}

	s, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建调度器失败: %w", err)
	}

	return &Manager{
		scheduler: s,
		jobSvc:    jobSvc,
		jobRepo:   jobRepo,
		rdb:       rdb,
		logger:    logger,
		jobIDs:    make(map[string]gocron.Job),
	}, nil
}

// Start 加载所有活跃任务并启动调度器。
func (m *Manager) Start(ctx context.Context) error {
	jobs, err := m.jobRepo.ActiveJobs()
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	loaded := 0
	for i := range jobs {
		if m.addJob(ctx, &jobs[i]) {
			loaded++
		}
	}
	m.scheduler.Start()
	m.logger.Info("调度器已启动", "loaded_jobs", loaded)

	// 启动心跳 goroutine：每 10s 写一次调度器心跳，供 /jobs/status/ 查询存活状态
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		// 启动后立即写一次，避免状态接口短暂显示 offline
		_ = m.jobRepo.WriteHeartbeat()
		for {
			select {
			case <-ticker.C:
				if err := m.jobRepo.WriteHeartbeat(); err != nil {
					m.logger.Warn("写调度器心跳失败", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// addJob 向调度器注册单个任务。
func (m *Manager) addJob(ctx context.Context, job *model.ScheduleJob) bool {
	jobDef, err := m.buildJobDefinition(job)
	if err != nil {
		m.logger.Warn("构建触发器失败，跳过任务", "job", job.Name, "error", err)
		return false
	}

	jobID := job.ID
	gj, err := m.scheduler.NewJob(
		jobDef,
		gocron.NewTask(func() {
			m.jobSvc.ExecuteJob(context.Background(), jobID)
		}),
		gocron.WithName(job.Name),
	)
	if err != nil {
		m.logger.Error("注册任务失败", "job", job.Name, "error", err)
		return false
	}
	m.mu.Lock()
	m.jobIDs[job.ID] = gj
	m.mu.Unlock()
	return true
}

// buildJobDefinition 根据配置构建 gocron.JobDefinition。
func (m *Manager) buildJobDefinition(job *model.ScheduleJob) (gocron.JobDefinition, error) {
	var cfg map[string]interface{}
	if job.TriggerConfig != "" {
		if err := json.Unmarshal([]byte(job.TriggerConfig), &cfg); err != nil {
			return nil, fmt.Errorf("解析触发配置失败: %w", err)
		}
	}

	switch job.TriggerType {
	case "cron":
		expr := extractString(cfg, "expr")
		if expr == "" {
			// 支持 cron 字段名兼容
			expr = extractString(cfg, "cron")
		}
		if expr == "" {
			return nil, fmt.Errorf("cron 缺少 expr 字段")
		}
		return gocron.CronJob(expr, true), nil
	case "interval":
		if seconds := extractInt(cfg, "seconds"); seconds > 0 {
			return gocron.DurationJob(time.Duration(seconds) * time.Second), nil
		}
		if minutes := extractInt(cfg, "minutes"); minutes > 0 {
			return gocron.DurationJob(time.Duration(minutes) * time.Minute), nil
		}
		if hours := extractInt(cfg, "hours"); hours > 0 {
			return gocron.DurationJob(time.Duration(hours) * time.Hour), nil
		}
		if days := extractInt(cfg, "days"); days > 0 {
			return gocron.DurationJob(time.Duration(days) * 24 * time.Hour), nil
		}
		return nil, fmt.Errorf("interval 缺少 seconds/minutes/hours/days")
	case "date":
		if dt := extractString(cfg, "datetime"); dt != "" {
			t, err := time.ParseInLocation(time.RFC3339, dt, time.Local)
			if err != nil {
				return nil, fmt.Errorf("datetime 格式错误（需 RFC3339）: %w", err)
			}
			return gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(t)), nil
		}
		return nil, fmt.Errorf("date 缺少 datetime 字段")
	}
	return nil, fmt.Errorf("未知触发类型: %s", job.TriggerType)
}

// Reload 重载所有任务（CRUD 后调用）。
func (m *Manager) Reload(ctx context.Context) error {
	// 加锁移除所有已注册任务
	m.mu.Lock()
	for id, gj := range m.jobIDs {
		_ = m.scheduler.RemoveJob(gj.ID())
		delete(m.jobIDs, id)
	}
	m.mu.Unlock()
	return m.Start(ctx)
}

// Stop 停止调度器。
func (m *Manager) Stop() error {
	return m.scheduler.Shutdown()
}

// 辅助: 从 map 提取值
func extractString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}
