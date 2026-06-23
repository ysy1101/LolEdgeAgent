package llm

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	openaiembed "github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/schema"
)

// Config LLM 配置
type Config struct {
	Model            string
	EmbeddingModel   string
	APIKey           string
	BaseURL          string
	EmbeddingBaseURL string
}

// LoadConfig 从环境变量加载配置
func LoadConfig() Config {
	key := os.Getenv("LLM_API_KEY")
	if key == "" {
		key = os.Getenv("API_KEY")
	}
	return Config{
		Model:            envOrDefault("LLM_MODEL", "deepseek-chat"),
		EmbeddingModel:   envOrDefault("EMBEDDING_MODEL", "text-embedding-3-small"),
		APIKey:           key,
		BaseURL:          baseURLOrDefault(),
		EmbeddingBaseURL: embeddingBaseURLOrDefault(),
	}
}

func baseURLOrDefault() string {
	if u := os.Getenv("LLM_BASE_URL"); u != "" {
		return u
	}
	return "https://api.deepseek.com"
}

func embeddingBaseURLOrDefault() string {
	if u := os.Getenv("EMBEDDING_BASE_URL"); u != "" {
		return u
	}
	return "https://api.openai.com"
}

// Client 封装 Eino ToolCallingChatModel + Embedder
type Client struct {
	chatModel model.ToolCallingChatModel
	model     string
	embedder  embedding.Embedder
}

// NewClient 创建客户端（ChatModel + Embedder 均通过 Eino）
func NewClient(cfg Config) *Client {
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		panic("failed to create chat model: " + err.Error())
	}

	emb, err := openaiembed.NewEmbedder(context.Background(), &openaiembed.EmbeddingConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.EmbeddingModel,
		BaseURL: cfg.EmbeddingBaseURL,
	})
	if err != nil {
		panic("failed to create embedder: " + err.Error())
	}

	return &Client{
		chatModel: cm,
		model:     cfg.Model,
		embedder:  emb,
	}
}

// ChatModel 返回原生 ToolCallingChatModel，供 Agent 使用
func (c *Client) ChatModel() model.ToolCallingChatModel {
	return c.chatModel
}

// Chat 发送 system + user 消息
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	resp, err := c.chatModel.Generate(ctx, []*schema.Message{
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
	resp, err := c.chatModel.Generate(ctx, messages)
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

// Embeddings 生成文本向量
func (c *Client) Embeddings(ctx context.Context, texts []string) ([][]float64, error) {
	return c.embedder.EmbedStrings(ctx, texts)
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

// NormalizeJSON 从 LLM 回复中提取 JSON
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
