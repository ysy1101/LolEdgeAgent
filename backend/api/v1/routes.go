package v1

import (
	"log/slog"

	"loledgeagent/internal/handler"
	"loledgeagent/internal/llm"
	"loledgeagent/internal/middleware"
	"loledgeagent/internal/pipeline"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/scheduler"
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger) {
	r.Use(middleware.CORS("http://localhost:5173"))

	// ---- 依赖注入 ----
	sourceRepo := repository.NewSourceRepo(db)
	articleRepo := repository.NewArticleRepo(db)
	briefingRepo := repository.NewBriefingRepo(db)
	prefRepo := repository.NewPreferenceRepo(db)
	logRepo := repository.NewFetchLogRepo(db)

	fetchSvc := service.NewFetchService(sourceRepo, articleRepo, logRepo, logger)
	embRepo := repository.NewEmbeddingRepo(db)

	// LLM client（如果未配置 API Key，client 为 nil，管线走降级逻辑）
	llmCfg := llm.LoadConfig()
	defaultPref, _ := prefRepo.Get(1)
	if llmCfg.APIKey == "" && defaultPref != nil && defaultPref.LLMAPIKey != "" {
		llmCfg.APIKey = defaultPref.LLMAPIKey
		llmCfg.BaseURL = defaultPref.LLMBaseURL
		llmCfg.Model = defaultPref.LLMModel
	}
	if defaultPref != nil && defaultPref.LLMAPIKey == "" && llmCfg.APIKey != "" {
		defaultPref.LLMAPIKey = llmCfg.APIKey
		defaultPref.LLMBaseURL = llmCfg.BaseURL
		defaultPref.LLMModel = llmCfg.Model
		_ = prefRepo.Update(1, defaultPref)
	}
	var llmClient *llm.Client
	if llmCfg.APIKey != "" {
		llmClient = llm.NewClient(llmCfg)
	}

	engine := pipeline.NewEngine(articleRepo, briefingRepo, prefRepo, llmClient, logger)
	briefingSvc := service.NewBriefingService(db, fetchSvc, articleRepo, engine, logger)

	// 启动定时调度器
	sched := scheduler.New(briefingSvc, logger)
	pref, _ := prefRepo.Get(1)
	if pref != nil && pref.BriefingSchedule != "" {
		if err := sched.Start(pref.BriefingSchedule); err != nil {
			logger.Error("scheduler start failed", "error", err)
		}
	}

	// ---- 路由 ----
	api := r.Group("/api/v1")
	{
		// 健康检查
		healthH := handler.NewHealthHandler(db)
		api.GET("/health", healthH.Check)

		// 源管理
		sourceH := handler.NewSourceHandler(sourceRepo, fetchSvc)
		api.GET("/sources", sourceH.List)
		api.POST("/sources", sourceH.Create)
		api.GET("/sources/:id", sourceH.Get)
		api.PUT("/sources/:id", sourceH.Update)
		api.DELETE("/sources/:id", sourceH.Delete)
		api.POST("/sources/:id/fetch", sourceH.Fetch)

		// 文章
		articleH := handler.NewArticleHandler(articleRepo, fetchSvc)
		api.GET("/articles", articleH.List)
		api.POST("/articles/fetch", articleH.FetchAll)

		// 简报
		briefingH := handler.NewBriefingHandler(briefingSvc)
		api.GET("/briefings", briefingH.List)
		api.POST("/briefings/generate", briefingH.Generate)
		api.GET("/briefings/:id", briefingH.Get)
		api.DELETE("/briefings/:id", briefingH.Delete)

		// 偏好设置
		prefH := handler.NewPreferenceHandler(prefRepo)
		api.GET("/preferences", prefH.Get)
		api.PUT("/preferences", prefH.Update)

		// RAG 问答（需要 LLM key）
		ragSvc := service.NewRAGService(embRepo, articleRepo, llmClient, logger)
		api.POST("/search", func(c *gin.Context) {
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
			articles, err := ragSvc.Search(c.Request.Context(), body.Query, body.TopK)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": articles})
		})
		api.POST("/ask", func(c *gin.Context) {
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
			answer, articles, err := ragSvc.Ask(c.Request.Context(), body.Question, body.TopK)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": gin.H{
				"answer":   answer,
				"articles": articles,
			}})
		})
	}
}
