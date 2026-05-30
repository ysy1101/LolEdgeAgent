package main

import (
	"log/slog"
	"os"

	v1 "loledgeagent/api/v1"
	"loledgeagent/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	_ = cfg // TODO: use when wiring DB, LLM

	r := gin.New()
	r.Use(gin.Recovery())

	v1.RegisterRoutes(r)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("server starting", "port", cfg.Server.Port)

	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
