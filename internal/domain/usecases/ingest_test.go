package usecases

import (
	"context"
	"testing"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
)

// mockEmbedder implements ports.EmbeddingService for testing
type mockEmbedder struct {
	embedFn func(text string) ([]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		emb, err := m.Embed(ctx, texts[i])
		if err != nil {
			return nil, err
		}
		result[i] = emb
	}
	return result, nil
}

// mockVectorStore implements ports.VectorStore for testing
type mockVectorStore struct {
	chunks  []entities.Chunk
	storeFn func(chunks []entities.Chunk) error
}

func (m *mockVectorStore) Store(ctx context.Context, chunks []entities.Chunk) error {
	if m.storeFn != nil {
		return m.storeFn(chunks)
	}
	m.chunks = append(m.chunks, chunks...)
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, emb []float32, topK int) ([]entities.QueryResult, error) {
	var results []entities.QueryResult
	for i, c := range m.chunks {
		if i >= topK {
			break
		}
		results = append(results, entities.QueryResult{Chunk: c, Score: 0.9})
	}
	return results, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, docID string) error {
	return nil
}

func (m *mockVectorStore) Clear(ctx context.Context) error {
	m.chunks = nil
	return nil
}

func TestIngestUseCase_ChunksDocument(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}
	uc := NewIngestUseCase(embedder, store, 100, 20)

	doc := &entities.Document{
		ID:      "doc-1",
		Name:    "test.txt",
		Content: "This is some content that should be chunked properly.",
	}

	err := uc.Ingest(context.Background(), doc)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	if len(store.chunks) == 0 {
		t.Error("expected chunks to be stored")
	}
}

func TestIngestUseCase_EmptyDocument(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}
	uc := NewIngestUseCase(embedder, store, 100, 20)

	doc := &entities.Document{ID: "empty", Content: ""}
	err := uc.Ingest(context.Background(), doc)

	if err != nil {
		t.Error("empty doc should not error")
	}
	if len(store.chunks) != 0 {
		t.Error("empty doc should produce no chunks")
	}
}

func TestIngestUseCase_LargeDocument(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}
	uc := NewIngestUseCase(embedder, store, 50, 10)

	// 200 char doc with 50 char chunks = ~4 chunks
	doc := &entities.Document{
		ID:      "big",
		Content: "word word word word word word word word word word word word word word word word word word word word",
	}

	err := uc.Ingest(context.Background(), doc)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	if len(store.chunks) < 2 {
		t.Errorf("expected multiple chunks, got %d", len(store.chunks))
	}
}

func TestIngestUseCase_Delete(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{}
	uc := NewIngestUseCase(embedder, store, 100, 20)

	err := uc.Delete(context.Background(), "doc-1")
	if err != nil {
		t.Errorf("delete failed: %v", err)
	}
}
