package repository

import (
	"time"

	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type FetchLogRepo struct{ db *gorm.DB }

func NewFetchLogRepo(db *gorm.DB) *FetchLogRepo { return &FetchLogRepo{db: db} }

func (r *FetchLogRepo) Create(log *models.FetchLog) error { return r.db.Create(log).Error }

func (r *FetchLogRepo) Finish(id uint, status string, count int, errMsg string) error {
	now := time.Now()
	return r.db.Model(&models.FetchLog{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":           status,
			"articles_fetched": count,
			"error_message":    errMsg,
			"completed_at":     &now,
		}).Error
}

// ListBySource 按源查询最近日志
func (r *FetchLogRepo) ListBySource(sourceID uint, limit int) ([]models.FetchLog, error) {
	var logs []models.FetchLog
	return logs, r.db.Where("source_id = ?", sourceID).
		Order("started_at DESC").Limit(limit).Find(&logs).Error
}
