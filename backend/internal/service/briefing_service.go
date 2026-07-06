package service

import (
	"context"
	"fmt"
	"log/slog"

	"loledgeagent/internal/models"
	"loledgeagent/internal/pipeline"
	"loledgeagent/internal/repository"

	"gorm.io/gorm"
)

type BriefingService struct {
	repo        *repository.BriefingRepo
	fetchSvc    *FetchService
	articleRepo *repository.ArticleRepo
	engine      *pipeline.Engine
	logger      *slog.Logger
}

func NewBriefingService(
	db *gorm.DB,
	fetchSvc *FetchService,
	articleRepo *repository.ArticleRepo,
	engine *pipeline.Engine,
	logger *slog.Logger,
) *BriefingService {
	return &BriefingService{
		repo:        repository.NewBriefingRepo(db),
		fetchSvc:    fetchSvc,
		articleRepo: articleRepo,
		engine:      engine,
		logger:      logger,
	}
}

// GenerateAsync 异步生成简报，立即返回 briefing_id，过程中更新 progress
func (s *BriefingService) GenerateAsync(ctx context.Context, userID uint) (uint, error) {
	placeholder := &models.Briefing{
		UserID:   userID,
		Title:    "生成中...",
		Status:   "generating",
		Progress: "loading:fetching articles",
	}
	if err := s.repo.Create(placeholder); err != nil {
		return 0, err
	}

	go func() {
		bg := context.Background()

		articles, err := s.articleRepo.GetRecent(200)
		if err != nil || len(articles) == 0 {
			_ = s.repo.UpdateStatus(placeholder.ID, "failed", "no articles in database, fetch first")
			s.logger.Error("no articles", "error", err)
			return
		}
		_ = s.repo.UpdateProgress(placeholder.ID, "generating", fmt.Sprintf("ranking %d articles", len(articles)))

		result, err := s.engine.RunInto(bg, articles, userID, placeholder.ID)
		if err != nil {
			_ = s.repo.UpdateStatus(placeholder.ID, "failed", err.Error())
			s.logger.Error("pipeline failed", "error", err)
			return
		}

		s.logger.Info("briefing generated", "id", result.ID, "articles", result.ArticleCount)
	}()

	return placeholder.ID, nil
}

func (s *BriefingService) Get(id uint, userID uint) (*models.Briefing, error) {
	return s.repo.GetByID(id, userID)
}

func (s *BriefingService) List(userID uint, page, limit int) ([]models.Briefing, int64, error) {
	return s.repo.List(userID, page, limit)
}

func (s *BriefingService) Delete(id uint) error { return s.repo.Delete(id) }
