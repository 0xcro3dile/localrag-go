// Package embedding provides the Ollama embedding adapter.
// Clean Architecture: This is an adapter that implements ports.EmbeddingService.
// It knows about Ollama specifics but the domain layer doesn't.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// OllamaAdapter implements ports.EmbeddingService using Ollama API.
type OllamaAdapter struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaAdapter creates a new Ollama embedding adapter.
func NewOllamaAdapter(baseURL, model string) *OllamaAdapter {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaAdapter{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ollamaEmbedRequest is the Ollama API request format.
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the Ollama API response format.
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed generates an embedding for a single text.
func (a *OllamaAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	log.Printf("[DEBUG] Embedding request to %s with model %s", a.baseURL, a.model)
	
	reqBody := ollamaEmbedRequest{
		Model:  a.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[ERROR] Marshal error: %v", err)
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("[ERROR] Request create error: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[DEBUG] Calling Ollama at %s/api/embeddings...", a.baseURL)
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Ollama call error: %v", err)
		return nil, fmt.Errorf("calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] Ollama responded with status %d", resp.StatusCode)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		log.Printf("[ERROR] Decode error: %v", err)
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	log.Printf("[OK] Got embedding with %d dimensions", len(embedResp.Embedding))
	return embedResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
// Currently calls Embed sequentially - can be parallelized if needed.
func (a *OllamaAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := a.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embedding text %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}
