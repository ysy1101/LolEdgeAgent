package scheduler

import (
	"context"
	"log/slog"

	"loledgeagent/internal/service"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron   *cron.Cron
	svc    *service.BriefingService
	logger *slog.Logger
}

func New(svc *service.BriefingService, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		cron:   cron.New(cron.WithSeconds()),
		svc:    svc,
		logger: logger,
	}
}

// Start 启动调度器，读取 schedule cron 表达式。
// schedule 为空或 "manual" 则不启动。
func (s *Scheduler) Start(schedule string) error {
	if schedule == "" || schedule == "manual" {
		s.logger.Info("scheduler: manual mode, no cron job started")
		return nil
	}

	// 停掉旧任务
	s.Stop()

	_, err := s.cron.AddFunc(schedule, func() {
		s.logger.Info("scheduler: triggering briefing generation")
		_, err := s.svc.GenerateAsync(context.Background(), 1) // userID=1 默认用户
		if err != nil {
			s.logger.Error("scheduler: generate failed", "error", err)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.logger.Info("scheduler: started", "cron", schedule)
	return nil
}

// Reload 根据新 schedule 重启调度器。
func (s *Scheduler) Reload(schedule string) error {
	return s.Start(schedule)
}

func (s *Scheduler) Stop() {
	crons := s.cron.Entries()
	if len(crons) > 0 {
		s.cron.Stop()
	}
}
