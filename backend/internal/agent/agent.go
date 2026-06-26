package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"loledgeagent/internal/llm"
	"loledgeagent/internal/repository"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Message 历史消息（与前端交互用）
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Step Agent 思考步骤（前端可视化用）
type Step struct {
	Round   int    `json:"round"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Tool    string `json:"tool,omitempty"`
}

// Reply Agent 回复
type Reply struct {
	Content    string `json:"content"`
	ToolCalled string `json:"tool_called"`
	Steps      []Step `json:"steps,omitempty"`
}

const (
	maxRounds        = 8
	estimationFactor = 2 // 中文字符 ≈ 2-3 token，保守按 2 char/token
	maxTokens        = 8000
)

// Agent 对话引擎
type Agent struct {
	defaultCfg llm.Config
	prefRepo   *repository.PreferenceRepo
	logger     *slog.Logger
}

// New 创建 Agent
func New(defaultCfg llm.Config, prefRepo *repository.PreferenceRepo, logger *slog.Logger) *Agent {
	return &Agent{defaultCfg: defaultCfg, prefRepo: prefRepo, logger: logger}
}

// Run 执行 Agent 对话
func (a *Agent) Run(ctx context.Context, history []Message, userMsg string) (*Reply, error) {
	tools := allEinoTools()
	steps := make([]Step, 0)

	// --- 创建 LLM Client（从 DB 读取最新配置，支持热加载）---
	client, err := a.buildClient(ctx)
	if err != nil {
		return &Reply{Content: "LLM 未配置，请先在偏好设置中配置 API Key。"}, nil
	}
	chatModel := client.ChatModel()

	// 获取工具信息列表（传给 LLM 的 tool schema）
	toolInfos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err == nil {
			toolInfos = append(toolInfos, info)
		}
	}

	// --- 构建消息列表 ---
	msgs := buildMessages(history, userMsg)

	a.logger.Info("agent start", "user_msg", truncate(userMsg, 200), "history_msgs", len(history))

	// --- Agent 循环（原生 Function Calling）---
	for round := 1; round <= maxRounds; round++ {
		// 每轮前截断，防止上下文溢出
		msgs = trimContext(msgs)
		a.logger.Info("agent round", "round", round, "msg_count", len(msgs))

		llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, err := chatModel.Generate(llmCtx, msgs, model.WithTools(toolInfos))
		cancel()
		if err != nil {
			steps = append(steps, Step{Round: round, Role: "error", Content: err.Error()})
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
				return &Reply{Content: "请求超时，请简化问题后重试。", Steps: steps}, nil
			}
			return &Reply{Content: fmt.Sprintf("请求失败: %v", err), Steps: steps}, nil
		}

		// 检查是否有工具调用（原生 ToolCalls 字段）
		if len(resp.ToolCalls) > 0 {
			for _, tc := range resp.ToolCalls {
				steps = append(steps, Step{
					Round:   round,
					Role:    "tool_call",
					Content: fmt.Sprintf("%s(%s)", tc.Function.Name, tc.Function.Arguments),
					Tool:    tc.Function.Name,
				})
				a.logger.Info("agent tool call", "tool", tc.Function.Name, "args", tc.Function.Arguments)
			}

			// ✅ 将 assistant 消息（含 ToolCalls）追加到历史
			// 防御性拷贝：resp 是 Generate 返回的指针，后续可能被内部复用
			assistantMsg := &schema.Message{
				Role:       resp.Role,
				Content:    resp.Content,
				ToolCalls:  resp.ToolCalls,
				ToolCallID: resp.ToolCallID,
			}
			msgs = append(msgs, assistantMsg)
			a.logger.Info("agent tool calls in round",
				"count", len(resp.ToolCalls),
				"msg_count_after", len(msgs))

			// 执行每个工具，结果用 ✅ tool 角色 + tool_call_id
			for _, tc := range resp.ToolCalls {
				result := executeToolCall(ctx, tc.Function.Name, tc.Function.Arguments, tools, a.logger)

				// 某些提供商的 tool_call_id 可能为空，生成回退 ID
				toolCallID := tc.ID
				if toolCallID == "" {
					toolCallID = fmt.Sprintf("call_%s_%d", tc.Function.Name, round)
					a.logger.Warn("empty tool_call_id, using fallback", "tool", tc.Function.Name, "fallback_id", toolCallID)
				}

				steps = append(steps, Step{
					Round:   round,
					Role:    "tool",
					Content: truncate(result, 300),
					Tool:    tc.Function.Name,
				})

				msgs = append(msgs, schema.ToolMessage(result, toolCallID))
			}
			continue
		}

		// 无工具调用 → 最终回答
		steps = append(steps, Step{Round: round, Role: "model", Content: resp.Content})
		a.logger.Info("agent final", "round", round)

		var toolCalled string
		for i := len(steps) - 1; i >= 0; i-- {
			if steps[i].Role == "tool" {
				toolCalled = steps[i].Tool
				break
			}
		}
		return &Reply{Content: resp.Content, ToolCalled: toolCalled, Steps: steps}, nil
	}

	return &Reply{Content: "抱歉，处理步骤超限，请简化问题重试。", Steps: steps}, nil
}

// executeToolCall 执行工具调用（传入 ctx 以保留 user_id）
func executeToolCall(ctx context.Context, name, argsJSON string, tools []tool.InvokableTool, logger *slog.Logger) string {
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			logger.Warn("tool info failed", "tool", name, "error", err)
			continue
		}
		if info.Name == name {
			result, err := t.InvokableRun(ctx, argsJSON)
			if err != nil {
				return fmt.Sprintf("错误: %s", err.Error())
			}
			return result
		}
	}
	return fmt.Sprintf("工具 %s 不存在", name)
}

// buildClient 从 DB 偏好 + 环境默认值创建 LLM Client（热加载）
func (a *Agent) buildClient(ctx context.Context) (*llm.Client, error) {
	cfg := a.defaultCfg

	pref, err := a.prefRepo.Get(getUserID(ctx))
	if err == nil && pref != nil {
		if pref.LLMAPIKey != "" {
			cfg.APIKey = pref.LLMAPIKey
		}
		if pref.LLMModel != "" {
			cfg.Model = pref.LLMModel
		}
		if pref.LLMBaseURL != "" {
			cfg.BaseURL = pref.LLMBaseURL
		}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM API Key 未配置")
	}

	return llm.NewClient(cfg), nil
}

// buildMessages 构建消息列表
func buildMessages(history []Message, userMsg string) []*schema.Message {
	var msgs []*schema.Message

	// System prompt（只含业务规则，工具通过原生 Function Calling 传递）
	msgs = append(msgs, schema.SystemMessage(buildSystemPrompt()))

	// 历史消息
	for _, m := range history {
		switch m.Role {
		case "user":
			msgs = append(msgs, schema.UserMessage(m.Content))
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
		}
	}

	// 当前用户消息
	msgs = append(msgs, schema.UserMessage(userMsg))

	return msgs
}

// buildSystemPrompt 构建系统提示（只含业务规则，工具通过原生 Function Calling 传递）
func buildSystemPrompt() string {
	return `你是 LolEdgeAgent，一个内容聚合和知识助手。

## 回复规则
1. 用户说"查看简报""最近的简报"→ 调用 list_briefings
2. 查看具体简报 → 调用 get_briefing
3. 搜索文章 → 调用 search_articles
4. 生成新简报 → 调用 generate_briefing
5. 闲聊问候不需要工具，直接回答
6. 用中文回复，引用文章时带标题和链接`
}

// trimContext 上下文截断（按 token 估算，保留 system + 最近的完整轮次）
// 关键：不能拆散 tool_calls 和对应的 tool 响应，否则 API 校验失败
func trimContext(msgs []*schema.Message) []*schema.Message {
	if len(msgs) <= 3 {
		return msgs
	}

	totalTokens := 0
	for _, m := range msgs {
		totalTokens += len(m.Content) / estimationFactor
	}

	if totalTokens <= maxTokens {
		return msgs
	}

	system := msgs[0]
	var recent []*schema.Message
	tokensUsed := len(system.Content) / estimationFactor

	// 从后往前加，按轮次边界截断（tool_calls + tool 响应不可拆分）
	pendingTools := 0 // 未配对 tool 响应计数器
	for i := len(msgs) - 1; i >= 1; i-- {
		m := msgs[i]
		msgTokens := len(m.Content) / estimationFactor

		if m.Role == schema.Tool {
			pendingTools++
		} else if m.Role == schema.Assistant && len(m.ToolCalls) > 0 {
			pendingTools -= len(m.ToolCalls)
		}

		if tokensUsed+msgTokens <= maxTokens {
			recent = append([]*schema.Message{m}, recent...)
			tokensUsed += msgTokens
		} else {
			// 如果还没凑够配对，继续保留
			if pendingTools > 0 {
				recent = append([]*schema.Message{m}, recent...)
				tokensUsed += msgTokens
			}
			// 配对完成，停止
			if pendingTools <= 0 && m.Role != schema.Tool && len(m.ToolCalls) == 0 {
				break
			}
		}
	}

	result := make([]*schema.Message, 0, 2+len(recent))
	result = append(result, system)
	if len(msgs)-1 > len(recent) {
		result = append(result, schema.SystemMessage("…(中间消息已截断)…"))
	}
	result = append(result, recent...)

	return result
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
