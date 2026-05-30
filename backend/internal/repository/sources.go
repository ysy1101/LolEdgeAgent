package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type SourceRepo struct{ db *gorm.DB }

func NewSourceRepo(db *gorm.DB) *SourceRepo { return &SourceRepo{db: db} }

func (r *SourceRepo) List(enabledOnly bool) ([]models.Source, error) {
	q := r.db.Order("created_at DESC")
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	var sources []models.Source
	return sources, q.Find(&sources).Error
}

func (r *SourceRepo) Get(id uint) (*models.Source, error) {
	var s models.Source
	if err := r.db.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SourceRepo) Create(s *models.Source) error { return r.db.Create(s).Error }

func (r *SourceRepo) Update(s *models.Source) error { return r.db.Save(s).Error }

func (r *SourceRepo) Delete(id uint) error { return r.db.Delete(&models.Source{}, id).Error }
