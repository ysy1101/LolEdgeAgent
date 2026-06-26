package pipeline

import (
	"strings"
	"testing"
	"time"

	"loledgeagent/internal/llm"
	"loledgeagent/internal/models"
)

func TestDefaultRank(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "Article 1", Description: "Desc 1"},
		{ID: 2, Title: "Article 2", Description: "Desc 2"},
	}

	scored := defaultRank(articles)

	if len(scored) != len(articles) {
		t.Errorf("defaultRank should return same count: got %d, want %d", len(scored), len(articles))
	}

	for _, s := range scored {
		if s.Score != 0.5 {
			t.Errorf("default score should be 0.5, got %f", s.Score)
		}
		if s.Rationale != "默认评分" {
			t.Errorf("default rationale should be '默认评分', got '%s'", s.Rationale)
		}
	}
}

func TestSelectTop(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "A", RelevanceScore: 0.1},
		{ID: 2, Title: "B", RelevanceScore: 0.9},
		{ID: 3, Title: "C", RelevanceScore: 0.5},
	}

	scored := []llm.ScoredArticle{
		{ID: 1, Title: "A", Score: 0.1},
		{ID: 2, Title: "B", Score: 0.9},
		{ID: 3, Title: "C", Score: 0.5},
	}

	engine := &Engine{}
	top := engine.selectTop(articles, scored, 2)

	if len(top) != 2 {
		t.Errorf("should select top 2, got %d", len(top))
	}

	// 应该按分数降序：B(0.9) 第一，C(0.5) 第二
	if top[0].ID != 2 {
		t.Errorf("first should be B (id=2), got id=%d", top[0].ID)
	}
	if top[1].ID != 3 {
		t.Errorf("second should be C (id=3), got id=%d", top[1].ID)
	}
}

func TestSelectTopAll(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "A", RelevanceScore: 0.1},
		{ID: 2, Title: "B", RelevanceScore: 0.9},
	}
	scored := []llm.ScoredArticle{
		{ID: 1, Title: "A", Score: 0.1},
		{ID: 2, Title: "B", Score: 0.9},
	}

	engine := &Engine{}
	// maxN 大于文章数
	top := engine.selectTop(articles, scored, 10)

	if len(top) != 2 {
		t.Errorf("should cap at article count, got %d", len(top))
	}
}

func TestBuildRankInput(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "Test Title", Description: "Test Description", URL: "https://example.com"},
	}

	input := buildRankInput(articles)

	if len(input) != 1 {
		t.Errorf("should have 1 item, got %d", len(input))
	}
	if input[0].ID != 1 {
		t.Errorf("id mismatch: got %d, want 1", input[0].ID)
	}
	if input[0].Title != "Test Title" {
		t.Errorf("title mismatch: got '%s', want 'Test Title'", input[0].Title)
	}
}

func TestTemplateBriefing(t *testing.T) {
	now := time.Now()
	articles := []models.Article{
		{
			ID: 1, Title: "Article 1", URL: "https://example.com/1",
			Summary: "Summary 1", RelevanceScore: 0.9,
			PublishedAt: &now,
		},
		{
			ID: 2, Title: "Article 2", URL: "https://example.com/2",
			Summary: "Summary 2", RelevanceScore: 0.5,
		},
	}
	keywords := []string{"AI", "Go"}

	markdown := templateBriefing(articles, keywords)

	if !contains(markdown, "Article 1") {
		t.Error("template should include article title")
	}
	if !contains(markdown, "AI") {
		t.Error("template should include keywords")
	}
	if !contains(markdown, "2") || !contains(markdown, "篇文章") {
		t.Error("template should include article count")
	}
	if !contains(markdown, "每日简报") {
		t.Error("template should have daily briefing title")
	}
}

func TestTemplateBriefingNoKeywords(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "Test", URL: "https://example.com", Summary: "Test summary", RelevanceScore: 0.5},
	}

	markdown := templateBriefing(articles, nil)

	if !contains(markdown, "今日文章") {
		t.Error("template should include articles section")
	}
}

func TestAssemblyInput(t *testing.T) {
	articles := []models.Article{
		{ID: 1, Title: "Test", URL: "https://example.com", Summary: "Summary", RelevanceScore: 0.8},
	}
	summaries := []string{"Generated summary"}

	input := buildAssemblyInput(articles, summaries)

	if len(input) != 1 {
		t.Errorf("should have 1 item, got %d", len(input))
	}
	if input[0].ID != 1 {
		t.Errorf("id mismatch")
	}
	if input[0].Summary != "Generated summary" {
		t.Errorf("summary should come from summaries param, got '%s'", input[0].Summary)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
