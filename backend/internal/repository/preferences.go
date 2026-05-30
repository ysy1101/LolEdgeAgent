package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type PreferenceRepo struct{ db *gorm.DB }

func NewPreferenceRepo(db *gorm.DB) *PreferenceRepo { return &PreferenceRepo{db: db} }

// Get 获取用户偏好，不存在则创建默认值
func (r *PreferenceRepo) Get(userID uint) (*models.Preference, error) {
	var p models.Preference
	if err := r.db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			p = models.Preference{UserID: userID}
			if createErr := r.db.Create(&p).Error; createErr != nil {
				return nil, createErr
			}
			return &p, nil
		}
		return nil, err
	}
	return &p, nil
}

// Update 全量更新偏好
func (r *PreferenceRepo) Update(userID uint, p *models.Preference) error {
	p.UserID = userID
	return r.db.Where("user_id = ?", userID).Save(p).Error
}
