// Package vectordb provides vector store adapters.
// Clean Architecture: Adapter implementing ports.VectorStore.
// LanceDB provides persistent storage with native Go bindings (via CGO/SQLite).
package vectordb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// LanceDBStore implements ports.VectorStore with SQLite-based persistence.
// This is a simplified LanceDB-like implementation using SQLite for portability.
// For production, swap with actual LanceDB Go bindings when available.
type LanceDBStore struct {
	mu       sync.RWMutex
	db       *sql.DB
	dataPath string
}

// NewLanceDBStore creates a new persistent vector store.
func NewLanceDBStore(dataPath string) (*LanceDBStore, error) {
	if dataPath == "" {
		dataPath = "./data"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataPath, "vectors.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &LanceDBStore{
		db:       db,
		dataPath: dataPath,
	}

	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables.
func (s *LanceDBStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS chunks (
		id TEXT PRIMARY KEY,
		document_id TEXT NOT NULL,
		content TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		embedding BLOB NOT NULL,
		source_doc TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_document_id ON chunks(document_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Store saves chunks with their embeddings.
func (s *LanceDBStore) Store(ctx context.Context, chunks []entities.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO chunks (id, document_id, content, chunk_index, embedding, source_doc)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		embeddingJSON, err := json.Marshal(chunk.Embedding)
		if err != nil {
			return fmt.Errorf("encoding embedding: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			chunk.ID,
			chunk.DocumentID,
			chunk.Content,
			chunk.Index,
			embeddingJSON,
			chunk.DocumentID, // source_doc
		)
		if err != nil {
			return fmt.Errorf("inserting chunk: %w", err)
		}
	}

	return tx.Commit()
}

// Search finds the most similar chunks to a query embedding.
func (s *LanceDBStore) Search(ctx context.Context, embedding []float32, topK int) ([]entities.QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Load all chunks and compute similarity (brute force for MVP)
	// For production, use FAISS or actual LanceDB with ANN indexing
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, content, chunk_index, embedding, source_doc
		FROM chunks
	`)
	if err != nil {
		return nil, fmt.Errorf("querying chunks: %w", err)
	}
	defer rows.Close()

	type scored struct {
		chunk entities.Chunk
		score float64
		doc   string
	}

	var results []scored
	for rows.Next() {
		var chunk entities.Chunk
		var embeddingJSON []byte
		var sourceDoc string

		err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.Index, &embeddingJSON, &sourceDoc)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		if err := json.Unmarshal(embeddingJSON, &chunk.Embedding); err != nil {
			continue // Skip corrupted embeddings
		}

		score := cosineSimilarity(embedding, chunk.Embedding)
		results = append(results, scored{chunk: chunk, score: score, doc: sourceDoc})
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
			SourceDoc: r.doc,
		}
	}

	return queryResults, nil
}

// Delete removes all chunks for a document.
func (s *LanceDBStore) Delete(ctx context.Context, documentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM chunks WHERE document_id = ?", documentID)
	return err
}

// Clear removes all data from the store.
func (s *LanceDBStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM chunks")
	return err
}

// Close closes the database connection.
func (s *LanceDBStore) Close() error {
	return s.db.Close()
}

// ChunkCount returns the number of stored chunks.
func (s *LanceDBStore) ChunkCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&count)
	return count, err
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
