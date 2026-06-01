package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct{ db *gorm.DB }

func NewHealthHandler(db *gorm.DB) *HealthHandler { return &HealthHandler{db: db} }

func (h *HealthHandler) Check(c *gin.Context) {
	dbStatus := "ok"
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		dbStatus = "error"
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"status": "ok", "db": dbStatus, "llm": "pending"},
	})
}
