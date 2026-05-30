package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BriefingRepo struct{ db *gorm.DB }

func NewBriefingRepo(db *gorm.DB) *BriefingRepo { return &BriefingRepo{db: db} }

func (r *BriefingRepo) Create(b *models.Briefing) error { return r.db.Create(b).Error }

// CreateWithArticles 创建简报并关联文章
func (r *BriefingRepo) CreateWithArticles(b *models.Briefing, articles []models.BriefingArticle) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(b).Error; err != nil {
			return err
		}
		if len(articles) > 0 {
			for i := range articles {
				articles[i].BriefingID = b.ID
			}
			return tx.Create(&articles).Error
		}
		return nil
	})
}

// List 分页列表（按用户）
func (r *BriefingRepo) List(userID uint, page, limit int) ([]models.Briefing, int64, error) {
	var total int64
	q := r.db.Model(&models.Briefing{}).Where("user_id = ?", userID).Order("generated_at DESC")

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var briefings []models.Briefing
	offset := (page - 1) * limit
	return briefings, total, q.Offset(offset).Limit(limit).Find(&briefings).Error
}

// GetByID 获取简报详情（含关联文章）
func (r *BriefingRepo) GetByID(id uint) (*models.Briefing, error) {
	var b models.Briefing
	if err := r.db.First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// GetArticles 获取简报关联的文章列表
func (r *BriefingRepo) GetArticles(briefingID uint) ([]models.Article, error) {
	var articles []models.Article
	err := r.db.Table("articles").
		Joins("JOIN briefing_articles ON articles.id = briefing_articles.article_id").
		Where("briefing_articles.briefing_id = ?", briefingID).
		Order("briefing_articles.rank_position ASC").
		Find(&articles).Error
	return articles, err
}

func (r *BriefingRepo) Delete(id uint) error {
	return r.db.Select(clause.Associations).Delete(&models.Briefing{}, id).Error
}

func (r *BriefingRepo) UpdateStatus(id uint, status string, errMsg string) error {
	return r.db.Model(&models.Briefing{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "error_message": errMsg}).Error
}
