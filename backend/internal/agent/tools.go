package agent

import (
	"context"
	"fmt"

	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"
)

// RegisterAllTools 注册所有工具，需要在 DI 完成后调用
func RegisterAllTools(
	articleRepo *repository.ArticleRepo,
	briefingSvc *service.BriefingService,
	ragSvc *service.RAGService,
	prefRepo *repository.PreferenceRepo,
	briefingRepo *repository.BriefingRepo,
) {
	// 1. 搜索文章
	Register(&Tool{
		Name:        "search_articles",
		Description: "根据关键词搜索已采集的文章库。用中文关键词。",
		Parameters: map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索关键词"},
			"limit": map[string]any{"type": "integer", "description": "返回数量，默认5"},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			query := getStr(args, "query")
			limit := getInt(args, "limit", 5)
			if query == "" {
				return "[]", fmt.Errorf("query required")
			}
			articles, err := ragSvc.Search(ctx, query, limit)
			if err != nil {
				return "", err
			}
			type simple struct {
				ID    uint   `json:"id"`
				Title string `json:"title"`
				Desc  string `json:"desc"`
				URL   string `json:"url"`
			}
			var list []simple
			for _, a := range articles {
				list = append(list, simple{ID: a.ID, Title: a.Title, Desc: a.Description, URL: a.URL})
			}
			return toolJSON(list), nil
		},
	})

	// 2. 生成简报
	Register(&Tool{
		Name:        "generate_briefing",
		Description: "生成一份新的内容简报。返回简报ID和摘要。",
		Parameters:  map[string]any{},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			id, err := briefingSvc.GenerateAsync(ctx, 1) // userID 从 ctx 取，暂时硬编码
			if err != nil {
				return "", err
			}
			return toolJSON(map[string]any{"briefing_id": id, "status": "generating"}), nil
		},
	})

	// 3. 列出简报
	Register(&Tool{
		Name:        "list_briefings",
		Description: "查看最近生成的简报列表。",
		Parameters: map[string]any{
			"limit": map[string]any{"type": "integer", "description": "返回数量，默认5"},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			limit := getInt(args, "limit", 5)
			list, _, err := briefingRepo.List(1, 1, limit)
			if err != nil {
				return "", err
			}
			type simple struct {
				ID           uint   `json:"id"`
				Title        string `json:"title"`
				ArticleCount int    `json:"article_count"`
				Status       string `json:"status"`
			}
			var out []simple
			for _, b := range list {
				out = append(out, simple{ID: b.ID, Title: b.Title, ArticleCount: b.ArticleCount, Status: b.Status})
			}
			return toolJSON(out), nil
		},
	})

	// 4. 查看简报
	Register(&Tool{
		Name:        "get_briefing",
		Description: "查看某份简报的详细内容。",
		Parameters: map[string]any{
			"id": map[string]any{"type": "integer", "description": "简报ID"},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			id := uint(getInt(args, "id", 0))
			if id == 0 {
				return "", fmt.Errorf("简报ID必填")
			}
			b, err := briefingRepo.GetByID(id)
			if err != nil {
				return "", err
			}
			return toolJSON(map[string]any{
				"id":       b.ID,
				"title":    b.Title,
				"markdown": b.ContentMarkdown,
				"count":    b.ArticleCount,
				"status":   b.Status,
			}), nil
		},
	})

	// 5. 查看偏好
	Register(&Tool{
		Name:        "get_preferences",
		Description: "查看当前用户的偏好设置。",
		Parameters:  map[string]any{},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := prefRepo.Get(1)
			if err != nil {
				return "", err
			}
			return toolJSON(map[string]any{
				"keywords":              p.Keywords,
				"max_briefing_articles": p.MaxBriefingArticles,
			}), nil
		},
	})
}

func getStr(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func getInt(args map[string]any, key string, def int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return def
}
