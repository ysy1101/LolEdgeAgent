package handler

import (
	"strconv"

	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	convRepo *repository.ConversationRepo
	msgRepo  *repository.MessageRepo
	ragSvc   *service.RAGService
}

func NewConversationHandler(
	convRepo *repository.ConversationRepo,
	msgRepo *repository.MessageRepo,
	ragSvc *service.RAGService,
) *ConversationHandler {
	return &ConversationHandler{convRepo: convRepo, msgRepo: msgRepo, ragSvc: ragSvc}
}

func (h *ConversationHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")
	conv := &models.Conversation{UserID: userID}
	if err := h.convRepo.Create(conv); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(201, gin.H{"code": 0, "message": "success", "data": conv})
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")
	list, err := h.convRepo.List(userID)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if list == nil {
		list = []models.Conversation{}
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": list})
}

func (h *ConversationHandler) GetMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	conv, err := h.convRepo.Get(uint(id), userID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "对话不存在"})
		return
	}
	_ = conv

	msgs, err := h.msgRepo.ListByConversation(uint(id))
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if msgs == nil {
		msgs = []models.ChatMessage{}
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": msgs})
}

type saveMsgReq struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *ConversationHandler) AddMessage(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	_, err := h.convRepo.Get(uint(id), userID)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "对话不存在"})
		return
	}

	var req saveMsgReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}

	msg := &models.ChatMessage{
		ConversationID: uint(id),
		Role:           req.Role,
		Content:        req.Content,
	}
	if err := h.msgRepo.Create(msg); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.convRepo.Touch(uint(id))
	c.JSON(201, gin.H{"code": 0, "message": "success", "data": msg})
}

func (h *ConversationHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.convRepo.Delete(uint(id), userID); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": nil})
}
