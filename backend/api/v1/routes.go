package v1

import (
	"log/slog"
	"net/http"

	"loledgeagent/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger) {
	r.Use(middleware.CORS("http://localhost:5173"))

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			sqlDB, _ := db.DB()
			dbStatus := "ok"
			if sqlDB != nil {
				dbStatus = "ok"
			}
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data": gin.H{
					"status": "ok",
					"db":     dbStatus,
					"llm":    "pending",
				},
			})
		})
	}

	_ = logger // TODO: use logger in handler wiring
}
