package service

import (
	"gorm.io/gorm"

	apperr "djangoadminx/pkg/errors"
	"djangoadminx/pkg/crypto"

	"djangoadminx/internal/model"
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

// UpdatePolicy 更新密码策略。
func (s *PolicyService) UpdatePolicy(p *model.PasswordPolicy) (*model.PasswordPolicy, error) {
	p.ID = 1 // 强制单例
	if err := s.db.Save(p).Error; err != nil {
		return nil, apperr.ErrInternal
	}
	return p, nil
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

	// 策略校验
	policy, _ := s.GetPolicy()
	if policy != nil {
		if err := policy.Validate(newPassword); err != nil {
			return apperr.New(400, err.Error())
		}
	}

	// 历史校验（防重用最近 N 个密码）
	var histories []model.PasswordHistory
	s.db.Where("user_id = ?", userID).Order("created_at DESC").
		Limit(policy.HistoryCount).Find(&histories)
	for _, h := range histories {
		if crypto.CheckPassword(h.PasswordHash, newPassword) == nil {
			return apperr.New(400, "新密码不能与最近使用过的密码相同")
		}
	}

	// 哈希新密码
	hashed, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperr.ErrInternal
	}

	// 事务: 更新密码 + 写历史
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", hashed).Error; err != nil {
			return err
		}
		return tx.Create(&model.PasswordHistory{
			UserID:       userID,
			PasswordHash: hashed,
		}).Error
	})
}
