package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"loledgeagent/internal/llm"
)

const (
	maxRounds       = 8
	maxContextChars = 12000 // 上下文字符数上限，超过则截断
)

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

	// 历史消息（超过阈值截断开头，保留最近对话）
	var userContent string
	for _, m := range history {
		userContent += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}
	if len(userContent) > maxContextChars {
		userContent = "…(更早的对话已省略)…\n" + userContent[len(userContent)-maxContextChars:]
	}
	if userContent != "" {
		system += "\n\n## 对话历史\n" + userContent
	}

	// Agent 循环（LLM 自主决定是否用工具）
	messages := []llm.ChatMessage{{Role: "system", Content: system}, {Role: "user", Content: userMsg}}

	for round := 0; round < maxRounds; round++ {
		a.logger.Info("agent round", "round", round+1)

		// 上下文保护：超过阈值截断中间消息
		messages = a.trimContext(messages)

		// 调 LLM（多轮消息）
		raw, err := a.llmClient.ChatMessages(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}

		resp, err := a.parseResponse(raw)
		if err != nil || resp.Type == "" {
			return &Reply{Content: cleanReply(raw)}, nil
		}

		if resp.Type == "final" || resp.Type == "answer" {
			return &Reply{Content: resp.Content}, nil
		}

		// 如果是工具调用但解析失败，不当成最终回答
		if resp.Type == "tool" {
			a.logger.Info("agent tool call (unexpected final)", "tool", resp.ToolName)
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


// trimContext 截断中间消息，保留 system prompt 和最近轮次
func (a *Agent) trimContext(msgs []llm.ChatMessage) []llm.ChatMessage {
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}
	if total <= maxContextChars || len(msgs) <= 5 {
		return msgs
	}
	keep := 4
	if len(msgs)-keep < 1 {
		keep = len(msgs) - 1
	}
	result := make([]llm.ChatMessage, 0, keep+2)
	result = append(result, msgs[0])
	result = append(result, llm.ChatMessage{Role: "system", Content: "…(中间消息已截断)…"})
	result = append(result, msgs[len(msgs)-keep:]...)
	return result
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

// cleanReply 去除回复中的 JSON 工具调用残留
func cleanReply(raw string) string {
	// 先尝试当 JSON 解，提取 content
	var resp LLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err == nil {
		if resp.Content != "" {
			return resp.Content
		}
	}
	// 去掉内嵌的 JSON tool call
	for {
		start := strings.Index(raw, `{"type":"tool"`)
		if start < 0 {
			break
		}
		end := strings.Index(raw[start:], "}")
		if end < 0 {
			break
		}
		raw = raw[:start] + raw[start+end+1:]
	}
	return strings.TrimSpace(raw)
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
