package llm

import (
	"context"
	"fmt"
	"log/slog"
)

// ScoredArticle LLM 排序后的文章评分
type ScoredArticle struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale"`
}

// RankArticles 用 LLM 对文章按用户兴趣打分排序
func RankArticles(ctx context.Context, c *Client, articlesJSON, interests string) ([]ScoredArticle, error) {
	sys := `你是一个内容相关性评分员。给定用户的关注领域（关键词）和一系列文章，对每篇文章从 0.0 到 1.0 进行相关性打分。
评分标准：专业度（0~0.4）、新鲜度（0~0.3）、匹配度（0~0.3）。
返回 JSON 数组：[{id, title, score, rationale}]，只返回 JSON。`

	user := fmt.Sprintf("用户关注：%s\n\n文章列表：\n%s", interests, articlesJSON)

	var result []ScoredArticle
	if err := c.ChatJSON(ctx, sys, user, &result); err != nil {
		return nil, fmt.Errorf("ranking failed: %w", err)
	}
	return result, nil
}

// RankArticlesWithLog 带日志的排名调用
func RankArticlesWithLog(ctx context.Context, c *Client, articlesJSON, interests string, logger *slog.Logger) ([]ScoredArticle, error) {
	logger.Info("ranking start", "article_count", len(articlesJSON))
	result, err := RankArticles(ctx, c, articlesJSON, interests)
	if err != nil {
		logger.Error("ranking failed", "error", err)
		return nil, err
	}
	logger.Info("ranking done", "scored_count", len(result))
	return result, nil
}
