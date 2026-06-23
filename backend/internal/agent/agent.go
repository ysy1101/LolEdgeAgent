package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"loledgeagent/internal/llm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type Message struct {
	Role    string
	Content string
}

type Reply struct {
	Content    string `json:"content"`
	ToolCalled string `json:"tool_called"`
}

type Agent struct {
	reactAgent *react.Agent
	logger     *slog.Logger
}

func New(client *llm.Client, logger *slog.Logger) *Agent {
	if client == nil {
		return &Agent{logger: logger}
	}

	tools := allEinoTools()
	var baseTools []tool.BaseTool
	for _, t := range tools {
		baseTools = append(baseTools, t)
	}

	ra, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: client.ChatModel(),
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: baseTools,
		},
		MaxStep: 8,
	})
	if err != nil {
		panic("failed to create agent: " + err.Error())
	}

	return &Agent{reactAgent: ra, logger: logger}
}

// Run 执行 Agent 对话
func (a *Agent) Run(ctx context.Context, history []Message, userMsg string) (*Reply, error) {
	if a.reactAgent == nil {
		return &Reply{Content: "LLM 未配置，请先在偏好设置中配置 API Key。"}, nil
	}

	messages := []*schema.Message{schema.SystemMessage(systemPrompt())}
	messages = append(messages, historyToMessages(history)...)
	messages = append(messages, schema.UserMessage(userMsg))

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := a.reactAgent.Generate(llmCtx, messages)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
			return &Reply{Content: "请求超时，请简化问题后重试。"}, nil
		}
		return nil, fmt.Errorf("agent: %w", err)
	}

	return &Reply{Content: result.Content}, nil
}

func historyToMessages(history []Message) []*schema.Message {
	var msgs []*schema.Message
	for _, m := range history {
		switch m.Role {
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
		case "system":
			msgs = append(msgs, schema.SystemMessage(m.Content))
		default:
			msgs = append(msgs, schema.UserMessage(m.Content))
		}
	}
	return msgs
}

func systemPrompt() string {
	return `你是 LolEdgeAgent，一个内容聚合和知识助手。

## 规则
1. 如果用户问题不需要工具（闲聊、问候），直接回答
2. 用中文回复
3. 引用搜索到的文章时带上标题和链接
4. 如果工具执行失败，告知用户并给出替代建议`
}
