package pipeline

import (
	"loledgeagent/internal/models"
)

// Deduplicate 去重：按 dedup_hash 过滤重复文章
// 返回去重后的文章列表
func Deduplicate(articles []models.Article, existingHashes map[string]bool) []models.Article {
	if len(articles) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(articles))
	result := make([]models.Article, 0, len(articles))

	for _, a := range articles {
		if seen[a.DedupHash] || existingHashes[a.DedupHash] {
			continue
		}
		seen[a.DedupHash] = true
		result = append(result, a)
	}
	return result
}
