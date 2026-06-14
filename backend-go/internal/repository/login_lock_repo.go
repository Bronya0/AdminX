package repository

import (
	"time"

	"gorm.io/gorm"

	"djangoadminx/internal/model"
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
func (r *LoginLockRepo) IncrementFailed(username string, maxAttempts int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var lock model.LoginLock
		result := tx.Where("username = ?", username).First(&lock)
		if result.Error == gorm.ErrRecordNotFound {
			// 新建记录
			lock = model.LoginLock{Username: username, FailedCount: 1}
			if err := tx.Create(&lock).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			// 原子递增
			if err := tx.Model(&lock).UpdateColumn("failed_count", gorm.Expr("failed_count + 1")).Error; err != nil {
				return err
			}
			lock.FailedCount++
		}
		// 达到阈值则锁定
		if lock.FailedCount >= maxAttempts {
			now := time.Now()
			return tx.Model(&lock).Where("username = ?", username).
				Update("locked_at", now).Error
		}
		return nil
	})
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
