package entities

import (
	"testing"
	"time"
)

func TestDocument_Creation(t *testing.T) {
	doc := Document{
		ID:        "doc-123",
		Name:      "test.pdf",
		Path:      "/tmp/test.pdf",
		Content:   "Hello world",
		CreatedAt: time.Now(),
	}

	if doc.ID != "doc-123" {
		t.Errorf("expected ID doc-123, got %s", doc.ID)
	}
	if doc.Name != "test.pdf" {
		t.Errorf("expected name test.pdf, got %s", doc.Name)
	}
}

func TestChunk_WithEmbedding(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-1",
		DocumentID: "doc-123",
		Content:    "some text",
		Index:      0,
		Embedding:  []float32{0.1, 0.2, 0.3},
	}

	if len(chunk.Embedding) != 3 {
		t.Errorf("expected 3 embedding dims, got %d", len(chunk.Embedding))
	}
}

func TestQueryResult_Score(t *testing.T) {
	result := QueryResult{
		Chunk:     Chunk{ID: "c1", Content: "test"},
		Score:     0.95,
		SourceDoc: "doc.pdf",
	}

	if result.Score < 0.9 {
		t.Error("expected high score")
	}
}

func TestChatMessage_Roles(t *testing.T) {
	user := ChatMessage{Role: "user", Content: "hello"}
	assistant := ChatMessage{Role: "assistant", Content: "hi there"}

	if user.Role != "user" || assistant.Role != "assistant" {
		t.Error("roles not set correctly")
	}
}

func TestChatRequest_WithHistory(t *testing.T) {
	req := ChatRequest{
		Query: "what is X?",
		History: []ChatMessage{
			{Role: "user", Content: "previous Q"},
			{Role: "assistant", Content: "previous A"},
		},
	}

	if len(req.History) != 2 {
		t.Errorf("expected 2 history items, got %d", len(req.History))
	}
}

func TestChatResponse_WithSources(t *testing.T) {
	resp := ChatResponse{
		Answer: "The answer is 42",
		Sources: []QueryResult{
			{Score: 0.9, SourceDoc: "guide.pdf"},
		},
	}

	if resp.Answer == "" {
		t.Error("answer should not be empty")
	}
	if len(resp.Sources) == 0 {
		t.Error("sources should not be empty")
	}
}
