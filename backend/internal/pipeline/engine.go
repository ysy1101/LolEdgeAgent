package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"loledgeagent/internal/llm"
	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
)

// Engine 管线编排器：将已采集的文章加工成简报
type Engine struct {
	articleRepo  *repository.ArticleRepo
	briefingRepo *repository.BriefingRepo
	prefRepo     *repository.PreferenceRepo
	llmClient    *llm.Client
	logger       *slog.Logger
}

func NewEngine(
	articleRepo *repository.ArticleRepo,
	briefingRepo *repository.BriefingRepo,
	prefRepo *repository.PreferenceRepo,
	llmClient *llm.Client,
	logger *slog.Logger,
) *Engine {
	return &Engine{
		articleRepo:  articleRepo,
		briefingRepo: briefingRepo,
		prefRepo:     prefRepo,
		llmClient:    llmClient,
		logger:       logger,
	}
}

// RunWithID 核心管线，更新已有的 placeholder briefing
func (e *Engine) RunWithID(ctx context.Context, briefingID uint, rawArticles []models.Article, userID uint) (*models.Briefing, error) {
	result, err := e.run(ctx, rawArticles, userID)
	if err != nil {
		return nil, err
	}
	// 用管线结果更新 placeholder
	if err := e.briefingRepo.UpdateContent(briefingID, result.Title, result.ContentMarkdown, result.ArticleCount); err != nil {
		return nil, err
	}
	// 删除之前自动创建的 briefing 记录
	_ = e.briefingRepo.Delete(result.ID)

	result.ID = briefingID
	return result, nil
}

// Run 核心管线：去重→排序→摘要→组装→保存（新建 briefing）
func (e *Engine) Run(ctx context.Context, rawArticles []models.Article, userID uint) (*models.Briefing, error) {
	return e.run(ctx, rawArticles, userID)
}

func (e *Engine) run(ctx context.Context, rawArticles []models.Article, userID uint) (*models.Briefing, error) {
	if len(rawArticles) == 0 {
		return nil, fmt.Errorf("no articles")
	}

	// ① 去重
	hashes := make([]string, len(rawArticles))
	for i, a := range rawArticles {
		hashes[i] = a.DedupHash
	}
	existing, _ := e.articleRepo.FindExistingHashes(hashes)
	unique := Deduplicate(rawArticles, existing)
	e.logger.Info("pipeline: dedup", "raw", len(rawArticles), "unique", len(unique))

	if len(unique) == 0 {
		return nil, fmt.Errorf("no new articles after dedup")
	}

	// ② 入库
	if err := e.articleRepo.UpsertBatch(unique); err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	// ③ 加载偏好
	pref, err := e.prefRepo.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("preferences: %w", err)
	}
	var keywords []string
	json.Unmarshal([]byte(pref.Keywords), &keywords)

	// ④ LLM 排名
	rankInput := buildRankInput(unique)
	articlesJSON, _ := json.Marshal(rankInput)
	interestsJSON, _ := json.Marshal(keywords)

	scored, err := llm.RankArticles(ctx, e.llmClient, string(articlesJSON), string(interestsJSON))
	if err != nil {
		e.logger.Warn("pipeline: ranking failed", "error", err)
		scored = defaultRank(unique)
	}

	// 取 top N
	maxN := pref.MaxBriefingArticles
	if maxN <= 0 {
		maxN = 10
	}
	topArticles := e.selectTop(unique, scored, maxN)

	// ⑤ LLM 摘要
	e.logger.Info("pipeline: summarizing", "count", len(topArticles))
	summaryInputs := make([]llm.SummaryInput, len(topArticles))
	for i := range topArticles {
		summaryInputs[i] = llm.SummaryInput{
			ID: topArticles[i].ID, Title: topArticles[i].Title,
			Content: topArticles[i].Content, Description: topArticles[i].Description,
			URL: topArticles[i].URL,
		}
	}
	summaries := llm.SummarizeBatch(ctx, e.llmClient, summaryInputs, e.logger)
	for i, s := range summaries {
		if s != "" {
			_ = e.articleRepo.UpdateSummary(topArticles[i].ID, s, topArticles[i].RelevanceScore)
			topArticles[i].Summary = s
		}
	}

	// ⑥ LLM 组装
	assemblyInput := buildAssemblyInput(topArticles, summaries)
	assemblyJSON, _ := json.Marshal(assemblyInput)

	markdown, err := llm.AssembleBriefing(ctx, e.llmClient, string(assemblyJSON), string(interestsJSON))
	if err != nil {
		e.logger.Warn("pipeline: assembly failed", "error", err)
		markdown = templateBriefing(topArticles, keywords)
	}

	// ⑦ 保存简报
	briefing := &models.Briefing{
		UserID:          userID,
		Title:           fmt.Sprintf("每日简报 - %s", time.Now().Format("2006-01-02")),
		ContentMarkdown: markdown,
		ArticleCount:    len(topArticles),
		GeneratedAt:     time.Now(),
		Status:          "completed",
	}
	ba := make([]models.BriefingArticle, len(topArticles))
	for i := range topArticles {
		ba[i] = models.BriefingArticle{
			ArticleID:    topArticles[i].ID,
			RankPosition: i + 1,
		}
	}
	if err := e.briefingRepo.CreateWithArticles(briefing, ba); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	e.logger.Info("pipeline: completed", "briefing_id", briefing.ID, "articles", len(topArticles))
	return briefing, nil
}

