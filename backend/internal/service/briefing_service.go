package service

import (
	"context"
	"log/slog"

	"loledgeagent/internal/models"
	"loledgeagent/internal/pipeline"
	"loledgeagent/internal/repository"

	"gorm.io/gorm"
)

type BriefingService struct {
	repo     *repository.BriefingRepo
	fetchSvc *FetchService
	engine   *pipeline.Engine
	logger   *slog.Logger
}

func NewBriefingService(
	db *gorm.DB,
	fetchSvc *FetchService,
	engine *pipeline.Engine,
	logger *slog.Logger,
) *BriefingService {
	return &BriefingService{
		repo:     repository.NewBriefingRepo(db),
		fetchSvc: fetchSvc,
		engine:   engine,
		logger:   logger,
	}
}

// GenerateAsync 异步生成简报，立即返回 briefing_id
func (s *BriefingService) GenerateAsync(ctx context.Context, userID uint) (uint, error) {
	// 先创建占位记录
	placeholder := &models.Briefing{
		UserID:  userID,
		Title:   "生成中...",
		Status:  "generating",
	}
	if err := s.repo.Create(placeholder); err != nil {
		return 0, err
	}

	go func() {
		bg := context.Background()

		articles, err := s.fetchSvc.FetchAll(bg)
		if err != nil {
			_ = s.repo.UpdateStatus(placeholder.ID, "failed", err.Error())
			s.logger.Error("fetch failed", "error", err)
			return
		}

		result, err := s.engine.Run(bg, articles, userID)
		if err != nil {
			_ = s.repo.UpdateStatus(placeholder.ID, "failed", err.Error())
			s.logger.Error("pipeline failed", "error", err)
			return
		}

		// 删除占位记录，改用管线生成的正式记录
		_ = s.repo.Delete(placeholder.ID)
		s.logger.Info("briefing generated", "id", result.ID, "articles", result.ArticleCount)
	}()

	return placeholder.ID, nil
}

func (s *BriefingService) Get(id uint) (*models.Briefing, error) { return s.repo.GetByID(id) }

func (s *BriefingService) List(userID uint, page, limit int) ([]models.Briefing, int64, error) {
	return s.repo.List(userID, page, limit)
}

func (s *BriefingService) Delete(id uint) error { return s.repo.Delete(id) }
