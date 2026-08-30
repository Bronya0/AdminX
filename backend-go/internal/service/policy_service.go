package service

import (
	"time"

	"gorm.io/gorm"

	"adminx/pkg/crypto"
	apperr "adminx/pkg/errors"

	"adminx/internal/model"
)

// PolicyService 密码策略。
type PolicyService struct {
	db *gorm.DB
}

func NewPolicyService(db *gorm.DB) *PolicyService { return &PolicyService{db: db} }

// GetPolicy 获取密码策略（单例，ID=1）。
func (s *PolicyService) GetPolicy() (*model.PasswordPolicy, error) {
	var p model.PasswordPolicy
	result := s.db.First(&p, 1)
	if result.Error == gorm.ErrRecordNotFound {
		// 不存在则返回默认策略
		return &model.PasswordPolicy{
			ID: 1, MinLength: 8, RequireUpper: true, RequireLower: true,
			RequireDigit: true, RequireSpecial: true, ExpireDays: 90,
			HistoryCount: 5, IsActive: true,
		}, nil
	}
	return &p, result.Error
}

// PolicyUpdateInput 策略更新入参（指针字段区分"未传"与"清零"，杜绝 map 直通 Updates 的任意列写入）。
type PolicyUpdateInput struct {
	MinLength     *int   `json:"min_length"`
	RequireUpper  *bool  `json:"require_upper"`
	RequireLower  *bool  `json:"require_lower"`
	RequireDigit  *bool  `json:"require_digit"`
	RequireSpecial *bool `json:"require_special"`
	ExpireDays    *int   `json:"expire_days"`
	HistoryCount  *int   `json:"history_count"`
	IsActive      *bool  `json:"is_active"`
}

// UpdatePolicy 更新密码策略（白名单字段）。
func (s *PolicyService) UpdatePolicy(in PolicyUpdateInput) (*model.PasswordPolicy, error) {
	updates := map[string]interface{}{}
	if in.MinLength != nil {
		updates["min_length"] = *in.MinLength
	}
	if in.RequireUpper != nil {
		updates["require_upper"] = *in.RequireUpper
	}
	if in.RequireLower != nil {
		updates["require_lower"] = *in.RequireLower
	}
	if in.RequireDigit != nil {
		updates["require_digit"] = *in.RequireDigit
	}
	if in.RequireSpecial != nil {
		updates["require_special"] = *in.RequireSpecial
	}
	if in.ExpireDays != nil {
		updates["expire_days"] = *in.ExpireDays
	}
	if in.HistoryCount != nil {
		updates["history_count"] = *in.HistoryCount
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}
	if len(updates) == 0 {
		return s.GetPolicy()
	}
	if err := s.db.Model(&model.PasswordPolicy{}).Where("id = 1").Updates(updates).Error; err != nil {
		return nil, apperr.ErrInternal
	}
	var p model.PasswordPolicy
	if err := s.db.First(&p, 1).Error; err != nil {
		return nil, apperr.ErrNotFound
	}
	return &p, nil
}

// ValidatePassword 校验密码是否符合当前策略（供用户创建/重置等场景复用）。
func (s *PolicyService) ValidatePassword(password string) error {
	return validatePasswordPolicy(s.db, password)
}

// validatePasswordPolicy 包级辅助：查库取策略并校验（无策略记录时用 model 默认值）。
func validatePasswordPolicy(db *gorm.DB, password string) error {
	var p model.PasswordPolicy
	if err := db.First(&p, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			p = model.PasswordPolicy{
				ID: 1, MinLength: 8, RequireUpper: true, RequireLower: true,
				RequireDigit: true, RequireSpecial: true, ExpireDays: 90,
				HistoryCount: 5, IsActive: true,
			}
		} else {
			return apperr.Wrap(500, "获取密码策略失败", err)
		}
	}
	if err := p.Validate(password); err != nil {
		return apperr.New(400, err.Error())
	}
	return nil
}

// ChangePassword 用户修改自己的密码（含历史校验 + 策略校验）。
func (s *PolicyService) ChangePassword(userID, oldPassword, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return apperr.ErrNotFound
	}

	// 校验旧密码
	if err := crypto.CheckPassword(user.Password, oldPassword); err != nil {
		return apperr.New(400, "旧密码错误")
	}

	// 策略校验（GetPolicy 不会返回 nil policy，但保险起见处理 error）
	policy, pErr := s.GetPolicy()
	if pErr != nil {
		return apperr.Wrap(500, "获取密码策略失败", pErr)
	}
	if policy == nil {
		policy = &model.PasswordPolicy{
			ID: 1, MinLength: 8, HistoryCount: 5, IsActive: true,
		}
	}
	if err := policy.Validate(newPassword); err != nil {
		return apperr.New(400, err.Error())
	}
	historyCount := policy.HistoryCount
	if historyCount < 0 {
		historyCount = 0
	}

	// 哈希新密码
	hashed, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperr.ErrInternal
	}

	// 事务: 历史检查 + 更新密码 + 写历史
	return s.db.Transaction(func(tx *gorm.DB) error {
		var histories []model.PasswordHistory
		if historyCount > 0 {
			if err := tx.Where("user_id = ?", userID).Order("created_at DESC").
				Limit(historyCount).Find(&histories).Error; err != nil {
				return err
			}
		}
		for _, h := range histories {
			if crypto.CheckPassword(h.PasswordHash, newPassword) == nil {
				return apperr.New(400, "新密码不能与最近使用过的密码相同")
			}
		}
		if err := tx.Model(&user).Update("password", hashed).Error; err != nil {
			return err
		}
		// 改密成功即吊销该用户所有既有 token（last_logout 之后签发的 token 才有效），
		// 防止"密码可能已泄露但旧会话仍存活"的窗口。
		if err := tx.Model(&model.User{}).Where("id = ?", userID).
			Update("last_logout", time.Now()).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.PasswordHistory{
			UserID:       userID,
			PasswordHash: hashed,
		}).Error; err != nil {
			return err
		}
		// 清理超出保留数量的历史记录（仅保留最近 historyCount 条）
		if historyCount > 0 {
			var extraIDs []string
			if err := tx.Model(&model.PasswordHistory{}).
				Where("user_id = ?", userID).
				Order("created_at DESC").
				Offset(historyCount).
				Pluck("id", &extraIDs).Error; err != nil {
				return err
			}
			if len(extraIDs) > 0 {
				if err := tx.Where("id IN ?", extraIDs).Delete(&model.PasswordHistory{}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
