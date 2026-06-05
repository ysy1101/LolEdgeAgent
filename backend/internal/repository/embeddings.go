package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type EmbeddingRepo struct{ db *gorm.DB }

func NewEmbeddingRepo(db *gorm.DB) *EmbeddingRepo { return &EmbeddingRepo{db: db} }

func (r *EmbeddingRepo) Upsert(emb *models.ArticleEmbedding) error {
	return r.db.Where("article_id = ?", emb.ArticleID).
		Assign(emb).FirstOrCreate(emb).Error
}

func (r *EmbeddingRepo) GetAll() ([]models.ArticleEmbedding, error) {
	var list []models.ArticleEmbedding
	return list, r.db.Order("id").Find(&list).Error
}

func (r *EmbeddingRepo) Exists(articleID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.ArticleEmbedding{}).Where("article_id = ?", articleID).Count(&count).Error
	return count > 0, err
}
