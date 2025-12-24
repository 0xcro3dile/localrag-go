// Package llm provides the Ollama LLM adapter.
// Clean Architecture: Adapter implementing ports.LLMService.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

// OllamaLLMAdapter implements ports.LLMService using Ollama API.
type OllamaLLMAdapter struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaLLMAdapter creates a new Ollama LLM adapter.
func NewOllamaLLMAdapter(baseURL, model string) *OllamaLLMAdapter {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaLLMAdapter{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 300 * time.Second, // Longer timeout for streaming
		},
	}
}

// ollamaGenerateRequest is the Ollama generate API request.
type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaGenerateResponse is the Ollama generate API response.
type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate produces a response given a prompt and context.
func (a *OllamaLLMAdapter) Generate(ctx context.Context, prompt string, context []string) (string, error) {
	reqBody := ollamaGenerateRequest{
		Model:  a.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/generate", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return genResp.Response, nil
}

// GenerateStream produces a real streaming response via Ollama's streaming API.
// Returns a channel of StreamTokens for real-time UI updates.
func (a *OllamaLLMAdapter) GenerateStream(ctx context.Context, prompt string, context []string) (<-chan ports.StreamToken, error) {
	reqBody := ollamaGenerateRequest{
		Model:  a.model,
		Prompt: prompt,
		Stream: true, // Enable streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/generate", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Ollama: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	ch := make(chan ports.StreamToken, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				ch <- ports.StreamToken{Done: true, Error: ctx.Err()}
				return
			default:
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaGenerateResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				continue // Skip malformed lines
			}

			ch <- ports.StreamToken{
				Content: chunk.Response,
				Done:    chunk.Done,
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- ports.StreamToken{Done: true, Error: err}
		}
	}()

	return ch, nil
}

