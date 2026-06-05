package main

import (
	"log/slog"
	"os"

	v1 "loledgeagent/api/v1"
	"loledgeagent/internal/config"
	"loledgeagent/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// 初始化数据库
	db, err := gorm.Open(sqlite.Open(cfg.Database.Path), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	db.AutoMigrate(
		&models.User{},
		&models.Source{},
		&models.Article{},
		&models.Briefing{},
		&models.BriefingArticle{},
		&models.Bookmark{},
		&models.Preference{},
		&models.FetchLog{},
		&models.ArticleEmbedding{},
	)

	// 默认用户（MVP 阶段硬编码）
	defaultUser := &models.User{Username: "admin", Email: "admin@loledgeagent.local"}
	db.FirstOrCreate(defaultUser, models.User{Username: "admin"})
	db.FirstOrCreate(&models.Preference{}, models.Preference{UserID: defaultUser.ID})

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("database initialized", "path", cfg.Database.Path)

	r := gin.New()
	r.Use(gin.Recovery())

	v1.RegisterRoutes(r, db, logger)

	logger.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
