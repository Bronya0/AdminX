package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScheduleJob 定时任务定义。
type ScheduleJob struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	CommandType   string    `gorm:"size:20;not null;default:'python'" json:"command_type"` // python/shell
	Handler       string    `gorm:"size:255;not null;default:''" json:"handler"`           // 如 "jobs.tasks.NtpSync"
	Command       string    `gorm:"type:text;not null;default:''" json:"command"`          // shell 命令
	TriggerType   string    `gorm:"size:20;not null;default:'interval'" json:"trigger_type"` // cron/interval/date
	TriggerConfig string    `gorm:"type:text;not null;default:''" json:"trigger_config"`   // JSON，如 {"minutes":5}
	Args          string    `gorm:"type:text;not null;default:''" json:"args"`             // JSON 数组
	Kwargs        string    `gorm:"type:text;not null;default:'{}'" json:"kwargs"`          // JSON 对象
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ScheduleJob) TableName() string { return "schedule_jobs" }

func (j *ScheduleJob) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	return nil
}

// JobLog 任务执行日志。
type JobLog struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	JobID      string     `gorm:"type:uuid;not null;index" json:"job_id"`
	Status     string     `gorm:"size:20;not null;default:'running'" json:"status"` // running/success/failed
	Result     string     `gorm:"type:text;not null;default:''" json:"result"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func (JobLog) TableName() string { return "job_logs" }

func (l *JobLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}

// SchedulerHeartbeat 调度器心跳（单例行，固定 ID）。
// Go 侧用 gocron 任务级锁，不需要 leader 选举，
// 但保留此表用于 API 状态查询兼容性。
type SchedulerHeartbeat struct {
	ID             string     `gorm:"type:uuid;primaryKey" json:"id"`
	LastHeartbeat  *time.Time `json:"last_heartbeat"`
	ReloadPending  bool       `gorm:"not null;default:false" json:"reload_pending"`
}

func (SchedulerHeartbeat) TableName() string { return "scheduler_heartbeats" }
