package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

// ---- Node 类型 ----

type NodeType string

const (
	NodeStart        NodeType = "START"
	NodeSystemPrompt NodeType = "SYSTEM_PROMPT"
	NodeModelCall    NodeType = "MODEL_CALL"
	NodeToolExec     NodeType = "TOOL_EXEC"
	NodeFinal        NodeType = "FINAL"
	NodeError        NodeType = "ERROR"
)

// ---- AgentState: 贯穿整个 Agent 生命周期的状态 ----

type AgentState struct {
	// Request 层（不变）
	SessionID string `json:"session_id"`
	UserID    uint   `json:"user_id"`
	Question  string `json:"question"`

	// Runtime 层（随节点流转更新）
	State      StateType    `json:"state"`
	Round      int          `json:"round"`
	Messages   []ChatRecord `json:"messages"`  // 压缩版消息记录
	Tokens     TokenUsage   `json:"tokens"`
	ToolCalls  []ToolRecord `json:"tool_calls"`
	Context    KVStore      `json:"context"`   // 键值对，节点间共享
	Transitions []Transition `json:"transitions"` // 状态流转记录

	// Result 层（最终填充）
	Content      string        `json:"content"`
	TotalRounds  int           `json:"total_rounds"`
	TotalLatency time.Duration `json:"total_latency_ms"`
	Status       RunStatus     `json:"status"`
	ErrorMessage string        `json:"error_message,omitempty"`

	// internal
	startTime time.Time `json:"-"`
	nodeStart time.Time `json:"-"`
}

type StateType string

const (
	StateIdle     StateType = "idle"
	StateThinking StateType = "thinking"
	StateActing   StateType = "acting" // executing tools
	StateFinal    StateType = "final"
	StateError    StateType = "error"
)

type RunStatus string

const (
	StatusCompleted RunStatus = "completed"
	StatusTimeout   RunStatus = "timeout"
	StatusMaxRounds RunStatus = "max_rounds"
	StatusError     RunStatus = "error"
)

// ---- 子结构 ----

type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
}

type ToolRecord struct {
	Round   int    `json:"round"`
	Name    string `json:"name"`
	Args    string `json:"args"`
	Result  string `json:"result,omitempty"`
	Success bool   `json:"success"`
	Latency int64  `json:"latency_ms"`
}

type ChatRecord struct {
	Role    string `json:"role"`
	Length  int    `json:"length"` // 字符数
	HasTools bool  `json:"has_tools,omitempty"`
}

type Transition struct {
	From  NodeType `json:"from"`
	To    NodeType `json:"to"`
	Round int      `json:"round"`
	At    string   `json:"at"` // RFC3339
}

// KVStore 节点间共享键值存储
type KVStore map[string]any

func (kv KVStore) Set(key string, val any) { kv[key] = val }
func (kv KVStore) Get(key string) any       { return kv[key] }
func (kv KVStore) GetString(key string) string {
	if v, ok := kv[key].(string); ok {
		return v
	}
	return ""
}

// ---- 构造函数 ----

func NewAgentState(sessionID string, userID uint, question string) *AgentState {
	return &AgentState{
		SessionID:   sessionID,
		UserID:      userID,
		Question:    question,
		State:       StateIdle,
		Context:     make(KVStore),
		Transitions: make([]Transition, 0),
	}
}

// ---- 流转方法（每个节点调用） ----

// Enter 进入新节点
func (s *AgentState) Enter(node NodeType) {
	if s.nodeStart.IsZero() {
		s.nodeStart = s.startTime
	}
	from := s.nodeType()
	s.MarkTransition(from, node, s.Round)
	s.nodeStart = time.Now()
}

// MarkTransition 记录状态流转
func (s *AgentState) MarkTransition(from, to NodeType, round int) {
	s.Transitions = append(s.Transitions, Transition{
		From:  from,
		To:    to,
		Round: round,
		At:    time.Now().Format(time.RFC3339Nano),
	})
}

func (s *AgentState) nodeType() NodeType {
	if len(s.Transitions) == 0 {
		return NodeStart
	}
	return s.Transitions[len(s.Transitions)-1].To
}

