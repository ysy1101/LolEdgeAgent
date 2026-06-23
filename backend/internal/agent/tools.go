package agent

import (
	"context"

	"loledgeagent/internal/middleware"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// ---- 1. search_articles ----

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

// ---- 2. generate_briefing ----

type GenerateBriefingOutput struct {
	BriefingID uint   `json:"briefing_id"`
	Status     string `json:"status"`
}

func makeGenerateBriefingTool(briefingSvc *service.BriefingService) tool.InvokableTool {
	type emptyInput struct{}
	t, _ := toolutils.InferTool(
		"generate_briefing",
		"生成一份新的内容简报。返回简报ID和状态。",
		func(ctx context.Context, _ emptyInput) (GenerateBriefingOutput, error) {
			id, err := briefingSvc.GenerateAsync(ctx, getUserID(ctx))
			if err != nil {
				return GenerateBriefingOutput{}, err
			}
			return GenerateBriefingOutput{BriefingID: id, Status: "generating"}, nil
		},
	)
	return t
}

// ---- 3. list_briefings ----

type ListBriefingsInput struct {
	Limit int `json:"limit" jsonschema_description:"返回数量，默认5"`
}

type BriefingSimple struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	ArticleCount int    `json:"article_count"`
	Status       string `json:"status"`
}

type ListBriefingsOutput struct {
	Briefings []BriefingSimple `json:"briefings"`
}

func makeListBriefingsTool(briefingRepo *repository.BriefingRepo) tool.InvokableTool {
	t, _ := toolutils.InferTool(
		"list_briefings",
		"查看最近生成的简报列表。",
		func(ctx context.Context, input ListBriefingsInput) (ListBriefingsOutput, error) {
			if input.Limit <= 0 {
				input.Limit = 5
			}
			list, _, err := briefingRepo.List(getUserID(ctx), 1, input.Limit)
			if err != nil {
				return ListBriefingsOutput{}, err
			}
			var out []BriefingSimple
			for _, b := range list {
				out = append(out, BriefingSimple{ID: b.ID, Title: b.Title, ArticleCount: b.ArticleCount, Status: b.Status})
			}
			return ListBriefingsOutput{Briefings: out}, nil
		},
	)
	return t
}

// ---- 4. get_briefing ----

type GetBriefingInput struct {
	ID uint `json:"id" jsonschema:"required" jsonschema_description:"简报ID"`
}

type GetBriefingOutput struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	Markdown     string `json:"markdown"`
	ArticleCount int    `json:"article_count"`
	Status       string `json:"status"`
}

func makeGetBriefingTool(briefingRepo *repository.BriefingRepo) tool.InvokableTool {
	t, _ := toolutils.InferTool(
		"get_briefing",
		"查看某份简报的详细内容。",
		func(ctx context.Context, input GetBriefingInput) (GetBriefingOutput, error) {
			b, err := briefingRepo.GetByID(input.ID)
			if err != nil {
				return GetBriefingOutput{}, err
			}
			return GetBriefingOutput{
				ID: b.ID, Title: b.Title, Markdown: b.ContentMarkdown,
				ArticleCount: b.ArticleCount, Status: b.Status,
			}, nil
		},
	)
	return t
}

// ---- 5. get_preferences ----

type GetPreferencesOutput struct {
	Keywords             string `json:"keywords"`
	MaxBriefingArticles  int    `json:"max_briefing_articles"`
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
