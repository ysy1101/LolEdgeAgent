package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"loledgeagent/internal/middleware"
	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger) {
	r.Use(middleware.CORS("http://localhost:5173"))

	// 初始化 repository 和 service
	sourceRepo := repository.NewSourceRepo(db)
	articleRepo := repository.NewArticleRepo(db)
	logRepo := repository.NewFetchLogRepo(db)
	prefRepo := repository.NewPreferenceRepo(db)
	briefingRepo := repository.NewBriefingRepo(db)

	fetchSvc := service.NewFetchService(sourceRepo, articleRepo, logRepo, logger)
	_ = fetchSvc
	_ = briefingRepo
	_ = prefRepo

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

		// === 临时测试路由，Step 6 会用正式 handler 替换 ===

		// 源管理
		api.GET("/sources", func(c *gin.Context) {
			list, err := sourceRepo.List(false)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": list})
		})

		api.POST("/sources", func(c *gin.Context) {
			var s models.Source
			if err := c.ShouldBindJSON(&s); err != nil {
				c.JSON(400, gin.H{"code": 400, "message": err.Error()})
				return
			}
			if err := sourceRepo.Create(&s); err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(201, gin.H{"code": 0, "message": "success", "data": s})
		})

		// 抓取
		api.POST("/fetch", func(c *gin.Context) {
			articles, err := fetchSvc.FetchAll(c.Request.Context())
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": gin.H{
				"articles_fetched": len(articles),
			}})
		})

		// 文章列表
		api.GET("/articles", func(c *gin.Context) {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			sourceID, _ := strconv.Atoi(c.DefaultQuery("source_id", "0"))
			list, total, err := articleRepo.List(uint(sourceID), page, limit)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": gin.H{
				"items": list, "total": total, "page": page, "page_size": limit,
			}})
		})
	}
}
