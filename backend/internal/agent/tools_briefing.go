package agent

import (
	"context"

	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

// ---- generate_briefing ----

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

// ---- list_briefings ----

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

// ---- get_briefing ----

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
			b, err := briefingRepo.GetByID(input.ID, getUserID(ctx))
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
