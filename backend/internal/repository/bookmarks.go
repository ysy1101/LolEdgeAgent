package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type BookmarkRepo struct{ db *gorm.DB }

func NewBookmarkRepo(db *gorm.DB) *BookmarkRepo { return &BookmarkRepo{db: db} }

func (r *BookmarkRepo) Create(b *models.Bookmark) error { return r.db.Create(b).Error }

func (r *BookmarkRepo) Delete(userID, articleID uint) error {
	return r.db.Where("user_id = ? AND article_id = ?", userID, articleID).Delete(&models.Bookmark{}).Error
}

func (r *BookmarkRepo) IsBookmarked(userID, articleID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Bookmark{}).
		Where("user_id = ? AND article_id = ?", userID, articleID).Count(&count).Error
	return count > 0, err
}

// List 用户收藏列表（含文章）
func (r *BookmarkRepo) List(userID uint, page, limit int) ([]models.Article, int64, error) {
	q := r.db.Model(&models.Article{}).
		Joins("JOIN bookmarks ON articles.id = bookmarks.article_id").
		Where("bookmarks.user_id = ?", userID).
		Order("bookmarks.created_at DESC")

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []models.Article
	offset := (page - 1) * limit
	return articles, total, q.Offset(offset).Limit(limit).Find(&articles).Error
}
