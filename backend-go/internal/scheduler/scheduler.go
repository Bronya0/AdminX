// Package scheduler 封装 gocron v2 调度器。
//
// HA 方案: gocron.WithDistributedLocker + gocron-redis-lock
// —— 任务级锁，多实例并行跑不同任务，同一任务同时只跑一次。
// 无 Redis 时降级为单机调度（多实例会重复执行，需告警）。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"adminx/internal/model"
	"adminx/internal/repository"
	"adminx/internal/service"
	"adminx/internal/trigger"
)

const (
	// lockKeyPrefix 分布式锁 key 前缀（完整 key = 前缀 + 数据库任务 ID）
	lockKeyPrefix = "adminx:scheduler:job:"
	// defaultLockTTL 未配置或配置过小时的锁 TTL。
	// 库默认只有 8s，而 shell 任务可以跑几分钟。
	defaultLockTTL = 30 * time.Second
	// minLockTTL 锁 TTL 下限（配合自动续期使用）
	minLockTTL = 10 * time.Second
	// shutdownTimeout 关闭时等待在跑任务收尾的上限（须大于 WithStopTimeout）
	shutdownTimeout = 15 * time.Second
	// reloadTimeout 单次重载的超时
	reloadTimeout = 30 * time.Second
	// heartbeatInterval 心跳/跨进程 reload 轮询间隔
	heartbeatInterval = 10 * time.Second
)

// Manager 调度器管理器。
type Manager struct {
	scheduler       gocron.Scheduler
	jobSvc          *service.JobService
	jobRepo         *repository.JobRepo
	rdb             *redis.Client
	logger          *slog.Logger
	jobIDs          map[string]gocron.Job // 数据库任务 ID → gocron.Job（用于 reload 移除）
	mu              sync.Mutex            // 保护 jobIDs / heartbeatCancel / started 并发访问
	heartbeatCancel context.CancelFunc    // 心跳 goroutine 的取消句柄（防止 Reload 重复启动泄漏）
	// started 记录是否已 Start 过：gocron 重复 Start 只打一行 "already started" 警告，
	// 而每次任务 CRUD 都会 Reload，白白刷日志。
	started bool
	// jobCtx 是任务执行上下文：Stop 时取消，让正在运行的 shell 子进程被真正终止
	// （否则 SIGTERM 后子进程变孤儿继续跑，job_logs 留下永久 running 的行）。
	jobCtx    context.Context
	jobCancel context.CancelFunc
}

// New 构造调度器。lockTTLSec 为分布式锁 TTL（秒），<=0 或过小时用 defaultLockTTL。
func New(jobSvc *service.JobService, jobRepo *repository.JobRepo, rdb *redis.Client, lockTTLSec int, logger *slog.Logger) (*Manager, error) {
	opts := []gocron.SchedulerOption{
		gocron.WithLogger(gocron.NewLogger(gocron.LogLevelInfo)),
		gocron.WithStopTimeout(10 * time.Second),
	}

	if rdb != nil {
		ttl := time.Duration(lockTTLSec) * time.Second
		if ttl < minLockTTL {
			ttl = defaultLockTTL
		}
		// 必须显式设置 TTL（库默认仅 8s）+ 自动续期：
		// 否则执行超过 TTL 的任务在锁过期后会被另一个实例同时执行，
		// 文档承诺的"同一任务同时只跑一次"就不成立。
		locker, err := redislock.NewRedisLockerWithOptions(rdb,
			redislock.WithRedsyncOptions(
				redislock.WithExpiry(ttl),
				redislock.WithTries(3),
				redislock.WithRetryDelay(50*time.Millisecond),
			),
			redislock.WithAutoExtendDuration(ttl/2),
			redislock.WithKeyPrefix(lockKeyPrefix),
		)
		if err != nil {
			logger.Warn("创建 Redis 分布式锁失败，降级单机模式", "error", err)
		} else {
			opts = append(opts, gocron.WithDistributedLocker(locker))
			logger.Info("调度器启用 Redis 分布式锁（任务级 HA）", "lock_ttl", ttl.String())
		}
	} else {
		logger.Warn("无 Redis，调度器降级单机模式（多实例会重复执行任务）")
	}

	s, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建调度器失败: %w", err)
	}

	jobCtx, jobCancel := context.WithCancel(context.Background())
	return &Manager{
		scheduler: s,
		jobSvc:    jobSvc,
		jobRepo:   jobRepo,
		rdb:       rdb,
		logger:    logger,
		jobIDs:    make(map[string]gocron.Job),
		jobCtx:    jobCtx,
		jobCancel: jobCancel,
	}, nil
}

// Start 加载所有活跃任务并启动调度器（幂等）。
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadLocked(ctx)
}

// Reload 重载所有任务（CRUD 后调用）。
func (m *Manager) Reload(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadLocked(ctx)
}

