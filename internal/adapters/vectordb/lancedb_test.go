package vectordb

import (
	"context"
	"os"
	"testing"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
)

func TestLanceDBStore_StoreAndSearch(t *testing.T) {
	// Create temp dir for test DB
	dir, _ := os.MkdirTemp("", "lancedb-test-*")
	defer os.RemoveAll(dir)

	store, err := NewLanceDBStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	chunks := []entities.Chunk{
		{ID: "c1", DocumentID: "doc1", Content: "hello", Embedding: []float32{1.0, 0.0, 0.0}},
		{ID: "c2", DocumentID: "doc1", Content: "world", Embedding: []float32{0.0, 1.0, 0.0}},
	}

	// Store
	if err := store.Store(ctx, chunks); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	// Search
	query := []float32{1.0, 0.0, 0.0} // Should match c1
	results, err := store.Search(ctx, query, 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if results[0].Chunk.ID != "c1" {
		t.Error("c1 should be top result")
	}
}

func TestLanceDBStore_Delete(t *testing.T) {
	dir, _ := os.MkdirTemp("", "lancedb-test-*")
	defer os.RemoveAll(dir)

	store, _ := NewLanceDBStore(dir)
	defer store.Close()

	ctx := context.Background()
	store.Store(ctx, []entities.Chunk{
		{ID: "c1", DocumentID: "doc1", Content: "test", Embedding: []float32{1, 0, 0}},
	})

	// Delete
	if err := store.Delete(ctx, "doc1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify gone
	results, _ := store.Search(ctx, []float32{1, 0, 0}, 10)
	if len(results) != 0 {
		t.Error("chunks should be deleted")
	}
}

func TestLanceDBStore_Clear(t *testing.T) {
	dir, _ := os.MkdirTemp("", "lancedb-test-*")
	defer os.RemoveAll(dir)

	store, _ := NewLanceDBStore(dir)
	defer store.Close()

	ctx := context.Background()
	store.Store(ctx, []entities.Chunk{
		{ID: "c1", Embedding: []float32{1, 0, 0}},
		{ID: "c2", Embedding: []float32{0, 1, 0}},
	})

	store.Clear(ctx)

	count, _ := store.ChunkCount(ctx)
	if count != 0 {
		t.Errorf("expected 0 chunks after clear, got %d", count)
	}
}

func TestLanceDBStore_CosineSimilarity(t *testing.T) {
	// Test the similarity function
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	c := []float32{0, 1, 0}

	same := cosineSimilarity(a, b)
	diff := cosineSimilarity(a, c)

	if same != 1.0 {
		t.Errorf("same vectors should have score 1.0, got %f", same)
	}
	if diff != 0.0 {
		t.Errorf("orthogonal vectors should have score 0.0, got %f", diff)
	}
}
