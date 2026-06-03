package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"loledgeagent/internal/models"
)

func init() { Register(&HackerNewsPlugin{}) }

type HackerNewsPlugin struct{}

func (p *HackerNewsPlugin) Name() string { return "hackernews" }

func (p *HackerNewsPlugin) Validate(source models.Source) error {
	if source.URL == "" {
		return fmt.Errorf("HackerNews API 地址不能为空")
	}
	return nil
}

type hnConfig struct {
	StoryType    string `json:"story_type"`    // top / new / best
	MaxItems     int    `json:"max_items"`     // 默认 30
	FetchContent bool   `json:"fetch_content"` // 是否抓取原文（暂未实现）
}

func (p *HackerNewsPlugin) Fetch(ctx context.Context, source models.Source) ([]models.Article, error) {
	cfg := hnConfig{StoryType: "top", MaxItems: 30}
	if source.ConfigJSON != "" {
		json.Unmarshal([]byte(source.ConfigJSON), &cfg)
	}
	if cfg.StoryType == "" {
		cfg.StoryType = "top"
	}
	if cfg.MaxItems <= 0 || cfg.MaxItems > 100 {
		cfg.MaxItems = 30
	}

	// ① 获取 ID 列表
	listURL := fmt.Sprintf("%s/v0/%sstories.json", source.URL, cfg.StoryType)
	ids, err := fetchIDs(ctx, listURL)
	if err != nil {
		return nil, fmt.Errorf("获取 HN 列表失败: %w", err)
	}

	// ② 截取 top N
	if len(ids) > cfg.MaxItems {
		ids = ids[:cfg.MaxItems]
	}

	// ③ 并发获取详情（信号量限制 5）
	articles := make([]models.Article, 0, len(ids))
	var mu sync.Mutex
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(itemID int) {
			defer wg.Done()
			defer func() { <-sem }()

			hnItem, err := fetchItem(ctx, source.URL, itemID)
			if err != nil {
				return // 忽略单条失败
			}
			if hnItem.URL == "" {
				hnItem.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", itemID)
			}

			t := time.Unix(hnItem.Time, 0)
			articles = append(articles, models.Article{
				SourceID:    source.ID,
				ExternalID:  strconv.Itoa(hnItem.ID),
				Title:       hnItem.Title,
				URL:         hnItem.URL,
				Description: fmt.Sprintf("Score: %d | Comments: %d | by %s", hnItem.Score, hnItem.Descendants, hnItem.By),
				Author:      hnItem.By,
				PublishedAt: &t,
				FetchedAt:   time.Now(),
				DedupHash:   strHash(strconv.Itoa(hnItem.ID)),
			})
		}(id)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return articles, nil
}

// ---- HN API types ----

type hnItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Descendants int    `json:"descendants"`
}

func fetchIDs(ctx context.Context, url string) ([]int, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func fetchItem(ctx context.Context, baseURL string, id int) (*hnItem, error) {
	url := fmt.Sprintf("%s/v0/item/%d.json", baseURL, id)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var item hnItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	item.ID = id
	return &item, nil
}

