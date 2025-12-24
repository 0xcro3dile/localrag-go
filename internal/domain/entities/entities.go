// Package entities contains core business entities.
// These are the enterprise business rules - pure domain objects with no external dependencies.
package entities

import "time"

// Document represents a source document (PDF, TXT, MD).
// This is a core entity - no knowledge of storage or external systems.
type Document struct {
	ID        string
	Name      string
	Path      string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Chunk represents a piece of a document for embedding.
// Clean Architecture: Entity knows nothing about how it's stored or embedded.
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	Index      int      // Position in document
	Embedding  []float32 // Vector representation (populated by adapter)
}

// QueryResult represents a search result with relevance.
type QueryResult struct {
	Chunk      Chunk
	Score      float64 // Similarity score
	SourceDoc  string  // Document name for citation
}

// ChatMessage represents a conversation turn.
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ChatRequest represents a query with conversation context.
type ChatRequest struct {
	Query   string
	History []ChatMessage
}

// ChatResponse represents the LLM's answer with sources.
type ChatResponse struct {
	Answer  string
	Sources []QueryResult
}
