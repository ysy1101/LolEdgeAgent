package sources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"loledgeagent/internal/models"

	"github.com/mmcdole/gofeed"
)

func init() { Register(&RSSPlugin{}) }

type RSSPlugin struct{}

func (p *RSSPlugin) Name() string { return "rss" }

func (p *RSSPlugin) Validate(source models.Source) error {
	if source.URL == "" {
		return fmt.Errorf("RSS 源 URL 不能为空")
	}
	return nil
}

func (p *RSSPlugin) Fetch(ctx context.Context, source models.Source) ([]models.Article, error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "LolEdgeAgent/1.0"

	feed, err := fp.ParseURL(source.URL)
	if err != nil {
		return nil, fmt.Errorf("解析 RSS 失败: %w", err)
	}

	articles := make([]models.Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		link := firstNonEmpty(item.Link, item.GUID)
		if link == "" {
			continue
		}

		var publishedAt *time.Time
		if item.PublishedParsed != nil {
			t := item.PublishedParsed.In(time.Local)
			publishedAt = &t
		}

		articles = append(articles, models.Article{
			SourceID:    source.ID,
			ExternalID:  strHash(link),
			Title:       strings.TrimSpace(item.Title),
			URL:         link,
			Description: strings.TrimSpace(item.Description),
			Content:     strings.TrimSpace(firstNonEmpty(item.Content, item.Description)),
			Author:      authorName(item.Author),
			PublishedAt: publishedAt,
			FetchedAt:   time.Now(),
			DedupHash:   strHash(strings.ToLower(strings.TrimSpace(item.Title)) + link),
		})
	}
	return articles, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func strHash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

func authorName(a *gofeed.Person) string {
	if a == nil {
		return ""
	}
	return a.Name
}
