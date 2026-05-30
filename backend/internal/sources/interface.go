package sources

import (
	"context"
	"fmt"
	"sync"

	"loledgeagent/internal/models"
)

// Plugin 是所有内容源必须实现的接口。
type Plugin interface {
	// Name 返回源类型标识，如 "rss"、"hackernews"、"github"。
	Name() string

	// Fetch 从源抓取文章。
	Fetch(ctx context.Context, source models.Source) ([]models.Article, error)

	// Validate 校验源配置是否有效。
	Validate(source models.Source) error
}

var (
	mu      sync.RWMutex
	plugins = make(map[string]Plugin)
)

// Register 注册一个源插件。
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	plugins[p.Name()] = p
}

// Get 根据名称获取插件。
func Get(name string) (Plugin, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := plugins[name]
	if !ok {
		return nil, fmt.Errorf("unknown source type: %s", name)
	}
	return p, nil
}

// Names 返回所有已注册的插件名称。
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(plugins))
	for n := range plugins {
		names = append(names, n)
	}
	return names
}
