package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbeddingClient 独立 embedding HTTP 客户端
type EmbeddingClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewEmbeddingClient(cfg Config) *EmbeddingClient {
	return &EmbeddingClient{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
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

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
