package handler

import (
	"context"

	"loledgeagent/internal/agent"
	"loledgeagent/internal/middleware"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	aiAgent *agent.Agent
}

func NewAgentHandler(aiAgent *agent.Agent) *AgentHandler {
	return &AgentHandler{aiAgent: aiAgent}
}

func (h *AgentHandler) Chat(c *gin.Context) {
	var body struct {
		Message string          `json:"message"`
		History []agent.Message `json:"history"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	ctx := context.WithValue(c.Request.Context(), middleware.UserIDKey, c.GetUint("user_id"))
	reply, err := h.aiAgent.Run(ctx, body.History, body.Message)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": reply})
}
