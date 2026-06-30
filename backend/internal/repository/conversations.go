package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type ConversationRepo struct{ db *gorm.DB }

func NewConversationRepo(db *gorm.DB) *ConversationRepo { return &ConversationRepo{db: db} }

func (r *ConversationRepo) Create(c *models.Conversation) error { return r.db.Create(c).Error }

func (r *ConversationRepo) List(userID uint) ([]models.Conversation, error) {
	var list []models.Conversation
	return list, r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&list).Error
}

func (r *ConversationRepo) Get(id, userID uint) (*models.Conversation, error) {
	var c models.Conversation
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConversationRepo) UpdateTitle(id uint, title string) error {
	return r.db.Model(&models.Conversation{}).Where("id = ?", id).Update("title", title).Error
}

func (r *ConversationRepo) GetLatest(userID uint) (*models.Conversation, error) {
	var c models.Conversation
	if err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConversationRepo) Touch(id uint) error {
	return r.db.Model(&models.Conversation{}).Where("id = ?", id).Update("updated_at", gorm.Expr("datetime('now')")).Error
}

func (r *ConversationRepo) Delete(id, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Conversation{}).Error
}
