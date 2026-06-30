package repository

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"loledgeagent/internal/models"

	"gorm.io/gorm"
)

type MemoryRepo struct{ db *gorm.DB }

func NewMemoryRepo(db *gorm.DB) *MemoryRepo { return &MemoryRepo{db: db} }

// Save 保存记忆 + 向量
func (r *MemoryRepo) Save(memory *models.Memory, embedding []float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(memory).Error; err != nil {
			return err
		}
		vecJSON, _ := json.Marshal(embedding)
		return tx.Create(&models.MemoryEmbedding{
			MemoryID:  memory.ID,
			Embedding: string(vecJSON),
		}).Error
	})
}

// Recent 最近 N 条记忆
func (r *MemoryRepo) Recent(userID uint, limit int) ([]models.Memory, error) {
	var list []models.Memory
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// KeywordSearch 关键词粗筛（JSON keywords 字段 LIKE 匹配）
func (r *MemoryRepo) KeywordSearch(userID uint, queryKeywords string) ([]models.Memory, error) {
	keywords := strings.Fields(queryKeywords)
	var list []models.Memory
	q := r.db.Where("user_id = ?", userID)
	likeClause := make([]string, 0, len(keywords))
	args := make([]any, 0, len(keywords))
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if len(kw) > 1 {
			likeClause = append(likeClause, "keywords LIKE ?")
			args = append(args, "%"+kw+"%")
		}
	}
	if len(likeClause) == 0 {
		// 没有有效关键词，返回最近记忆
		return r.Recent(userID, 10)
	}
	err := q.Where(strings.Join(likeClause, " OR "), args...).
		Order("importance DESC, access_count DESC").
		Limit(10).Find(&list).Error
	return list, err
}

// SemanticRank 语义精排：对候选记忆按余弦相似度排序
func (r *MemoryRepo) SemanticRank(candidates []models.Memory, queryVec []float64, topK int) []models.Memory {
	if len(queryVec) == 0 || len(candidates) == 0 {
		return candidates
	}

	type item struct {
		memory models.Memory
		score  float64
	}
	var ranked []item
	for _, m := range candidates {
		var emb models.MemoryEmbedding
		if err := r.db.Where("memory_id = ?", m.ID).First(&emb).Error; err != nil {
			continue
		}
		var vec []float64
		if err := json.Unmarshal([]byte(emb.Embedding), &vec); err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, vec)
		ranked = append(ranked, item{memory: m, score: sim})
	}

	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if topK > len(ranked) {
		topK = len(ranked)
	}
	result := make([]models.Memory, topK)
	for i := 0; i < topK; i++ {
		result[i] = ranked[i].memory
	}
	return result
}

// Touch 增加访问计数
func (r *MemoryRepo) Touch(id uint) {
	r.db.Model(&models.Memory{}).Where("id = ?", id).
		UpdateColumn("access_count", gorm.Expr("access_count + 1"))
}

// DeleteOld 删除过旧低重要性的记忆（保留最近 N 条）
func (r *MemoryRepo) DeleteOld(userID uint, keep int) error {
	if keep <= 0 {
		keep = 50
	}
	// 删除 keep 名次之外的低重要性记忆
	return r.db.Exec(`
		DELETE FROM memories WHERE user_id = ? AND id NOT IN (
			SELECT id FROM memories WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
		) AND importance < 0.3
	`, userID, userID, keep).Error
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
