package v1

import (
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

	sched := scheduler.New(briefingSvc, logger)
	if pref != nil && pref.BriefingSchedule != "" {
		if err := sched.Start(pref.BriefingSchedule); err != nil {
			logger.Error("scheduler start failed", "error", err)
		}
	}

	msgRepo := repository.NewMessageRepo(db)
	memoryRepo := repository.NewMemoryRepo(db)
	memorySvc := service.NewMemoryService(memoryRepo, msgRepo, llmClient, logger)

	// Agent 工具 + 引擎
	agent.RegisterAllTools(articleRepo, briefingSvc, ragSvc, prefRepo, briefingRepo)
	aiAgent := agent.New(llmCfg, prefRepo, memorySvc, logger)

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

		convRepo := repository.NewConversationRepo(db)
		convH := handler.NewConversationHandler(convRepo, msgRepo, ragSvc, memorySvc, logger)
		protected.POST("/conversations", convH.Create)
		protected.GET("/conversations", convH.List)
		protected.GET("/conversations/:id/messages", convH.GetMessages)
		protected.POST("/conversations/:id/messages", convH.AddMessage)
		protected.DELETE("/conversations/:id", convH.Delete)

		prefH := handler.NewPreferenceHandler(prefRepo)
		protected.GET("/preferences", prefH.Get)
		protected.PUT("/preferences", prefH.Update)

		agentH := handler.NewAgentHandler(aiAgent)
		protected.POST("/agent/chat", agentH.Chat)

		ragH := handler.NewRAGHandler(ragSvc)
		protected.POST("/search", ragH.Search)
		protected.POST("/ask", ragH.Ask)
	}
}
