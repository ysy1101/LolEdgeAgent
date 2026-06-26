package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/embedding"
)

type OllamaEmbedder struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaEmbedder(baseURL, model string) embedding.Embedder {
	return &OllamaEmbedder{
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *OllamaEmbedder) EmbedStrings(ctx context.Context, texts []string, _ ...embedding.Option) ([][]float64, error) {
	result := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vec, err := e.embedOne(ctx, text)
		if err != nil {
			return nil, err
		}
		result = append(result, vec)
	}
	return result, nil
}

func (e *OllamaEmbedder) embedOne(ctx context.Context, text string) ([]float64, error) {
	body := map[string]string{
		"model":  e.model,
		"prompt": text,
	}
	jsonBody, _ := json.Marshal(body)

	url := e.baseURL + "/api/embeddings"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama embed %d: %s", resp.StatusCode, truncateStr(string(data), 200))
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("ollama parse: %w, body: %s", err, truncateStr(string(data), 100))
	}

	return result.Embedding, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
