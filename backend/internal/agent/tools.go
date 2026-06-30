package agent

import (
	"context"

	"loledgeagent/internal/middleware"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// ---- search_articles ----

type SearchArticlesInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
	Limit int    `json:"limit" jsonschema_description:"返回数量，默认5"`
}

type ArticleSimple struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	URL   string `json:"url"`
}

type SearchArticlesOutput struct {
	Articles []ArticleSimple `json:"articles"`
}

func makeSearchArticlesTool(ragSvc *service.RAGService) tool.InvokableTool {
	t, _ := toolutils.InferTool(
		"search_articles",
		"根据关键词搜索已采集的文章库。用中文关键词。",
		func(ctx context.Context, input SearchArticlesInput) (SearchArticlesOutput, error) {
			if input.Limit <= 0 {
				input.Limit = 5
			}
			articles, err := ragSvc.Search(ctx, input.Query, input.Limit)
			if err != nil {
				return SearchArticlesOutput{}, err
			}
			var list []ArticleSimple
			for _, a := range articles {
				list = append(list, ArticleSimple{ID: a.ID, Title: a.Title, Desc: a.Description, URL: a.URL})
			}
			return SearchArticlesOutput{Articles: list}, nil
		},
	)
	return t
}

// ---- get_preferences ----

type GetPreferencesOutput struct {
	Keywords            string `json:"keywords"`
	MaxBriefingArticles int    `json:"max_briefing_articles"`
}

func makeGetPreferencesTool(prefRepo *repository.PreferenceRepo) tool.InvokableTool {
	type emptyInput struct{}
	t, _ := toolutils.InferTool(
		"get_preferences",
		"查看当前用户的偏好设置（关键词、简报数量等）。",
		func(ctx context.Context, _ emptyInput) (GetPreferencesOutput, error) {
			p, err := prefRepo.Get(getUserID(ctx))
			if err != nil {
				return GetPreferencesOutput{}, err
			}
			return GetPreferencesOutput{
				Keywords:            p.Keywords,
				MaxBriefingArticles: p.MaxBriefingArticles,
			}, nil
		},
	)
	return t
}

func getUserID(ctx context.Context) uint {
	if id, ok := ctx.Value(middleware.UserIDKey).(uint); ok && id > 0 {
		return id
	}
	return 1
}
