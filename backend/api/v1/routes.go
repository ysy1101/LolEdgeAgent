package v1

import (
	"context"
	"log/slog"

	"loledgeagent/internal/agent"
	"loledgeagent/internal/config"
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

func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *slog.Logger, cfg *config.Config) {
	r.Use(middleware.CORS(cfg.Server.CorsOrigin))

	// ---- 依赖注入 ----
	sourceRepo := repository.NewSourceRepo(db)
	articleRepo := repository.NewArticleRepo(db)
	briefingRepo := repository.NewBriefingRepo(db)
	prefRepo := repository.NewPreferenceRepo(db)
	logRepo := repository.NewFetchLogRepo(db)

	embRepo := repository.NewEmbeddingRepo(db)

	// LLM client（环境变量优先，DB 补充缺失项）
	llmCfg := llm.LoadConfig()
	pref, _ := prefRepo.Get(1)

	if pref != nil {
		if llmCfg.APIKey == "" && pref.LLMAPIKey != "" {
			llmCfg.APIKey = pref.LLMAPIKey
		}
		if llmCfg.Model == "" || llmCfg.Model == "deepseek-chat" {
			if pref.LLMModel != "" {
				llmCfg.Model = pref.LLMModel
			}
		}
		if llmCfg.BaseURL == "" && pref.LLMBaseURL != "" {
			llmCfg.BaseURL = pref.LLMBaseURL
		}
	}
	var llmClient *llm.Client
	llm.SetupGlobalCallbacks(logger)

	if llmCfg.APIKey != "" {
		llmClient = llm.NewClient(llmCfg)
	}
	ragSvc := service.NewRAGService(embRepo, articleRepo, llmClient, logger)

	fetchSvc := service.NewFetchService(sourceRepo, articleRepo, logRepo, ragSvc, logger)

	engine := pipeline.NewEngine(articleRepo, briefingRepo, prefRepo, llmClient, logger)
	briefingSvc := service.NewBriefingService(db, fetchSvc, articleRepo, engine, logger)

	// 启动定时调度器
	sched := scheduler.New(briefingSvc, logger)
	if pref != nil && pref.BriefingSchedule != "" {
		if err := sched.Start(pref.BriefingSchedule); err != nil {
			logger.Error("scheduler start failed", "error", err)
		}
	}

	// ---- 路由 ----
	api := r.Group("/api/v1")
	{
		healthH := handler.NewHealthHandler(db)
		api.GET("/health", healthH.Check)

		authH := handler.NewAuthHandler(db)
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.GET("/auth/verify", middleware.AuthRequired(), authH.Verify)
	}

	protected := api.Group("", middleware.AuthRequired())
	{
		sourceH := handler.NewSourceHandler(sourceRepo, fetchSvc)
		protected.GET("/sources", sourceH.List)
		protected.POST("/sources", sourceH.Create)
		protected.GET("/sources/:id", sourceH.Get)
		protected.PUT("/sources/:id", sourceH.Update)
		protected.DELETE("/sources/:id", sourceH.Delete)
		protected.POST("/sources/:id/fetch", sourceH.Fetch)

		articleH := handler.NewArticleHandler(articleRepo, fetchSvc)
		protected.GET("/articles", articleH.List)
		protected.POST("/articles/fetch", articleH.FetchAll)

		briefingH := handler.NewBriefingHandler(briefingSvc)
		protected.GET("/briefings", briefingH.List)
		protected.POST("/briefings/generate", briefingH.Generate)
		protected.GET("/briefings/:id", briefingH.Get)
		protected.DELETE("/briefings/:id", briefingH.Delete)

		msgRepo := repository.NewMessageRepo(db)
		convRepo := repository.NewConversationRepo(db)
		convH := handler.NewConversationHandler(convRepo, msgRepo, ragSvc)
		protected.POST("/conversations", convH.Create)
		protected.GET("/conversations", convH.List)
		protected.GET("/conversations/:id/messages", convH.GetMessages)
		protected.POST("/conversations/:id/messages", convH.AddMessage)
		protected.DELETE("/conversations/:id", convH.Delete)

		prefH := handler.NewPreferenceHandler(prefRepo)
		protected.GET("/preferences", prefH.Get)
		protected.PUT("/preferences", prefH.Update)

		// Agent 工具注册
		agent.RegisterAllTools(articleRepo, briefingSvc, ragSvc, prefRepo, briefingRepo)

		// Agent 对话（支持 LLM 配置热加载）
		aiAgent := agent.New(llmCfg, prefRepo, logger)
		protected.POST("/agent/chat", func(c *gin.Context) {
			var body struct {
				Message string          `json:"message"`
				History []agent.Message `json:"history"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(400, gin.H{"code": 400, "message": err.Error()})
				return
			}
			ctx := context.WithValue(c.Request.Context(), middleware.UserIDKey, c.GetUint("user_id"))
			reply, err := aiAgent.Run(ctx, body.History, body.Message)
			if err != nil {
				c.JSON(500, gin.H{"code": 500, "message": err.Error()})
				return
			}
			c.JSON(200, gin.H{"code": 0, "message": "success", "data": reply})
		})

		// RAG 语义搜索
		protected.POST("/search", func(c *gin.Context) {
			if llmClient == nil {
				c.JSON(400, gin.H{"code": 400, "message": "AI 功能未配置，请先在偏好设置中设置 API Key"})
				return
			}
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

		protected.POST("/ask", func(c *gin.Context) {
			if llmClient == nil {
				c.JSON(400, gin.H{"code": 400, "message": "AI 功能未配置，请先在偏好设置中设置 API Key"})
				return
			}
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
