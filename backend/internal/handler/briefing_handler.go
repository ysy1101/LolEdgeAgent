package handler

import (
	"net/http"
	"strconv"

	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
)

type BriefingHandler struct {
	svc *service.BriefingService
}

func NewBriefingHandler(svc *service.BriefingService) *BriefingHandler {
	return &BriefingHandler{svc: svc}
}

func (h *BriefingHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	list, total, err := h.svc.List(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
		"items": list, "total": total, "page": page, "page_size": limit,
	}})
}

func (h *BriefingHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}
	b, err := h.svc.Get(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "briefing not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": b})
}

func (h *BriefingHandler) Generate(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	id, err := h.svc.GenerateAsync(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "success", "data": gin.H{"briefing_id": id}})
}

func (h *BriefingHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}
