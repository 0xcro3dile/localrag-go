package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaAdapter_Embed(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": []float32{0.1, 0.2, 0.3},
		})
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	emb, err := adapter.Embed(context.Background(), "hello")

	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}
	if len(emb) != 3 {
		t.Errorf("expected 3 dims, got %d", len(emb))
	}
}

func TestOllamaAdapter_EmbedBatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": []float32{float32(callCount) * 0.1},
		})
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test-model")
	texts := []string{"a", "b", "c"}
	results, err := adapter.EmbedBatch(context.Background(), texts)

	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestOllamaAdapter_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := NewOllamaAdapter(server.URL, "test")
	_, err := adapter.Embed(context.Background(), "test")

	if err == nil {
		t.Error("should error on 500")
	}
}

func TestOllamaAdapter_DefaultValues(t *testing.T) {
	adapter := NewOllamaAdapter("", "")
	if adapter.baseURL != "http://localhost:11434" {
		t.Error("should default to localhost")
	}
	if adapter.model != "nomic-embed-text" {
		t.Error("should default to nomic-embed-text")
	}
}
