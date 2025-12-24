package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaLLM_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "Hello there!",
			"done":     true,
		})
	}))
	defer server.Close()

	adapter := NewOllamaLLMAdapter(server.URL, "test-model")
	resp, err := adapter.Generate(context.Background(), "Hi", nil)

	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if resp != "Hello there!" {
		t.Errorf("unexpected response: %s", resp)
	}
}

func TestOllamaLLM_GenerateStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Streaming response - newline delimited JSON
		w.Write([]byte(`{"response":"Hello","done":false}` + "\n"))
		w.Write([]byte(`{"response":" world","done":false}` + "\n"))
		w.Write([]byte(`{"response":"!","done":true}` + "\n"))
	}))
	defer server.Close()

	adapter := NewOllamaLLMAdapter(server.URL, "test")
	ch, err := adapter.GenerateStream(context.Background(), "test", nil)

	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	var tokens []string
	for token := range ch {
		tokens = append(tokens, token.Content)
		if token.Done {
			break
		}
	}

	if len(tokens) < 2 {
		t.Errorf("expected multiple tokens, got %d", len(tokens))
	}
}

func TestOllamaLLM_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := NewOllamaLLMAdapter(server.URL, "test")
	_, err := adapter.Generate(context.Background(), "test", nil)

	if err == nil {
		t.Error("should error on 404")
	}
}

func TestOllamaLLM_DefaultValues(t *testing.T) {
	adapter := NewOllamaLLMAdapter("", "")
	if adapter.baseURL != "http://localhost:11434" {
		t.Error("should default to localhost")
	}
	if adapter.model != "llama3.2" {
		t.Error("should default to llama3.2")
	}
}
