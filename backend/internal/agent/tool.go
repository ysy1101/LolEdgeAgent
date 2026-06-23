package agent

import (
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/cloudwego/eino/components/tool"
)

var (
	_articleRepo  *repository.ArticleRepo
	_briefingSvc  *service.BriefingService
	_ragSvc       *service.RAGService
	_prefRepo     *repository.PreferenceRepo
	_briefingRepo *repository.BriefingRepo
)

// RegisterAllTools 注册工具依赖（由路由层调用）
func RegisterAllTools(
	articleRepo *repository.ArticleRepo,
	briefingSvc *service.BriefingService,
	ragSvc *service.RAGService,
	prefRepo *repository.PreferenceRepo,
	briefingRepo *repository.BriefingRepo,
) {
	_articleRepo = articleRepo
	_briefingSvc = briefingSvc
	_ragSvc = ragSvc
	_prefRepo = prefRepo
	_briefingRepo = briefingRepo
}

// allEinoTools 创建当前注册的所有工具
func allEinoTools() []tool.InvokableTool {
	return []tool.InvokableTool{
		makeSearchArticlesTool(_ragSvc),
		makeGenerateBriefingTool(_briefingSvc),
		makeListBriefingsTool(_briefingRepo),
		makeGetBriefingTool(_briefingRepo),
		makeGetPreferencesTool(_prefRepo),
	}
}
