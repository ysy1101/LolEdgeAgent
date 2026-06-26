package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/sources"
)

type FetchService struct {
	sourceRepo  *repository.SourceRepo
	articleRepo *repository.ArticleRepo
	logRepo     *repository.FetchLogRepo
	ragSvc      *RAGService
	logger      *slog.Logger
}

func NewFetchService(
	sourceRepo *repository.SourceRepo,
	articleRepo *repository.ArticleRepo,
	logRepo *repository.FetchLogRepo,
	ragSvc *RAGService,
	logger *slog.Logger,
) *FetchService {
	return &FetchService{
		sourceRepo:  sourceRepo,
		articleRepo: articleRepo,
		logRepo:     logRepo,
		ragSvc:      ragSvc,
		logger:      logger,
	}
}

// FetchAll 拉取所有启用的源，返回新文章总数。
func (s *FetchService) FetchAll(ctx context.Context) ([]models.Article, error) {
	list, err := s.sourceRepo.List(true)
	if err != nil {
		return nil, fmt.Errorf("获取源列表失败: %w", err)
	}

	var (
		allArticles []models.Article
		mu          sync.Mutex
		wg          sync.WaitGroup
		sem         = make(chan struct{}, 5)
	)

	for i := range list {
		wg.Add(1)
		sem <- struct{}{}
		go func(src models.Source) {
			defer wg.Done()
			defer func() { <-sem }()

			articles, err := s.fetchOne(ctx, src)
			if err != nil {
				s.logger.Warn("source fetch failed", "name", src.Name, "error", err)
				return
			}

			mu.Lock()
			allArticles = append(allArticles, articles...)
			mu.Unlock()
		}(list[i])
	}
	wg.Wait()

	return allArticles, nil
}

// FetchSource 拉取单个源。
func (s *FetchService) FetchSource(ctx context.Context, sourceID uint) (int, error) {
	src, err := s.sourceRepo.Get(sourceID)
	if err != nil {
		return 0, err
	}

	articles, err := s.fetchOne(ctx, *src)
	return len(articles), err
}

func (s *FetchService) fetchOne(ctx context.Context, src models.Source) ([]models.Article, error) {
	started := time.Now()
	log := &models.FetchLog{
		SourceID:  src.ID,
		Status:    "error",
		StartedAt: started,
	}
	defer func() {
		log.CompletedAt = timePtr(time.Now())
		_ = s.logRepo.Create(log)
	}()

	plugin, err := sources.Get(src.SourceType)
	if err != nil {
		log.ErrorMessage = err.Error()
		return nil, err
	}

	articles, err := plugin.Fetch(ctx, src)
	if err != nil {
		log.ErrorMessage = err.Error()
		return nil, err
	}

	// 去重后入库
	deduped := s.deduplicate(articles)
	if len(deduped) > 0 {
		if err := s.articleRepo.UpsertBatch(deduped); err != nil {
			log.ErrorMessage = err.Error()
			return nil, err
		}
		// 异步建向量索引（不阻塞抓取，失败不影响入库）
		for _, a := range deduped {
			go func(article models.Article) {
				if err := s.ragSvc.IndexArticle(context.Background(), &article); err != nil {
					s.logger.Warn("article index failed", "article_id", article.ID, "error", err)
				}
			}(a)
		}
	}

	log.Status = "success"
	log.ArticlesFetched = len(deduped)
	return deduped, nil
}

// deduplicate 过滤掉数据库中已存在的文章。
func (s *FetchService) deduplicate(articles []models.Article) []models.Article {
	if len(articles) == 0 {
		return nil
	}

	hashes := make([]string, len(articles))
	for i, a := range articles {
		hashes[i] = a.DedupHash
	}

	existing, err := s.articleRepo.FindExistingHashes(hashes)
	if err != nil {
		s.logger.Warn("dedup hash lookup failed", "error", err)
		return articles
	}

	result := make([]models.Article, 0, len(articles))
	for _, a := range articles {
		if !existing[a.DedupHash] {
			result = append(result, a)
		}
	}
	return result
}

func timePtr(t time.Time) *time.Time { return &t }
