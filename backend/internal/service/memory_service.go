package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"loledgeagent/internal/llm"
	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
)

type MemoryService struct {
	memoryRepo *repository.MemoryRepo
	msgRepo    *repository.MessageRepo
	llmClient  *llm.Client
	logger     *slog.Logger
}

func NewMemoryService(
	memoryRepo *repository.MemoryRepo,
	msgRepo *repository.MessageRepo,
	llmClient *llm.Client,
	logger *slog.Logger,
) *MemoryService {
	return &MemoryService{
		memoryRepo: memoryRepo,
		msgRepo:    msgRepo,
		llmClient:  llmClient,
		logger:     logger,
	}
}

// Compress 将对话历史压缩为记忆摘要，存入 memories + embedding
func (s *MemoryService) Compress(ctx context.Context, userID uint, conversationID uint) error {
	if s.llmClient == nil {
		return fmt.Errorf("LLM 未配置，无法压缩记忆")
	}

	msgs, err := s.msgRepo.ListByConversation(conversationID)
	if err != nil || len(msgs) == 0 {
		return err
	}

	// 取最近消息（不超过 20 条）
	if len(msgs) > 20 {
		msgs = msgs[len(msgs)-20:]
	}

	// 构建对话记录的文本
	var sb strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}

	compressPrompt := `根据以下对话记录，生成一段简洁的摘要，包含：
1. 用户的核心意图是什么
2. 用户是否表露了新的偏好或兴趣
3. 用户对哪些话题/文章表现出特别关注

只返回摘要，不超过 100 字。`

	summary, err := s.llmClient.Chat(ctx, compressPrompt, sb.String())
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return nil
	}

	// 提取关键词
	keywords := extractKeywords(summary)

	memory := &models.Memory{
		UserID:     userID,
		Content:    summary,
		Keywords:   keywords,
		Importance: 0.5,
	}

	// 生成向量
	vecs, err := s.llmClient.Embeddings(ctx, []string{summary})
	if err != nil {
		s.logger.Warn("memory embedding failed, saving without vector", "error", err)
		return s.memoryRepo.Save(memory, nil)
	}

	return s.memoryRepo.Save(memory, vecs[0])
}

// Recall 检索相关记忆，返回格式化文本（用于拼入 System Prompt）
func (s *MemoryService) Recall(ctx context.Context, userID uint, query string) string {
	// 1. 关键词粗筛
	candidates, err := s.memoryRepo.KeywordSearch(userID, query)
	if err != nil || len(candidates) == 0 {
		return ""
	}

	// 2. 语义精排（如果有 LLM 且向量可用）
	var ranked []models.Memory
	if s.llmClient != nil && len(candidates) > 5 {
		vecs, err := s.llmClient.Embeddings(ctx, []string{query})
		if err == nil && len(vecs) > 0 {
			ranked = s.memoryRepo.SemanticRank(candidates, vecs[0], 5)
		}
	}
	if len(ranked) == 0 {
		ranked = candidates
		if len(ranked) > 5 {
			ranked = ranked[:5]
		}
	}

	// 标记访问
	for _, m := range ranked {
		s.memoryRepo.Touch(m.ID)
	}

	// 格式化为文本
	var parts []string
	for _, m := range ranked {
		parts = append(parts, "- "+m.Content)
	}
	return strings.Join(parts, "\n")
}

// Cleanup 清理旧的低重要性记忆
func (s *MemoryService) Cleanup(userID uint) {
	if err := s.memoryRepo.DeleteOld(userID, 50); err != nil {
		s.logger.Warn("memory cleanup failed", "error", err)
	}
}

// extractKeywords 简单关键词提取
func extractKeywords(text string) string {
	// 简单实现：按常见分隔符分词，取长度 >= 2 的词
	var words []string
	for _, w := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '，' || r == '、' || r == '。' || r == '；' || r == '\n'
	}) {
		w = strings.TrimSpace(w)
		if len(w) >= 2 && !isStopWord(w) {
			words = append(words, w)
		}
	}
	if len(words) > 10 {
		words = words[:10]
	}
	return "[" + strings.Join(words, ",") + "]"
}

func isStopWord(w string) bool {
	stop := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "和": true,
		"不": true, "也": true, "都": true, "就": true, "要": true,
		"用户": true, "关注": true, "偏好": true, "兴趣": true, "话题": true,
	}
	return stop[w]
}
