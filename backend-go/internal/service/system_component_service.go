package service

import (
	"context"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"adminx/internal/repository"
)

// SystemComponentItem 系统内置组件状态（对齐前端 types 的 SystemComponentItem）。
type SystemComponentItem struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"` // ok / error / offline
	Backend string `json:"backend,omitempty"`
	Message string `json:"message"`
}

// PlatformInfo 运行时平台信息（前端「组件管理」页头部展示）。
type PlatformInfo struct {
	RuntimeVersion string `json:"runtime_version"`
	OS             string `json:"os"`
}

// SystemComponentsData GET /common/components/ 的响应。
type SystemComponentsData struct {
	Platform   PlatformInfo          `json:"platform"`
	Components []SystemComponentItem `json:"components"`
}

// SystemComponentService 系统内置组件（数据库 / Redis / 调度器）健康探测。
type SystemComponentService struct {
	db      *gorm.DB
	rdb     *redis.Client
	jobRepo *repository.JobRepo
}

// NewSystemComponentService 构造。rdb 为 nil 表示 Redis 未配置（后端降级运行）。
func NewSystemComponentService(db *gorm.DB, rdb *redis.Client, jobRepo *repository.JobRepo) *SystemComponentService {
	return &SystemComponentService{db: db, rdb: rdb, jobRepo: jobRepo}
}

// Status 汇总内置组件状态：单项探测失败不影响其他项（各自带 message 说明）。
func (s *SystemComponentService) Status(ctx context.Context) *SystemComponentsData {
	return &SystemComponentsData{
		Platform: PlatformInfo{
			RuntimeVersion: runtime.Version(),
			OS:             runtime.GOOS + "/" + runtime.GOARCH,
		},
		Components: []SystemComponentItem{
			s.databaseStatus(ctx),
			s.redisStatus(ctx),
			s.schedulerStatus(),
		},
	}
}

// databaseStatus 数据库连通性（含方言，便于确认跑在 postgres 还是 sqlite 上）。
func (s *SystemComponentService) databaseStatus(ctx context.Context) SystemComponentItem {
	item := SystemComponentItem{Key: "database", Name: "数据库", Type: "system"}
	if s.db == nil {
		item.Status = "offline"
		item.Message = "未初始化"
		return item
	}
	item.Backend = s.db.Dialector.Name()

	sqlDB, err := s.db.DB()
	if err != nil {
		item.Status = "error"
		item.Message = err.Error()
		return item
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		item.Status = "error"
		item.Message = err.Error()
		return item
	}
	item.Status = "ok"
	item.Message = "连接正常"
	return item
}

// redisStatus Redis 连通性；未配置时明确标注为降级而非故障。
func (s *SystemComponentService) redisStatus(ctx context.Context) SystemComponentItem {
	item := SystemComponentItem{Key: "redis", Name: "Redis", Type: "system"}
	if s.rdb == nil {
		item.Status = "offline"
		item.Message = "未配置：后端降级为无缓存模式（验证码/黑名单/调度器锁不可用）"
		return item
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.rdb.Ping(pingCtx).Err(); err != nil {
		item.Status = "error"
		item.Message = err.Error()
		return item
	}
	item.Status = "ok"
	item.Message = "连接正常"
	return item
}

// schedulerStatus 内置调度器心跳（30s 内无心跳视为未运行）。
func (s *SystemComponentService) schedulerStatus() SystemComponentItem {
	item := SystemComponentItem{Key: "scheduler", Name: "调度器", Type: "system"}
	if s.jobRepo == nil {
		item.Status = "offline"
		item.Message = "未初始化"
		return item
	}
	alive, err := s.jobRepo.IsSchedulerAlive()
	if err != nil {
		item.Status = "error"
		item.Message = err.Error()
		return item
	}
	if !alive {
		item.Status = "offline"
		item.Message = "未运行（未以 --scheduler 启动或心跳已过期）"
		return item
	}
	item.Status = "ok"
	item.Message = "运行中"
	return item
}
