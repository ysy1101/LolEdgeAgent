package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ArticleRepo struct{ db *gorm.DB }

func NewArticleRepo(db *gorm.DB) *ArticleRepo { return &ArticleRepo{db: db} }

// UpsertBatch 批量插入或忽略（根据 source_id + external_id 唯一约束）
func (r *ArticleRepo) UpsertBatch(articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&articles).Error
}

// FindByHash 根据 dedup_hash 查找
func (r *ArticleRepo) FindByHash(hash string) (*models.Article, error) {
	var a models.Article
	if err := r.db.Where("dedup_hash = ?", hash).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// FindExistingHashes 批量查询已存在的 dedup_hash
func (r *ArticleRepo) FindExistingHashes(hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}
	var existing []string
	if err := r.db.Model(&models.Article{}).
		Where("dedup_hash IN ?", hashes).
		Pluck("dedup_hash", &existing).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(existing))
	for _, h := range existing {
		set[h] = true
	}
	return set, nil
}

// List 分页列表
func (r *ArticleRepo) List(sourceID uint, page, limit int) ([]models.Article, int64, error) {
	q := r.db.Model(&models.Article{}).Order("fetched_at DESC")
	if sourceID > 0 {
		q = q.Where("source_id = ?", sourceID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []models.Article
	offset := (page - 1) * limit
	return articles, total, q.Offset(offset).Limit(limit).Find(&articles).Error
}

// UpdateSummary 更新单篇文章的摘要和评分
func (r *ArticleRepo) UpdateSummary(id uint, summary string, score float64) error {
	return r.db.Model(&models.Article{}).Where("id = ?", id).
		Updates(map[string]any{"summary": summary, "relevance_score": score}).Error
}
