package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"adminx/internal/model"
)

// LoginLockRepo 登录锁定数据访问。
type LoginLockRepo struct {
	db *gorm.DB
}

func NewLoginLockRepo(db *gorm.DB) *LoginLockRepo { return &LoginLockRepo{db: db} }

// Get 按用户名查询锁定记录。
func (r *LoginLockRepo) Get(username string) (*model.LoginLock, error) {
	var lock model.LoginLock
	err := r.db.Where("username = ?", username).First(&lock).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &lock, err
}

// IncrementFailed 原子递增失败次数，达到阈值时设置 locked_at。
//
// 用 ON CONFLICT(username) DO UPDATE 一次写完（postgres / sqlite 都支持）：
// 先查后插在并发下会双双查不到、双双 INSERT，撞 login_locks.username 唯一索引，
// 把“密码错误计数 +1”变成一个 500。
func (r *LoginLockRepo) IncrementFailed(username string, maxAttempts int) error {
	now := time.Now()
	// 插入分支（首次失败）：计数 1，阈值 <=1 时同时锁定。
	var lockedAt *time.Time
	if maxAttempts <= 1 {
		lockedAt = &now
	}
	lock := model.LoginLock{
		Username:    username,
		FailedCount: 1,
		LockedAt:    lockedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "username"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"failed_count": gorm.Expr("failed_count + 1"),
			// SET 右侧均按旧行求值：failed_count 即递增前的值，+1 后判阈值。
			// 未达阈值时保持原 locked_at 不变（与旧实现“只在达阀时更新 locked_at”一致）。
			"locked_at":  gorm.Expr("CASE WHEN failed_count + 1 >= ? THEN ? ELSE locked_at END", maxAttempts, now),
			"updated_at": now,
		}),
	}).Create(&lock).Error
}

// IsLocked 检查是否被锁定且未过锁定期。返回 (locked, remainingSeconds)。
func (r *LoginLockRepo) IsLocked(username string, lockDuration time.Duration) (bool, int, error) {
	lock, err := r.Get(username)
	if err != nil {
		return false, 0, err
	}
	if lock == nil || lock.LockedAt == nil {
		return false, 0, nil
	}
	elapsed := time.Since(*lock.LockedAt)
	if elapsed >= lockDuration {
		// 锁定期已过，自动删除记录
		_ = r.Delete(username)
		return false, 0, nil
	}
	remaining := int((lockDuration - elapsed).Seconds())
	return true, remaining, nil
}

// Delete 删除锁定记录（登录成功时调用）。
func (r *LoginLockRepo) Delete(username string) error {
	return r.db.Where("username = ?", username).Delete(&model.LoginLock{}).Error
}
