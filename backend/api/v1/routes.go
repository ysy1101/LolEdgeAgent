package v1

import (
	"log/slog"

	"loledgeagent/internal/handler"
	"loledgeagent/internal/llm"
	"loledgeagent/internal/middleware"
	"loledgeagent/internal/pipeline"
	"loledgeagent/internal/repository"
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

	// LLM client（如果未配置 API Key，client 为 nil，管线走降级逻辑）
	llmCfg := llm.LoadConfig()
	var llmClient *llm.Client
	if llmCfg.APIKey != "" {
		llmClient = llm.NewClient(llmCfg)
	}

	engine := pipeline.NewEngine(articleRepo, briefingRepo, prefRepo, llmClient, logger)
	briefingSvc := service.NewBriefingService(db, fetchSvc, engine, logger)

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
	}
}
