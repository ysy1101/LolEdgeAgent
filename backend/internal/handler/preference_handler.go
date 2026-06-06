package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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
	current, err := h.repo.Get(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 如果前端传来的 api_key 包含 ****，说明是脱敏值，不更新
	if v, ok := input["llm_api_key"].(string); ok && strings.Contains(v, "****") {
		delete(input, "llm_api_key")
	}

	// 转回 struct 更新
	data, _ := json.Marshal(input)
	json.Unmarshal(data, current)

	if err := h.repo.Update(userID, current); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// 返回时脱敏
	if len(current.LLMAPIKey) > 4 {
		current.LLMAPIKey = current.LLMAPIKey[:4] + "****"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": current})
}