type idxScore struct {
	idx   int
	score float64
}

func (e *Engine) selectTop(articles []models.Article, scored []llm.ScoredArticle, maxN int) []models.Article {
	scoreMap := make(map[uint]float64, len(scored))
	for _, s := range scored {
		scoreMap[s.ID] = s.Score
	}

	var ws []idxScore
	for i, a := range articles {
		s, ok := scoreMap[a.ID]
		if !ok {
			s = 0.5
		}
		articles[i].RelevanceScore = s
		ws = append(ws, idxScore{idx: i, score: s})
	}

	sortIdxScore(ws)

	if maxN > len(articles) {
		maxN = len(articles)
	}
	result := make([]models.Article, maxN)
	for i := 0; i < maxN; i++ {
		result[i] = articles[ws[i].idx]
	}
	return result
}

func sortIdxScore(ws []idxScore) {
	for i := 0; i < len(ws); i++ {
		for j := i + 1; j < len(ws); j++ {
			if ws[j].score > ws[i].score {
				ws[i], ws[j] = ws[j], ws[i]
			}
		}
	}
}

// === helpers ===

type rankInput struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

func buildRankInput(articles []models.Article) []rankInput {
	r := make([]rankInput, len(articles))
	for i, a := range articles {
		r[i] = rankInput{ID: a.ID, Title: a.Title, Desc: a.Description}
	}
	return r
}

type assemblyInput struct {
	ID      uint    `json:"id"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Summary string  `json:"summary"`
	Score   float64 `json:"score"`
}

func buildAssemblyInput(articles []models.Article, summaries []string) []assemblyInput {
	r := make([]assemblyInput, len(articles))
	for i := range articles {
		r[i] = assemblyInput{
			ID:      articles[i].ID,
			Title:   articles[i].Title,
			URL:     articles[i].URL,
			Summary: summaries[i],
			Score:   articles[i].RelevanceScore,
		}
	}
	return r
}

func defaultRank(articles []models.Article) []llm.ScoredArticle {
	r := make([]llm.ScoredArticle, len(articles))
	for i, a := range articles {
		r[i] = llm.ScoredArticle{ID: a.ID, Title: a.Title, Score: 0.5, Rationale: "默认评分"}
	}
	return r
}

func templateBriefing(articles []models.Article, keywords []string) string {
	var sb strings.Builder
	sb.WriteString("# 每日简报\n\n")
	if len(keywords) > 0 {
		fmt.Fprintf(&sb, "> 关注领域：%s\n\n", strings.Join(keywords, "、"))
	}
	sb.WriteString("## 今日文章\n\n")
	for i, a := range articles {
		fmt.Fprintf(&sb, "%d. **[%s](%s)** - %s - 评分: %.2f\n\n",
			i+1, a.Title, a.URL, a.Summary, a.RelevanceScore)
	}
	fmt.Fprintf(&sb, "---\n今日共收录 **%d** 篇文章\n", len(articles))
	return sb.String()
}
