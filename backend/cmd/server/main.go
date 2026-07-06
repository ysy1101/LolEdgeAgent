package main

import (
	"log/slog"
	"os"
	"time"

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
		&models.Conversation{},
		&models.ChatMessage{},
		&models.Memory{},
		&models.MemoryEmbedding{},
	)

	// （注册时：存在但无密码 → 设置密码；不存在 → 新建）

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}))// 记录日志到 stdout，格式为 JSON，时间格式为 "2006-01-02 15:04:05"
	logger.Info("database initialized", "path", cfg.Database.Path)

	r := gin.New()
	r.Use(gin.Recovery())

	v1.RegisterRoutes(r, db, logger, cfg)

	logger.Info("server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
