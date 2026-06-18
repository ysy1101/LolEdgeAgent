package llm

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/model"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// Config LLM 配置
type Config struct {
	Model   string
	APIKey  string
	BaseURL string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() Config {
	key := os.Getenv("LLM_API_KEY")
	if key == "" {
		key = os.Getenv("API_KEY")
	}
	return Config{
		Model:   envOrDefault("LLM_MODEL", "deepseek-chat"),
		APIKey:  key,
		BaseURL: baseURLOrDefault(),
	}
}

func baseURLOrDefault() string {
	if u := os.Getenv("LLM_BASE_URL"); u != "" {
		return u
	}
	return "https://api.deepseek.com"
}

// Client 封装 Eino ChatModel + HTTP Embedding
type Client struct {
	cm     model.ChatModel
	model  string
	embCli *EmbeddingClient
}

// NewClient 创建客户端
func NewClient(cfg Config) *Client {
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		panic("failed to create chat model: " + err.Error())
	}
	return &Client{
		cm:     cm,
		model:  cfg.Model,
		embCli: NewEmbeddingClient(cfg),
	}
}

// Chat 发送 system + user 消息
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	resp, err := c.cm.Generate(ctx, []*schema.Message{
		schema.SystemMessage(system),
		schema.UserMessage(user),
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ChatMessages 发送多轮消息
func (c *Client) ChatMessages(ctx context.Context, msgs []ChatMessage) (string, error) {
	messages := make([]*schema.Message, len(msgs))
	for i, m := range msgs {
		switch m.Role {
		case "system":
			messages[i] = schema.SystemMessage(m.Content)
		case "user":
			messages[i] = schema.UserMessage(m.Content)
		case "assistant":
			messages[i] = schema.AssistantMessage(m.Content, nil)
		default:
			messages[i] = schema.UserMessage(m.Content)
		}
	}
	resp, err := c.cm.Generate(ctx, messages)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// ChatJSON 发送消息并解析 JSON 回复
func (c *Client) ChatJSON(ctx context.Context, system, user string, result any) error {
	raw, err := c.Chat(ctx, system, user)
	if err != nil {
		return err
	}
	s := NormalizeJSON(raw)
	return json.Unmarshal([]byte(s), result)
}

// Embeddings 委托给 EmbeddingClient
func (c *Client) Embeddings(ctx context.Context, texts []string) ([][]float64, error) {
	return c.embCli.Embed(ctx, texts)
}

// ChatMessage 消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// normalizeJSON 从 LLM 回复中提取 JSON
func NormalizeJSON(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 && s[:7] == "```json" {
		s = s[7:]
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if len(s) >= 3 && s[:3] == "```" {
		s = s[3:]
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}
