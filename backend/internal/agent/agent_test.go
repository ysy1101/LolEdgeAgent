package agent

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// TestBuildSystemPrompt 验证系统提示不含工具 Schema 描述
func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt()

	// 系统提示不应包含工具的参数描述/JSON Schema（工具通过原生 Function Calling 传递）
	if contains(prompt, "jsonschema") {
		t.Error("system prompt should not contain tool JSON Schema")
	}
	if contains(prompt, "required") && !contains(prompt, "回复") {
		t.Error("system prompt should not contain tool parameter schemas")
	}

	// 但工具名称出现在业务规则中是正常的
	if !contains(prompt, "generate_briefing") {
		t.Error("tool names in business rules are fine")
	}

	// 应该包含业务规则
	if !contains(prompt, "LolEdgeAgent") {
		t.Error("system prompt should contain agent name")
	}
	if !contains(prompt, "中文") {
		t.Error("system prompt should specify Chinese language")
	}
}

// TestTrimContext 验证上下文截断正确
func TestTrimContext(t *testing.T) {
	// 构造一个长上下文
	msgs := []*schema.Message{
		schema.SystemMessage("你是 LolEdgeAgent"),
		schema.UserMessage("你好"),
		schema.AssistantMessage("你好！有什么可以帮助你的？", nil),
	}

	// 3 条消息不应截断
	result := trimContext(msgs)
	if len(result) != 3 {
		t.Errorf("short context should not be truncated: got %d, want 3", len(result))
	}

	// 构造超出限制的上下文
	longContent := string(make([]byte, maxTokens*3)) // 远超限制
	for i := 0; i < 20; i++ {
		msgs = append(msgs, schema.UserMessage(longContent))
		msgs = append(msgs, schema.AssistantMessage(longContent, nil))
	}

	result = trimContext(msgs)
	if len(result) >= len(msgs) {
		t.Errorf("oversized context should be truncated: got %d, want < %d", len(result), len(msgs))
	}

	// system prompt 必须保留
	if result[0].Role != "system" {
		t.Error("first message must be system prompt")
	}

	// 截断标记应存在（如果中间消息被丢弃）
	hasTruncationMarker := false
	for _, m := range result {
		if m.Content == "…(中间消息已截断)…" {
			hasTruncationMarker = true
			break
		}
	}
	if len(msgs) != len(result) && !hasTruncationMarker {
		t.Error("truncated context should contain truncation marker")
	}
}

// TestTruncate 验证截断函数
func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("short string should not be truncated")
	}
	if truncate("hello world", 5) != "hello..." {
		t.Error("long string should be truncated with ellipsis")
	}
	if truncate("", 5) != "" {
		t.Error("empty string should stay empty")
	}
}

// TestBuildMessages 验证消息构建
func TestBuildMessages(t *testing.T) {
	history := []Message{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}
	userMsg := "查看最近的简报"

	msgs := buildMessages(history, userMsg)

	// 第一条必须是 system
	if msgs[0].Role != "system" {
		t.Error("first message must be system")
	}

	// 最后一条必须是当前用户消息
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "user" || lastMsg.Content != userMsg {
		t.Errorf("last message should be user message: got role=%s, content=%s", lastMsg.Role, lastMsg.Content)
	}

	// 历史消息数量正确
	expectedLen := 1 + len(history) + 1 // system + history + current
	if len(msgs) != expectedLen {
		t.Errorf("message count: got %d, want %d", len(msgs), expectedLen)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
