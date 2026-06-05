package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Config LLM 配置
type Config struct {
	Model   string
	APIKey  string
	BaseURL string
}

// LoadConfig 从环境变量加载 LLM 配置
func LoadConfig() Config {
	return Config{
		Model:   envOrDefault("LLM_MODEL", "deepseek-chat"),
		APIKey:  os.Getenv("LLM_API_KEY"),
		BaseURL: baseURLOrDefault(),
	}
}

func baseURLOrDefault() string {
	if u := os.Getenv("LLM_BASE_URL"); u != "" {
		return u
	}
	if p := os.Getenv("LLM_PROVIDER"); p == "deepseek" {
		return "https://api.deepseek.com"
	}
	return "https://api.openai.com"
}

// Client 封装 OpenAI 兼容的 Chat Completions API
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

// Chat 发送 system + user 消息，返回 assistant 回复
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	req := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	return c.doChat(ctx, req)
}

// ChatJSON 发送消息并将回复解析为 JSON 结构体
func (c *Client) ChatJSON(ctx context.Context, system, user string, result any) error {
	content, err := c.Chat(ctx, system, user)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(content), result)
}

func (c *Client) doChat(ctx context.Context, body chatRequest) (string, error) {
	jsonBody, _ := json.Marshal(body)

	url := c.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("llm api error %d: %s", resp.StatusCode, truncateStr(string(data), 300))
	}

	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return normalizeJSON(cr.Choices[0].Message.Content), nil
}

// Embeddings 调用 embedding API 获取文本向量
func (c *Client) Embeddings(ctx context.Context, texts []string) ([][]float64, error) {
	type embReq struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}
	type embData struct {
		Embedding []float64 `json:"embedding"`
	}
	type embResp struct {
		Data []embData `json:"data"`
	}

	jsonBody, _ := json.Marshal(embReq{Model: c.model, Input: texts})
	url := c.baseURL + "/v1/embeddings"
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embeddings request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embeddings api error %d: %s", resp.StatusCode, truncateStr(string(data), 300))
	}

	var er embResp
	if err := json.Unmarshal(data, &er); err != nil {
		return nil, fmt.Errorf("parse embeddings: %w", err)
	}
	result := make([][]float64, len(er.Data))
	for i, d := range er.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

// normalizeJSON 从 LLM 回复中提取 JSON 数组（去除可能的 markdown 包裹）
func normalizeJSON(s string) string {
	s = trimSpace(s)
	if len(s) >= 6 && s[:7] == "```json" {
		s = s[7:]
		if idx := lastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if len(s) >= 3 && s[:3] == "```" {
		s = s[3:]
		if idx := lastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return trimSpace(s)
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\r' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func lastIndex(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
