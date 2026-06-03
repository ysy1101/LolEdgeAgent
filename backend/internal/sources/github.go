package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"loledgeagent/internal/models"

	"github.com/PuerkitoBio/goquery"
)

func init() { Register(&GitHubPlugin{}) }

type GitHubPlugin struct{}

func (p *GitHubPlugin) Name() string { return "github" }

func (p *GitHubPlugin) Validate(source models.Source) error {
	if source.URL == "" {
		return fmt.Errorf("GitHub Trending URL 不能为空")
	}
	return nil
}

type ghConfig struct {
	Language       string `json:"language"`
	Since          string `json:"since"`           // daily / weekly / monthly
	SpokenLanguage string `json:"spoken_language"` // zh / en / ""
	MaxItems       int    `json:"max_items"`
}

func (p *GitHubPlugin) Fetch(ctx context.Context, source models.Source) ([]models.Article, error) {
	cfg := ghConfig{Since: "daily", MaxItems: 25}
	if source.ConfigJSON != "" {
		_ = json.Unmarshal([]byte(source.ConfigJSON), &cfg)
	}
	if cfg.Since == "" {
		cfg.Since = "daily"
	}
	if cfg.MaxItems <= 0 || cfg.MaxItems > 25 {
		cfg.MaxItems = 25
	}

	// 构造 URL: https://github.com/trending/{lang}?since={since}
	pageURL := fmt.Sprintf("%s/trending/%s?since=%s", source.URL, cfg.Language, cfg.Since)
	if cfg.SpokenLanguage != "" {
		pageURL += "&spoken_language_code=" + cfg.SpokenLanguage
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	req.Header.Set("User-Agent", "LolEdgeAgent/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub 返回 %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %w", err)
	}

	var articles []models.Article
	now := time.Now()

	doc.Find("article.Box-row").Each(func(i int, s *goquery.Selection) {
		if i >= cfg.MaxItems {
			return
		}

		// repo 名称
		nameSel := s.Find("h2.h3 a")
		href, _ := nameSel.Attr("href")
		fullName := strings.TrimPrefix(strings.TrimSpace(href), "/")
		repoURL := "https://github.com/" + fullName

		// 描述
		desc := strings.TrimSpace(s.Find("p.col-9").First().Text())

		// 语言
		lang := strings.TrimSpace(s.Find("[itemprop='programmingLanguage']").Text())

		// star 和 fork 数字
		starsToday := ""
		s.Find("span.d-inline-block").Each(func(_ int, sp *goquery.Selection) {
			text := strings.TrimSpace(sp.Text())
			if strings.Contains(text, "stars today") {
				starsToday = strings.TrimSpace(strings.Split(text, " ")[0])
			}
		})

		// 描述构建
		fullDesc := fmt.Sprintf("Language: %s", lang)
		if starsToday != "" {
			fullDesc += fmt.Sprintf(" | Stars today: %s", starsToday)
		}
		if desc != "" {
			fullDesc += " | " + desc
		}

		articles = append(articles, models.Article{
			SourceID:    source.ID,
			ExternalID:  "github/" + fullName,
			Title:       fmt.Sprintf("%s - %s", fullName, desc),
			URL:         repoURL,
			Description: fullDesc,
			PublishedAt: &now,
			FetchedAt:   now,
			DedupHash:   strHash("github/" + fullName),
		})
	})

	return articles, nil
}

