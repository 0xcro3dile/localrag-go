// Package ports defines interfaces for external dependencies.
// Clean Architecture: These are the boundaries - usecases depend on these abstractions,
// not concrete implementations. Adapters implement these interfaces.
// This follows Dependency Inversion Principle (DIP) strictly.
package ports

import (
	"context"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
)

// EmbeddingService generates vector embeddings for text.
// Interface Segregation: Only embedding responsibility, nothing else.
type EmbeddingService interface {
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts efficiently.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// LLMService generates text responses from a language model.
// Single Responsibility: Only LLM inference, no embedding logic.
type LLMService interface {
	// Generate produces a response given a prompt and context.
	Generate(ctx context.Context, prompt string, context []string) (string, error)

	// GenerateStream produces a streaming response (for real-time UI).
	// Returns a channel of StreamTokens for token-by-token output.
	GenerateStream(ctx context.Context, prompt string, context []string) (<-chan StreamToken, error)
}

// VectorStore persists and queries document embeddings.
// Dependency Inversion: Usecases depend on this abstraction, not LanceDB directly.
type VectorStore interface {
	// Store saves chunks with their embeddings.
	Store(ctx context.Context, chunks []entities.Chunk) error

	// Search finds the most similar chunks to a query embedding.
	Search(ctx context.Context, embedding []float32, topK int) ([]entities.QueryResult, error)

	// Delete removes all chunks for a document.
	Delete(ctx context.Context, documentID string) error

	// Clear removes all data from the store.
	Clear(ctx context.Context) error
}

// DocumentLoader reads and parses documents from various formats.
type DocumentLoader interface {
	// Load reads a document from the given path.
	Load(ctx context.Context, path string) (*entities.Document, error)

	// SupportedExtensions returns file extensions this loader handles.
	SupportedExtensions() []string
}

// DocumentParser extracts text from binary document formats (PDF, DOCX, etc).
// Interface Segregation: Separate from DocumentLoader for different responsibilities.
type DocumentParser interface {
	// Parse extracts text content from document bytes.
	Parse(ctx context.Context, data []byte, filename string) (string, error)

	// SupportedFormats returns formats this parser handles (e.g., "pdf", "docx").
	SupportedFormats() []string
}

// StreamToken represents a single token in a streaming LLM response.
type StreamToken struct {
	Content string
	Done    bool
	Error   error
}

// FileWatcher monitors a directory for changes.
type FileWatcher interface {
	// Watch starts monitoring the directory and emits events.
	Watch(ctx context.Context, dir string) (<-chan FileEvent, error)

	// Stop stops the watcher.
	Stop() error
}

// FileEvent represents a file system change.
type FileEvent struct {
	Path      string
	Operation FileOperation
}

// FileOperation is the type of file change.
type FileOperation int

const (
	FileCreated FileOperation = iota
	FileModified
	FileDeleted
)
