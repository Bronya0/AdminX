package model

import "time"

// MonitorSample 系统资源历史采样点（供 /monitor/resources/history/ 出趋势图）。
//
// 采样不依赖调度器：每次拉实时资源时按最小间隔补一条（见 service.MonitorService），
// 因此单实例/多实例都不会产生高频写入。
type MonitorSample struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CPUPercent    float64   `gorm:"not null;default:0" json:"cpu_percent"`
	MemoryPercent float64   `gorm:"not null;default:0" json:"memory_percent"`
	DiskPercent   float64   `gorm:"not null;default:0" json:"disk_percent"`
	DiskReadMBps  float64   `gorm:"not null;default:0" json:"disk_read_mbps"`
	DiskWriteMBps float64   `gorm:"not null;default:0" json:"disk_write_mbps"`
	CreatedAt     time.Time `gorm:"index;autoCreateTime" json:"created_at"`
}

func (MonitorSample) TableName() string { return "monitor_samples" }
