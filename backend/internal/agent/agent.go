package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"loledgeagent/internal/llm"
)

const maxRounds = 8

type Message struct {
	Role    string // system / user / assistant / tool
	Content string
}

type LLMResponse struct {
	Type     string         `json:"type"` // "tool" or "final"
	ToolName string         `json:"name,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
	Content  string         `json:"content,omitempty"`
}

type Reply struct {
	Content    string `json:"content"`
	ToolCalled string `json:"tool_called"`
}

type Agent struct {
	llmClient *llm.Client
	logger    *slog.Logger
}

func New(client *llm.Client, logger *slog.Logger) *Agent {
	return &Agent{llmClient: client, logger: logger}
}

// Run 执行 Agent 对话（发送多轮消息，支持工具调用）
func (a *Agent) Run(ctx context.Context, history []Message, userMsg string) (*Reply, error) {
	if a.llmClient == nil {
		return &Reply{Content: "LLM 未配置，请先在偏好设置中配置 API Key。"}, nil
	}

	// 系统提示
	system := a.systemPrompt()

	// 历史消息压缩
	var userContent string
	for _, m := range history {
		userContent += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}
	if userContent != "" {
		system += "\n\n## 对话历史\n" + userContent
	}

	// LLM 调用（聊天类问题直接回答）
	if !a.needsTools(userMsg) {
		raw, err := a.llmClient.Chat(ctx, system, userMsg)
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}
		return &Reply{Content: raw}, nil
	}

	// 需要工具 → 循环
	messages := []llm.ChatMessage{{Role: "system", Content: system}, {Role: "user", Content: userMsg}}

	for round := 0; round < maxRounds; round++ {
		a.logger.Info("agent round", "round", round+1)

		// 调 LLM（多轮消息）
		raw, err := a.llmClient.ChatMessages(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}

		resp, err := a.parseResponse(raw)
		if err != nil || resp.Type == "" {
			// LLM 没按 JSON 回复，直接作为最终回答
			return &Reply{Content: raw}, nil
		}

		if resp.Type == "final" || resp.Type == "answer" {
			return &Reply{Content: resp.Content}, nil
		}

		// 执行工具
		tool, err := Get(resp.ToolName)
		if err != nil {
			messages = append(messages,
				llm.ChatMessage{Role: "assistant", Content: raw},
				llm.ChatMessage{Role: "user", Content: fmt.Sprintf("工具 %s 不存在，请选择其他方式回答", resp.ToolName)},
			)
			continue
		}

		a.logger.Info("agent tool call", "tool", resp.ToolName)
		result, execErr := tool.Execute(ctx, resp.Args)
		if execErr != nil {
			result = fmt.Sprintf("错误: %s", execErr.Error())
		}

		messages = append(messages,
			llm.ChatMessage{Role: "assistant", Content: raw},
			llm.ChatMessage{Role: "user", Content: fmt.Sprintf("工具 %s 返回: %s", resp.ToolName, result)},
		)
	}

	return &Reply{Content: "抱歉，处理超时，请简化问题重试。"}, nil
}


// needsTools 简单判断是否需要调用工具
func (a *Agent) needsTools(msg string) bool {
	keywords := []string{"搜索", "文章", "简报", "偏好", "生成", "找", "帮我", "今天", "最近"}
	for _, k := range keywords {
		if contains(msg, k) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (a *Agent) parseResponse(raw string) (*LLMResponse, error) {
	s := strings.TrimSpace(raw)

	// 去掉 ```json``` 包裹
	if strings.HasPrefix(s, "```json") || strings.HasPrefix(s, "```JSON") {
		s = s[7:]
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	// 找第一个 { 到最后一个 }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}

	var resp LLMResponse
	if err := json.Unmarshal([]byte(s), &resp); err != nil {
		a.logger.Warn("parse response failed", "raw", raw[:min(100, len(raw))], "error", err)
		return nil, err
	}
	return &resp, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


func (a *Agent) systemPrompt() string {
	toolsJSON, _ := json.Marshal(ToolDefinitions())
	prompt := `你是 LolEdgeAgent，一个内容聚合和知识助手。

## 可用工具
` + string(toolsJSON) + `

## 回复格式（严格 JSON）
需要调用工具时：
{"type":"tool","name":"<工具名>","args":{<参数>}}

给出最终回答时：
{"type":"final","content":"<你的回答>"}

## 规则
1. 如果用户问题不需要工具（闲聊、问候），直接用 final 回答
2. 用中文回复
3. 引用搜索到的文章时带上标题和链接
4. 如果工具执行失败，告知用户并给出替代建议`
	return prompt
}