// reloadLocked 在持锁状态下重建全部任务（调用方必须已持有 m.mu）。
//
// 两点必须维持：
//  1. 先读库、再动现有任务 —— 读失败时保持当前已注册任务不变，
//     否则一次瞬时 DB 错误就把全部定时任务清空且不再恢复；
//  2. 移除与重建全程持锁 —— 若在两者之间放开锁，并发重载会让同一任务被注册两次，
//     而 jobIDs 只留后一个，前一个成为再也清不掉的孤儿（该任务每次触发跑两遍）。
func (m *Manager) reloadLocked(ctx context.Context) error {
	jobs, err := m.jobRepo.ActiveJobs()
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	for id, gj := range m.jobIDs {
		_ = m.scheduler.RemoveJob(gj.ID())
		delete(m.jobIDs, id)
	}

	loaded := 0
	for i := range jobs {
		if m.addJobLocked(&jobs[i]) {
			loaded++
		}
	}
	// 首次才调 Start（已启动时新增/移除任务直接生效，不必再 Start）
	if !m.started {
		m.scheduler.Start()
		m.started = true
	}
	m.logger.Info("调度任务已就绪", "loaded_jobs", loaded)

	// 启动心跳 goroutine（幂等：已有则复用，防止 Reload 重复启动泄漏）
	m.startHeartbeatLocked()
	return nil
}

// startHeartbeatLocked 启动心跳 goroutine（调用方必须已持有 m.mu）。
// 每 10s 写一次调度器心跳，供 /jobs/status/ 查询存活状态，并消费跨进程 reload 通知。
func (m *Manager) startHeartbeatLocked() {
	if m.heartbeatCancel != nil {
		return // 已在运行
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.heartbeatCancel = cancel

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		// 启动后立即写一次，避免状态接口短暂显示 offline
		_ = m.jobRepo.WriteHeartbeat()
		for {
			select {
			case <-ticker.C:
				if err := m.jobRepo.WriteHeartbeat(); err != nil {
					m.logger.Warn("写调度器心跳失败", "error", err)
				}
				// 消费跨进程 reload 通知（web 进程任务 CRUD 置位）→ 重载全部任务
				if pending, err := m.jobRepo.ConsumeReloadPending(); err != nil {
					m.logger.Warn("读取 reload_pending 失败", "error", err)
				} else if pending {
					m.logger.Info("收到 reload_pending，重载调度任务")
					rctx, rcancel := context.WithTimeout(context.Background(), reloadTimeout)
					if err := m.Reload(rctx); err != nil {
						m.logger.Error("调度器重载失败，重置 reload_pending 待下次重试", "error", err)
						// 标记已被消费而重载未生效：必须放回去，否则再也没人触发重载，
						// 全量定时任务会静默停摆到下一次任务变更或重启。
						if err := m.jobRepo.SetReloadPending(); err != nil {
							m.logger.Warn("重置 reload_pending 失败", "error", err)
						}
					}
					rcancel()
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// addJobLocked 向调度器注册单个任务（调用方必须已持有 m.mu）。
func (m *Manager) addJobLocked(job *model.ScheduleJob) bool {
	jobDef, err := buildJobDefinition(job.TriggerType, job.TriggerConfig)
	if err != nil {
		m.logger.Warn("构建触发器失败，跳过任务", "job", job.Name, "job_id", job.ID, "error", err)
		return false
	}

	jobID := job.ID
	gj, err := m.scheduler.NewJob(
		jobDef,
		gocron.NewTask(func() {
			m.jobSvc.ExecuteJob(m.jobCtx, jobID)
		}),
		// 任务名即 gocron 分布式锁的 key：必须用数据库任务 ID。
		// 用任务名会让两个同名任务共用一把锁，其中一个的执行被静默吞掉
		// （name 列没有唯一约束，接口也不校验重名）。
		gocron.WithName(job.ID),
		gocron.WithEventListeners(
			// 拿不到锁时 gocron 直接 Skip：不执行任务函数、不写任何日志。
			// 注册监听后才能在 job_logs 里看到"被跳过"，
			// 否则锁竞争 / Redis 抖动期间任务静默漏跑而运维只看得到心跳正常。
			gocron.AfterLockError(func(_ uuid.UUID, lockedJobName string, lockErr error) {
				m.logger.Warn("未取得分布式锁，本次执行被跳过",
					"job_id", lockedJobName, "error", lockErr)
				m.jobSvc.RecordSkippedLock(lockedJobName, lockErr)
			}),
		),
	)
	if err != nil {
		m.logger.Error("注册任务失败", "job", job.Name, "job_id", job.ID, "error", err)
		return false
	}
	m.jobIDs[job.ID] = gj
	return true
}

// Stop 停止调度器（含心跳 goroutine），并取消正在执行的任务。
func (m *Manager) Stop() error {
	m.mu.Lock()
	if m.heartbeatCancel != nil {
		m.heartbeatCancel()
		m.heartbeatCancel = nil
	}
	jobCancel := m.jobCancel
	m.mu.Unlock()

	// 先取消任务上下文，再等待收尾：正在跑的 shell 任务必须被终止，
	// 否则进程退出后子进程被 init 收养继续执行（且 job_logs 留永久 running）。
	if jobCancel != nil {
		jobCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return m.scheduler.ShutdownWithContext(ctx)
}

// buildJobDefinition 由触发配置构建 gocron 定义。
// 解析与校验统一走 trigger 包（与 CRUD 写入校验同一实现）。
func buildJobDefinition(triggerType, triggerConfig string) (gocron.JobDefinition, error) {
	spec, err := trigger.Parse(triggerType, triggerConfig)
	if err != nil {
		return nil, err
	}
	switch spec.Kind {
	case trigger.KindCron:
		return gocron.CronJob(spec.CronExpr, spec.WithSeconds), nil
	case trigger.KindInterval:
		return gocron.DurationJob(spec.Interval), nil
	case trigger.KindDate:
		return gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(spec.RunAt)), nil
	default:
		return nil, fmt.Errorf("未知触发类型: %s", spec.Kind)
	}
}
