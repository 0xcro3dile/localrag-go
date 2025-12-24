package usecases

import (
	"context"
	"testing"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

// mockLLM implements ports.LLMService for testing
type mockLLM struct {
	response string
}

func (m *mockLLM) Generate(ctx context.Context, prompt string, context []string) (string, error) {
	if m.response != "" {
		return m.response, nil
	}
	return "mocked answer", nil
}

func (m *mockLLM) GenerateStream(ctx context.Context, prompt string, context []string) (<-chan ports.StreamToken, error) {
	ch := make(chan ports.StreamToken, 1)
	go func() {
		ch <- ports.StreamToken{Content: m.response, Done: true}
		close(ch)
	}()
	return ch, nil
}

func TestQueryUseCase_ReturnsAnswer(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		chunks: []entities.Chunk{
			{ID: "c1", Content: "relevant context", DocumentID: "doc1"},
		},
	}
	llm := &mockLLM{response: "The answer is here"}
	uc := NewQueryUseCase(embedder, store, llm, 3)

	req := &entities.ChatRequest{Query: "what is this?"}
	resp, err := uc.Query(context.Background(), req)

	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if resp.Answer != "The answer is here" {
		t.Errorf("unexpected answer: %s", resp.Answer)
	}
}

func TestQueryUseCase_IncludesSources(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		chunks: []entities.Chunk{
			{ID: "c1", Content: "source 1", DocumentID: "doc1"},
			{ID: "c2", Content: "source 2", DocumentID: "doc2"},
		},
	}
	llm := &mockLLM{}
	uc := NewQueryUseCase(embedder, store, llm, 5)

	req := &entities.ChatRequest{Query: "find info"}
	resp, err := uc.Query(context.Background(), req)

	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(resp.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(resp.Sources))
	}
}

func TestQueryUseCase_EmptyStore(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{chunks: nil}
	llm := &mockLLM{response: "no context available"}
	uc := NewQueryUseCase(embedder, store, llm, 5)

	req := &entities.ChatRequest{Query: "hello"}
	resp, err := uc.Query(context.Background(), req)

	if err != nil {
		t.Fatalf("should not fail on empty store: %v", err)
	}
	if len(resp.Sources) != 0 {
		t.Error("should have no sources")
	}
}

func TestQueryUseCase_Search(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		chunks: []entities.Chunk{{ID: "c1", Content: "test"}},
	}
	llm := &mockLLM{}
	uc := NewQueryUseCase(embedder, store, llm, 5)

	results, err := uc.Search(context.Background(), "test query")

	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected search results")
	}
}
