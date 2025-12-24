// Package vectordb provides vector store adapters.
// Clean Architecture: Adapter implementing ports.VectorStore.
// For MVP, using an in-memory store. LanceDB integration can be swapped in.
package vectordb

import (
	"context"
	"sort"
	"sync"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
)

// InMemoryStore is a simple in-memory vector store for MVP.
// Open-Closed: Can be replaced with LanceDB adapter without changing usecases.
type InMemoryStore struct {
	mu     sync.RWMutex
	chunks map[string]entities.Chunk // chunkID -> chunk
	docs   map[string][]string       // docID -> []chunkID
}

// NewInMemoryStore creates a new in-memory vector store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		chunks: make(map[string]entities.Chunk),
		docs:   make(map[string][]string),
	}
}

// Store saves chunks with their embeddings.
func (s *InMemoryStore) Store(ctx context.Context, chunks []entities.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chunk := range chunks {
		s.chunks[chunk.ID] = chunk
		s.docs[chunk.DocumentID] = append(s.docs[chunk.DocumentID], chunk.ID)
	}
	return nil
}

// Search finds the most similar chunks to a query embedding.
func (s *InMemoryStore) Search(ctx context.Context, embedding []float32, topK int) ([]entities.QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		chunk entities.Chunk
		score float64
	}

	var results []scored
	for _, chunk := range s.chunks {
		score := cosineSimilarity(embedding, chunk.Embedding)
		results = append(results, scored{chunk: chunk, score: score})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Take top K
	if len(results) > topK {
		results = results[:topK]
	}

	// Convert to QueryResult
	queryResults := make([]entities.QueryResult, len(results))
	for i, r := range results {
		queryResults[i] = entities.QueryResult{
			Chunk:     r.chunk,
			Score:     r.score,
			SourceDoc: r.chunk.DocumentID, // Could be enhanced with actual doc name
		}
	}

	return queryResults, nil
}

// Delete removes all chunks for a document.
func (s *InMemoryStore) Delete(ctx context.Context, documentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chunkIDs, ok := s.docs[documentID]
	if !ok {
		return nil
	}

	for _, id := range chunkIDs {
		delete(s.chunks, id)
	}
	delete(s.docs, documentID)
	return nil
}

// Clear removes all data from the store.
func (s *InMemoryStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chunks = make(map[string]entities.Chunk)
	s.docs = make(map[string][]string)
	return nil
}

