// Package usecases contains application business rules.
// Clean Architecture: Usecases orchestrate entities and depend on port interfaces.
// They contain NO framework code, NO external dependencies - just pure business logic.
package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

// IngestUseCase handles document ingestion into the vector store.
// Single Responsibility: Only ingestion logic.
type IngestUseCase struct {
	embedder    ports.EmbeddingService
	vectorStore ports.VectorStore
	chunkSize   int
	chunkOverlap int
}

// NewIngestUseCase creates an IngestUseCase with injected dependencies.
// Dependency Injection: Adapters are passed in, not created here.
func NewIngestUseCase(
	embedder ports.EmbeddingService,
	vectorStore ports.VectorStore,
	chunkSize, chunkOverlap int,
) *IngestUseCase {
	if chunkSize <= 0 {
		chunkSize = 500 // Default chunk size in characters
	}
	if chunkOverlap < 0 {
		chunkOverlap = 50
	}
	return &IngestUseCase{
		embedder:     embedder,
		vectorStore:  vectorStore,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// Ingest processes a document: chunks it, embeds it, stores it.
func (uc *IngestUseCase) Ingest(ctx context.Context, doc *entities.Document) error {
	// 1. Chunk the document
	chunks := uc.chunkDocument(doc)
	if len(chunks) == 0 {
		return nil // Empty document
	}

	// 2. Extract text for embedding
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	// 3. Generate embeddings via port (adapter)
	embeddings, err := uc.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return err
	}

	// 4. Attach embeddings to chunks
	for i := range chunks {
		chunks[i].Embedding = embeddings[i]
	}

	// 5. Store in vector DB via port
	return uc.vectorStore.Store(ctx, chunks)
}

// Delete removes a document from the store.
func (uc *IngestUseCase) Delete(ctx context.Context, documentID string) error {
	return uc.vectorStore.Delete(ctx, documentID)
}

// chunkDocument splits document content into overlapping chunks.
// Pure business logic - no external dependencies.
func (uc *IngestUseCase) chunkDocument(doc *entities.Document) []entities.Chunk {
	content := strings.TrimSpace(doc.Content)
	if len(content) == 0 {
		return nil
	}

	var chunks []entities.Chunk
	start := 0
	index := 0

	for start < len(content) {
		end := start + uc.chunkSize
		if end > len(content) {
			end = len(content)
		}

		// Try to break at word boundary
		if end < len(content) {
			lastSpace := strings.LastIndex(content[start:end], " ")
			if lastSpace > 0 {
				end = start + lastSpace
			}
		}

		chunkContent := strings.TrimSpace(content[start:end])
		if len(chunkContent) > 0 {
			chunks = append(chunks, entities.Chunk{
				ID:         generateChunkID(doc.ID, index),
				DocumentID: doc.ID,
				Content:    chunkContent,
				Index:      index,
			})
			index++
		}

		start = end - uc.chunkOverlap
		if start < 0 {
			start = 0
		}
		if start >= len(content) {
			break
		}
	}

	return chunks
}

// generateChunkID creates a deterministic ID for a chunk.
func generateChunkID(docID string, index int) string {
	hash := sha256.Sum256([]byte(docID + string(rune(index))))
	return hex.EncodeToString(hash[:8])
}
