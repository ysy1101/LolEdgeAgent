package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool LLM 可调用的工具
type Tool struct {
	Name        string
	Description string         // 给 LLM 看的功能描述
	Parameters  map[string]any // JSON Schema 参数定义
	Execute     func(ctx context.Context, args map[string]any) (string, error)
}

var registry = map[string]*Tool{}

func Register(t *Tool) {
	registry[t.Name] = t
}

func Get(name string) (*Tool, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return t, nil
}

func AllTools() []*Tool {
	var list []*Tool
	for _, t := range registry {
		list = append(list, t)
	}
	return list
}

// ToolDef 用于序列化给 LLM 的工具定义
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

func ToolDefinitions() []ToolDef {
	var defs []ToolDef
	for _, t := range registry {
		defs = append(defs, ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	return defs
}

func toolJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
