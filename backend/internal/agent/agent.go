package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
	Content    string
	ToolCalled string
}

type Agent struct {
	llmClient *llm.Client
	logger    *slog.Logger
}

func New(client *llm.Client, logger *slog.Logger) *Agent {
	return &Agent{llmClient: client, logger: logger}
}

func (a *Agent) Run(ctx context.Context, history []Message, userMsg string) (*Reply, error) {
	if a.llmClient == nil {
		return &Reply{Content: "LLM 未配置，请先在偏好设置中配置 API Key。"}, nil
	}

	// ① 构建初始 messages
	messages := []Message{
		{Role: "system", Content: a.systemPrompt()},
	}
	messages = append(messages, history...)
	messages = append(messages, Message{Role: "user", Content: userMsg})

	// ② 循环
	for round := 0; round < maxRounds; round++ {
		a.logger.Info("agent round", "round", round+1)

		// 把 messages 转成 LLM 格式
		var llmMsgs []string
		for _, m := range messages {
			llmMsgs = append(llmMsgs, fmt.Sprintf("[%s] %s", m.Role, m.Content))
		}

		// 调 LLM
		raw, err := a.llmClient.Chat(ctx, a.chatSystem(), a.chatUser(llmMsgs))
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}

		// 解析 LLM 响应
		resp, err := a.parseResponse(raw)
		if err != nil {
			// 解析失败，当成最终回答
			return &Reply{Content: raw}, nil
		}

		if resp.Type == "final" {
			return &Reply{Content: resp.Content}, nil
		}

		// 执行工具
		tool, err := Get(resp.ToolName)
		if err != nil {
			messages = append(messages,
				Message{Role: "assistant", Content: raw},
				Message{Role: "tool", Content: fmt.Sprintf("工具不存在: %s", resp.ToolName)},
			)
			continue
		}

		a.logger.Info("agent tool call", "tool", resp.ToolName)
		result, execErr := tool.Execute(ctx, resp.Args)
		if execErr != nil {
			result = fmt.Sprintf("工具执行失败: %s", execErr.Error())
		}

		messages = append(messages,
			Message{Role: "assistant", Content: fmt.Sprintf("调用工具: %s", resp.ToolName)},
			Message{Role: "tool", Content: result},
		)
	}

	return &Reply{Content: "抱歉，处理超时，请简化问题重试。"}, nil
}

func (a *Agent) parseResponse(raw string) (*LLMResponse, error) {
	// 提取 JSON（可能在 ```json``` 代码块中）
	s := raw
	if len(s) > 7 && s[:7] == "```json" {
		s = s[7:]
		if i := len(s) - 3; i > 0 && s[i:] == "```" {
			s = s[:i]
		}
	}
	var resp LLMResponse
	if err := json.Unmarshal([]byte(s), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (a *Agent) chatSystem() string {
	return "你是一个 JSON 响应机器人。你的每次回复必须是严格的 JSON 格式，不包含任何其他文字。"
}

func (a *Agent) chatUser(llmMsgs []string) string {
	content := ""
	for _, m := range llmMsgs {
		content += m + "\n"
	}
	content += "\n现在请根据以上对话决定下一步。只回复 JSON。"
	return content
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
