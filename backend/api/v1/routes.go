package v1

import (
	"net/http"

	"loledgeagent/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(middleware.CORS("http://localhost:5173"))

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "success",
				"data": gin.H{
					"status": "ok",
					"db":     "pending",
					"llm":    "pending",
				},
			})
		})
	}
}