// CurrentNode 当前所在节点
func (s *AgentState) CurrentNode() NodeType {
	if len(s.Transitions) == 0 {
		return NodeStart
	}
	return s.Transitions[len(s.Transitions)-1].To
}

// RecordToolCall 记录工具调用
func (s *AgentState) RecordToolCall(name, args string) {
	s.State = StateActing
	s.ToolCalls = append(s.ToolCalls, ToolRecord{
		Round: s.Round,
		Name:  name,
		Args:  args,
	})
}

// RecordToolResult 记录工具结果
func (s *AgentState) RecordToolResult(name, result string, success bool, latency time.Duration) {
	if len(s.ToolCalls) > 0 {
		last := &s.ToolCalls[len(s.ToolCalls)-1]
		last.Result = truncate(result, 200)
		last.Success = success
		last.Latency = latency.Milliseconds()
	}
}

// RecordMessage 记录消息元数据（不存全量内容）
func (s *AgentState) RecordMessage(role, content string, hasTools bool) {
	s.Messages = append(s.Messages, ChatRecord{
		Role:     role,
		Length:   len(content),
		HasTools: hasTools,
	})
}

// AddTokens 累加 token
func (s *AgentState) AddTokens(prompt, completion int) {
	s.Tokens.Prompt += prompt
	s.Tokens.Completion += completion
}

// Finalize 结束
func (s *AgentState) Finalize(content string, totalRounds int) {
	s.State = StateFinal
	s.Status = StatusCompleted
	s.Content = content
	s.TotalRounds = totalRounds
	s.TotalLatency = time.Since(s.startTime)
	s.MarkTransition(s.CurrentNode(), NodeFinal, s.Round)
}

// FinalizeError 异常结束
func (s *AgentState) FinalizeError(err error, status RunStatus) {
	s.State = StateError
	s.Status = status
	s.ErrorMessage = err.Error()
	s.TotalLatency = time.Since(s.startTime)
	s.MarkTransition(s.CurrentNode(), NodeError, s.Round)
}

// Step 转换为前端可视化 Step（兼容现有接口）
func (s *AgentState) ToSteps() []Step {
	steps := make([]Step, 0)
	for _, tc := range s.ToolCalls {
		steps = append(steps, Step{
			Round:   tc.Round,
			Role:    "tool_call",
			Content: tc.Name + "(" + tc.Args + ")",
			Tool:    tc.Name,
		})
		if tc.Result != "" {
			steps = append(steps, Step{
				Round:   tc.Round,
				Role:    "tool",
				Content: tc.Result,
				Tool:    tc.Name,
			})
		}
	}
	if s.Content != "" {
		steps = append(steps, Step{
			Round:   s.TotalRounds,
			Role:    "model",
			Content: s.Content,
		})
	}
	return steps
}

// Summary 生成人类可读摘要
func (s *AgentState) Summary() string {
	b, _ := json.Marshal(map[string]any{
		"session_id":  s.SessionID,
		"user_id":     s.UserID,
		"status":      s.Status,
		"rounds":      s.TotalRounds,
		"tool_calls":  len(s.ToolCalls),
		"tokens":      s.Tokens,
		"latency_ms":  s.TotalLatency.Milliseconds(),
		"transitions": len(s.Transitions),
	})
	return string(b)
}

// Graphviz 输出状态流转的 DOT 图
func (s *AgentState) Graphviz() string {
	var out string
	out += "digraph AgentState {\n"
	out += "  rankdir=LR;\n"
	out += "  node [shape=box, style=rounded];\n"
	for i, t := range s.Transitions {
		out += "  \"" + string(t.From) + "_" + itoa(i) + "\" [label=\"" + string(t.From) + "\"];\n"
		out += "  \"" + string(t.From) + "_" + itoa(i) + "\" -> \"" + string(t.To) + "_" + itoa(i+1) + "\"";
		if t.Round > 0 {
			out += " [label=\"round " + itoa(t.Round) + "\"]"
		}
		out += ";\n"
	}
	out += "}\n"
	return out
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
