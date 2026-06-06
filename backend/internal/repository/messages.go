package repository

import (
	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type MessageRepo struct{ db *gorm.DB }

func NewMessageRepo(db *gorm.DB) *MessageRepo { return &MessageRepo{db: db} }

func (r *MessageRepo) Create(msg *models.ChatMessage) error { return r.db.Create(msg).Error }

func (r *MessageRepo) ListByConversation(conversationID uint) ([]models.ChatMessage, error) {
	var list []models.ChatMessage
	return list, r.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").Find(&list).Error
}
