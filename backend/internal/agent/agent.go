package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"loledgeagent/internal/llm"
)

type Message struct {
	Role    string
	Content string
}

type Step struct {
	Round   int    `json:"round"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Tool    string `json:"tool,omitempty"`
}

type Reply struct {
	Content    string `json:"content"`
	ToolCalled string `json:"tool_called"`
	Steps      []Step `json:"steps,omitempty"`
}

const (
	maxRounds       = 8
	maxContextChars = 12000 // 上下文字符数上限，超过则截断中间消息
)

type Agent struct {
	client *llm.Client
	logger *slog.Logger
}

type agentTool struct {
	Name        string
	Description string
	Schema      string // JSON Schema string for the tool's parameters
	Execute     func(ctx context.Context, argsJSON string) (string, error)
}

func New(client *llm.Client, logger *slog.Logger) *Agent {
	return &Agent{client: client, logger: logger}
}

// Run 执行 Agent 对话
func (a *Agent) Run(ctx context.Context, history []Message, userMsg string) (*Reply, error) {
	if a.client == nil {
		return &Reply{Content: "LLM 未配置，请先在偏好设置中配置 API Key。"}, nil
	}

	tools := buildAgentTools()
	steps := make([]Step, 0)

	// 构建消息列表：system prompt + 截断后的历史 + 当前消息
	var messages []llm.ChatMessage
	messages = append(messages, llm.ChatMessage{Role: "system", Content: buildSystemPrompt(tools)})
	for _, m := range history {
		messages = append(messages, llm.ChatMessage{Role: m.Role, Content: m.Content})
	}
	messages = trimContext(messages) // 历史截断，保留 system prompt 头尾
	messages = append(messages, llm.ChatMessage{Role: "user", Content: userMsg})

	a.logger.Info("agent start", "user_msg", truncate(userMsg, 200))

	// Agent 循环
	for round := 1; round <= maxRounds; round++ {
		a.logger.Info("agent round", "round", round, "msg_count", len(messages))

		// 每轮前截断，防止 tool 结果累积撑爆上下文
		messages = trimContext(messages)

		llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		raw, err := a.client.ChatMessages(llmCtx, messages)
		cancel()
		if err != nil {
			steps = append(steps, Step{Round: round, Role: "error", Content: err.Error()})
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
				return &Reply{Content: "请求超时，请简化问题后重试。", Steps: steps}, nil
			}
			return &Reply{Content: fmt.Sprintf("请求失败: %v", err), Steps: steps}, nil
		}

		// 尝试解析 JSON 工具调用
		toolCall, ok := parseToolCall(raw)
		if ok {
			steps = append(steps, Step{Round: round, Role: "tool_call", Content: raw, Tool: toolCall.Tool})
			a.logger.Info("agent tool call", "tool", toolCall.Tool, "args", toolCall.Args)

			// 查找并执行工具
			var result string
			for _, t := range tools {
				if t.Name == toolCall.Tool {
					result, err = t.Execute(ctx, toolCall.Args)
					if err != nil {
						result = fmt.Sprintf("错误: %s", err.Error())
					}
					break
				}
			}
			if result == "" && err == nil {
				result = fmt.Sprintf("工具 %s 不存在", toolCall.Tool)
			}
			err = nil // 重置 err

			steps = append(steps, Step{Round: round, Role: "tool", Content: truncate(result, 300), Tool: toolCall.Tool})

			messages = append(messages, llm.ChatMessage{Role: "assistant", Content: raw})
			messages = append(messages, llm.ChatMessage{Role: "user", Content: fmt.Sprintf("工具 %s 执行结果:\n%s", toolCall.Tool, result)})
			continue
		}

		// 不是工具调用 → 最终回答
		steps = append(steps, Step{Round: round, Role: "model", Content: raw})
		a.logger.Info("agent final", "round", round)

		var toolCalled string
		for i := len(steps) - 1; i >= 0; i-- {
			if steps[i].Role == "tool" {
				toolCalled = steps[i].Tool
				break
			}
		}
		return &Reply{Content: raw, ToolCalled: toolCalled, Steps: steps}, nil
	}

	return &Reply{Content: "抱歉，处理超时，请简化问题重试。", Steps: steps}, nil
}

// toolCallRequest LLM 返回的 JSON 工具调用
type toolCallRequest struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
}

func parseToolCall(raw string) (*toolCallRequest, bool) {
	s := strings.TrimSpace(raw)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	s = s[start : end+1]

	var req toolCallRequest
	if err := json.Unmarshal([]byte(s), &req); err != nil {
		return nil, false
	}
	if req.Tool == "" {
		return nil, false
	}
	return &req, true
}

// ---- 工具 ----

func buildAgentTools() []agentTool {
	tools := allEinoTools()
	var result []agentTool
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			continue
		}
		at := agentTool{
			Name:        info.Name,
			Description: info.Desc,
			Execute: func(ctx context.Context, argsJSON string) (string, error) {
				return t.InvokableRun(ctx, argsJSON)
			},
		}
		// 序列化 tool schema
		schemaJSON, _ := json.Marshal(info)
		at.Schema = string(schemaJSON)
		result = append(result, at)
	}
	return result
}

func buildSystemPrompt(tools []agentTool) string {
	var sb strings.Builder
	sb.WriteString(`你是 LolEdgeAgent，一个内容聚合和知识助手。

## 可用工具
`)

	for _, t := range tools {
		fmt.Fprintf(&sb, "- **%s**: %s\n", t.Name, t.Description)
	}

	sb.WriteString(`
## 回复格式
需要调用工具时：
{"tool":"<工具名>","args":"<JSON 参数>"}

直接回答时，正常用 Markdown 回复。

## 规则
1. 用户说"查看简报""最近的简报"→ 调用 list_briefings
2. 查看具体简报 → 调用 get_briefing
3. 搜索文章 → 调用 search_articles
4. 生成新简报 → 调用 generate_briefing
5. 闲聊问候不需要工具，直接回答
6. 用中文回复，引用文章时带标题和链接
`)

	return sb.String()
}

// trimContext 截断中间消息，保留 system prompt 和最近轮次，防止 token 失控
func trimContext(msgs []llm.ChatMessage) []llm.ChatMessage {
	if len(msgs) <= 5 {
		return msgs
	}
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}
	if total <= maxContextChars {
		return msgs
	}
	// 保留 system prompt(第一条) + 最近 4 条
	keep := 4
	if len(msgs)-keep < 1 {
		keep = len(msgs) - 1
	}
	result := make([]llm.ChatMessage, 0, keep+2)
	result = append(result, msgs[0]) // system prompt
	result = append(result, llm.ChatMessage{Role: "system", Content: "…(中间消息已截断)…"})
	result = append(result, msgs[len(msgs)-keep:]...)
	return result
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
