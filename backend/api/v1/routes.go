package v1

import (
	"log/slog"

	"loledgeagent/internal/agent"
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
		// 健康检查（公开）
		healthH := handler.NewHealthHandler(db)
		api.GET("/health", healthH.Check)

		// 认证（公开）
		authH := handler.NewAuthHandler(db)
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)

		// token 验证（需要登录）
		api.GET("/auth/verify", middleware.AuthRequired(), authH.Verify)
	}

	// 以下需要登录
	protected := api.Group("", middleware.AuthRequired())
	{
		// 源管理
		sourceH := handler.NewSourceHandler(sourceRepo, fetchSvc)
		protected.GET("/sources", sourceH.List)
		protected.POST("/sources", sourceH.Create)
		protected.GET("/sources/:id", sourceH.Get)
		protected.PUT("/sources/:id", sourceH.Update)
		protected.DELETE("/sources/:id", sourceH.Delete)
		protected.POST("/sources/:id/fetch", sourceH.Fetch)

		// 文章
		articleH := handler.NewArticleHandler(articleRepo, fetchSvc)
		protected.GET("/articles", articleH.List)
		protected.POST("/articles/fetch", articleH.FetchAll)

		// 简报
		briefingH := handler.NewBriefingHandler(briefingSvc)
		protected.GET("/briefings", briefingH.List)
		protected.POST("/briefings/generate", briefingH.Generate)
		protected.GET("/briefings/:id", briefingH.Get)
		protected.DELETE("/briefings/:id", briefingH.Delete)

		// 对话管理
		msgRepo := repository.NewMessageRepo(db)
		convRepo := repository.NewConversationRepo(db)
		ragSvc := service.NewRAGService(embRepo, articleRepo, llmClient, logger)
		convH := handler.NewConversationHandler(convRepo, msgRepo, ragSvc)
		protected.POST("/conversations", convH.Create)
		protected.GET("/conversations", convH.List)
		protected.GET("/conversations/:id/messages", convH.GetMessages)
		protected.POST("/conversations/:id/messages", convH.AddMessage)
		protected.DELETE("/conversations/:id", convH.Delete)

		// 偏好设置
		prefH := handler.NewPreferenceHandler(prefRepo)
		protected.GET("/preferences", prefH.Get)
		protected.PUT("/preferences", prefH.Update)

		// Agent 工具注册
		agent.RegisterAllTools(articleRepo, briefingSvc, ragSvc, prefRepo, briefingRepo)

		// Agent 对话
		aiAgent := agent.New(llmClient, logger)
		protected.POST("/agent/chat", func(c *gin.Context) {
			var body struct {
				Message string           `json:"message"`
				History []agent.Message  `json:"history"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(400, gin.H{"code": 400, "message": err.Error()})
				return
			}
			reply, err := aiAgent.Run(c.Request.Context(), body.History, body.Message)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": reply})
		})

		// RAG 问答（需要 LLM key）
		ragSvc = service.NewRAGService(embRepo, articleRepo, llmClient, logger)
		protected.POST("/search", func(c *gin.Context) {
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
