// Package usecases - query.go handles document search and response generation.
package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
)

// QueryUseCase handles search and response generation.
// Single Responsibility: Only query/response logic.
type QueryUseCase struct {
	embedder    ports.EmbeddingService
	vectorStore ports.VectorStore
	llm         ports.LLMService
	topK        int
}

// NewQueryUseCase creates a QueryUseCase with injected dependencies.
func NewQueryUseCase(
	embedder ports.EmbeddingService,
	vectorStore ports.VectorStore,
	llm ports.LLMService,
	topK int,
) *QueryUseCase {
	if topK <= 0 {
		topK = 5
	}
	return &QueryUseCase{
		embedder:    embedder,
		vectorStore: vectorStore,
		llm:         llm,
		topK:        topK,
	}
}

// Query searches for relevant context and generates a response.
func (uc *QueryUseCase) Query(ctx context.Context, req *entities.ChatRequest) (*entities.ChatResponse, error) {
	// 1. Embed the query
	queryEmbedding, err := uc.embedder.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}

	// 2. Search vector store
	results, err := uc.vectorStore.Search(ctx, queryEmbedding, uc.topK)
	if err != nil {
		return nil, fmt.Errorf("searching vectors: %w", err)
	}

	// 3. Build context from results
	contextParts := make([]string, len(results))
	for i, r := range results {
		contextParts[i] = fmt.Sprintf("[Source: %s]\n%s", r.SourceDoc, r.Chunk.Content)
	}

	// 4. Generate response via LLM
	prompt := uc.buildPrompt(req.Query, contextParts)
	answer, err := uc.llm.Generate(ctx, prompt, contextParts)
	if err != nil {
		return nil, fmt.Errorf("generating response: %w", err)
	}

	return &entities.ChatResponse{
		Answer:  answer,
		Sources: results,
	}, nil
}

// Search only retrieves relevant chunks without LLM generation.
func (uc *QueryUseCase) Search(ctx context.Context, query string) ([]entities.QueryResult, error) {
	embedding, err := uc.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	return uc.vectorStore.Search(ctx, embedding, uc.topK)
}

// buildPrompt creates the LLM prompt with context.
func (uc *QueryUseCase) buildPrompt(query string, context []string) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant. Answer the question based on the provided context.\n\n")
	sb.WriteString("Context:\n")
	sb.WriteString(strings.Join(context, "\n\n"))
	sb.WriteString("\n\nQuestion: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer:")
	return sb.String()
}
