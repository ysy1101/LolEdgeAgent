package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"

	"loledgeagent/internal/llm"
	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
)

type RAGService struct {
	embRepo     *repository.EmbeddingRepo
	articleRepo *repository.ArticleRepo
	llmClient   *llm.Client
	logger      *slog.Logger
}

func NewRAGService(
	embRepo *repository.EmbeddingRepo,
	articleRepo *repository.ArticleRepo,
	llmClient *llm.Client,
	logger *slog.Logger,
) *RAGService {
	return &RAGService{embRepo: embRepo, articleRepo: articleRepo, llmClient: llmClient, logger: logger}
}

// IndexArticle 为一篇文章生成并存储向量
func (s *RAGService) IndexArticle(ctx context.Context, article *models.Article) error {
	if s.llmClient == nil {
		return fmt.Errorf("AI 功能未配置，请先在偏好设置中设置 API Key")
	}
	exists, _ := s.embRepo.Exists(article.ID)
	if exists {
		return nil
	}

	text := article.Title + " " + article.Description
	if text == " " {
		text = article.Title
	}

	vecs, err := s.llmClient.Embeddings(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("embed article %d: %w", article.ID, err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("empty embedding for article %d", article.ID)
	}

	vecJSON, _ := json.Marshal(vecs[0])
	return s.embRepo.Upsert(&models.ArticleEmbedding{
		ArticleID: article.ID,
		Embedding: string(vecJSON),
	})
}

// Search 语义搜索文章
func (s *RAGService) Search(ctx context.Context, query string, topK int) ([]models.Article, error) {
	if s.llmClient == nil {
		return nil, fmt.Errorf("AI 功能未配置，请先在偏好设置中设置 API Key")
	}

	// ① 查询向量化
	vecs, err := s.llmClient.Embeddings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vecs[0]

	// ② 获取所有已索引文章的向量
	all, err := s.embRepo.GetAll()
	if err != nil {
		return nil, err
	}

	// ③ 余弦相似度排序
	type scored struct {
		articleID uint
		score     float64
	}
	var scores []scored
	for _, ae := range all {
		var dbVec []float64
		if err := json.Unmarshal([]byte(ae.Embedding), &dbVec); err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, dbVec)
		scores = append(scores, scored{articleID: ae.ArticleID, score: sim})
	}

	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if topK > len(scores) {
		topK = len(scores)
	}

	// ④ 获取文章详情
	result := make([]models.Article, 0, topK)
	for i := 0; i < topK; i++ {
		a, err := s.articleRepo.Get( // need a Get method
			scores[i].articleID,
		)
		if err == nil {
			result = append(result, *a)
		}
	}
	return result, nil
}

// Ask 知识问答：检索 + LLM 回答
func (s *RAGService) Ask(ctx context.Context, question string, topK int) (string, []models.Article, error) {
	articles, err := s.Search(ctx, question, topK)
	if err != nil {
		return "", nil, err
	}

	// 构建上下文
	var sb strings.Builder
	for i, a := range articles {
		sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, a.Title, a.Description))
	}

	sys := `根据以下文章内容回答用户问题。如果文章中找不到答案，诚实说明。引用时注明文章编号。用中文回答。`
	user := fmt.Sprintf("文章：\n%s\n\n问题：%s", sb.String(), question)

	answer, err := s.llmClient.Chat(ctx, sys, user)
	return answer, articles, err
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
