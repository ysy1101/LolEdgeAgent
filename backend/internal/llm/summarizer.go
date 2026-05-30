package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// SummarizeArticle 对单篇文章生成 1-3 句摘要
func SummarizeArticle(ctx context.Context, c *Client, title, content string) (string, error) {
	if len(content) > 3000 {
		content = content[:3000]
	}
	sys := `用 1-3 句中/英文总结以下文章（原文是中文就用中文，英文就用英文）。直接输出摘要。`
	user := fmt.Sprintf("标题：%s\n\n内容：%s", title, content)
	return c.Chat(ctx, sys, user)
}

// AssembleBriefing 将已排序+摘要的文章组装为 Markdown 简报
func AssembleBriefing(ctx context.Context, c *Client, articlesJSON, interests string) (string, error) {
	sys := `你是内容简报编辑。根据提供的文章列表和用户兴趣，生成 Markdown 格式简报。
要求：
1. 开头概述今日热点（中文）
2. 按重要程度分成 2-3 个 ## 分类版块
3. 每篇文章：标题（带原文链接）、一句话摘要、来源标签
4. 结尾标注收录文章数

直接输出 Markdown，不要额外说明。`

	user := fmt.Sprintf("用户兴趣：%s\n\n文章列表：\n%s", interests, articlesJSON)
	return c.Chat(ctx, sys, user)
}

// SummarizeBatch 批量生成文章摘要（顺序执行）
func SummarizeBatch(ctx context.Context, c *Client, articles []SummaryInput, logger *slog.Logger) []string {
	results := make([]string, len(articles))
	for i, a := range articles {
		summary, err := SummarizeArticle(ctx, c, a.Title, a.Content)
		if err != nil {
			logger.Warn("summarize article failed", "title", a.Title, "error", err)
			results[i] = a.Description // fallback
		} else {
			results[i] = strings.TrimSpace(summary)
		}
	}
	return results
}

// SummaryInput 摘要输入
type SummaryInput struct {
	ID          uint
	Title       string
	Content     string
	Description string
	URL         string
}
