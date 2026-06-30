package handler

import (
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
)

type RAGHandler struct {
	ragSvc *service.RAGService
}

func NewRAGHandler(ragSvc *service.RAGService) *RAGHandler {
	return &RAGHandler{ragSvc: ragSvc}
}

func (h *RAGHandler) Search(c *gin.Context) {
	var body struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if body.TopK <= 0 {
		body.TopK = 5
	}
	articles, err := h.ragSvc.Search(c.Request.Context(), body.Query, body.TopK)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": articles})
}

func (h *RAGHandler) Ask(c *gin.Context) {
	var body struct {
		Question string `json:"question"`
		TopK     int    `json:"top_k"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if body.TopK <= 0 {
		body.TopK = 5
	}
	answer, articles, err := h.ragSvc.Ask(c.Request.Context(), body.Question, body.TopK)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": gin.H{
		"answer":   answer,
		"articles": articles,
	}})
}
