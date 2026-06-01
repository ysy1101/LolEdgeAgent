package handler

import (
	"net/http"

	"loledgeagent/internal/repository"

	"github.com/gin-gonic/gin"
)

type PreferenceHandler struct {
	repo *repository.PreferenceRepo
}

func NewPreferenceHandler(repo *repository.PreferenceRepo) *PreferenceHandler {
	return &PreferenceHandler{repo: repo}
}

func (h *PreferenceHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}
	p, err := h.repo.Get(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// 返回时隐藏 API Key 明文
	if len(p.LLMAPIKey) > 4 {
		masked := p.LLMAPIKey[:4] + "****"
		p.LLMAPIKey = masked
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": p})
}

func (h *PreferenceHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}
	p, err := h.repo.Get(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := c.ShouldBindJSON(p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.repo.Update(userID, p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": p})
}
