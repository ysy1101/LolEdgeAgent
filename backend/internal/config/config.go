package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds all configuration values.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	LLM      LLMConfig
	Fetch    FetchConfig
}

type ServerConfig struct {
	Port       string
	CorsOrigin string
}

type DatabaseConfig struct {
	Path string
}

type LLMConfig struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Timeout  int
}

type FetchConfig struct {
	Timeout        int
	MaxConcurrency int
}

// Load reads config from default path with environment overrides.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:       envOrDefault("SERVER_PORT", "8080"),
			CorsOrigin: envOrDefault("CORS_ORIGIN", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			Path: resolvePath(envOrDefault("DB_PATH", "./data/loledgeagent.db")),
		},
		LLM: LLMConfig{
			Provider: envOrDefault("LLM_PROVIDER", "deepseek"),
			Model:    envOrDefault("LLM_MODEL", "deepseek-chat"),
			APIKey:   os.Getenv("LLM_API_KEY"),
			BaseURL:  os.Getenv("LLM_BASE_URL"),
			Timeout:  60,
		},
		Fetch: FetchConfig{
			Timeout:        30,
			MaxConcurrency: 5,
		},
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return strings.TrimSpace(v)
	}
	return defaultVal
}

func resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, p)
}
