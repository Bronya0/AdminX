package repository

import (
	"gorm.io/gorm"

	"djangoadminx/internal/model"
)

// JobRepo 定时任务数据访问。
type JobRepo struct {
	db *gorm.DB
}

func NewJobRepo(db *gorm.DB) *JobRepo { return &JobRepo{db: db} }

func (r *JobRepo) FindByID(id string) (*model.ScheduleJob, error) {
	var j model.ScheduleJob
	err := r.db.First(&j, "id = ?", id).Error
	return &j, err
}

func (r *JobRepo) List(offset, limit int, search string) ([]model.ScheduleJob, int64, error) {
	var jobs []model.ScheduleJob
	var count int64
	q := r.db.Model(&model.ScheduleJob{})
	if search != "" {
		q = q.Where("name LIKE ?", "%"+search+"%")
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&jobs).Error
	return jobs, count, err
}

// ActiveJobs 查询所有启用的任务（调度器启动时加载）。
func (r *JobRepo) ActiveJobs() ([]model.ScheduleJob, error) {
	var jobs []model.ScheduleJob
	err := r.db.Where("is_active = ?", true).Find(&jobs).Error
	return jobs, err
}

func (r *JobRepo) Create(j *model.ScheduleJob) error {
	return r.db.Create(j).Error
}

func (r *JobRepo) Update(j *model.ScheduleJob) error {
	return r.db.Save(j).Error
}

func (r *JobRepo) Delete(id string) error {
	return r.db.Delete(&model.ScheduleJob{}, "id = ?", id).Error
}

// ── JobLog ──

func (r *JobRepo) CreateLog(l *model.JobLog) error {
	return r.db.Create(l).Error
}

func (r *JobRepo) UpdateLog(l *model.JobLog) error {
	return r.db.Save(l).Error
}

func (r *JobRepo) ListLogs(offset, limit int, jobID, status string) ([]model.JobLog, int64, error) {
	var logs []model.JobLog
	var count int64
	q := r.db.Model(&model.JobLog{})
	if jobID != "" {
		q = q.Where("job_id = ?", jobID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("started_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, count, err
}

// ── SchedulerHeartbeat ──

// WriteHeartbeat 写调度器心跳（固定单例 ID）。
func (r *JobRepo) WriteHeartbeat() error {
	return r.db.Exec(`INSERT INTO scheduler_heartbeats (id, last_heartbeat, reload_pending)
		VALUES ('00000000-0000-0000-0000-000000000001', NOW(), false)
		ON CONFLICT (id) DO UPDATE SET last_heartbeat = NOW()`).Error
}

// IsSchedulerAlive 检查调度器是否存活（30s 内有心跳）。
func (r *JobRepo) IsSchedulerAlive() (bool, error) {
	var count int64
	err := r.db.Raw(`SELECT COUNT(*) FROM scheduler_heartbeats
		WHERE id = '00000000-0000-0000-0000-000000000001'
		AND last_heartbeat > NOW() - INTERVAL '30 seconds'`).Scan(&count).Error
	return count > 0, err
}
