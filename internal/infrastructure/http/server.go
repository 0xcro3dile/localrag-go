// Package http provides the HTTP server infrastructure.
// Clean Architecture: Framework/driver layer - outermost circle.
package http

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
	"github.com/0xcro3dile/localrag-go/internal/domain/ports"
	"github.com/0xcro3dile/localrag-go/internal/domain/usecases"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server is the HTTP server for the RAG API and UI.
type Server struct {
	queryUseCase  *usecases.QueryUseCase
	ingestUseCase *usecases.IngestUseCase
	llm           ports.LLMService
	embedder      ports.EmbeddingService
	vectorStore   ports.VectorStore
	templates     *template.Template
	addr          string
}

// NewServer creates a new HTTP server.
func NewServer(
	queryUC *usecases.QueryUseCase,
	ingestUC *usecases.IngestUseCase,
	llm ports.LLMService,
	embedder ports.EmbeddingService,
	vectorStore ports.VectorStore,
	addr string,
) (*Server, error) {
	// Parse embedded templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		tmpl = template.New("index")
	}

	return &Server{
		queryUseCase:  queryUC,
		ingestUseCase: ingestUC,
		llm:           llm,
		embedder:      embedder,
		vectorStore:   vectorStore,
		templates:     tmpl,
		addr:          addr,
	}, nil
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static files
	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// UI
	mux.HandleFunc("/", s.handleIndex)

	// API
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/query/stream", s.handleQueryStream) // SSE streaming
	mux.HandleFunc("/api/health", s.handleHealth)

	server := &http.Server{
		Addr:         s.addr,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 300 * time.Second, // Longer for streaming
	}

	log.Printf("[INFO] LocalRAG server starting on %s", s.addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}

// handleIndex renders the main chat UI with SSE support.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LocalRAG</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://unpkg.com/htmx-ext-sse@2.0.0/sse.js"></script>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>LocalRAG</h1>
            <p class="subtitle">100% private · Zero cloud · Your docs, your data</p>
        </header>
        
        <main>
            <div id="chat-container">
                <div id="messages"></div>
            </div>
            
            <form id="query-form" onsubmit="sendQuery(event)">
                <input type="text" id="query-input" name="query" placeholder="Ask about your documents..." autocomplete="off" required>
                <button type="submit" id="send-btn">Send</button>
            </form>
        </main>
        
        <footer>
            <p>Drop PDFs in <code>./documents</code> folder to ingest</p>
        </footer>
    </div>
    
    <script>
        function sendQuery(e) {
            e.preventDefault();
            const input = document.getElementById('query-input');
            const messages = document.getElementById('messages');
            const query = input.value.trim();
            if (!query) return;
            
            // Add user message
            messages.innerHTML += '<div class="message user">' + escapeHtml(query) + '</div>';
            
            // Add streaming response container
            const responseId = 'response-' + Date.now();
            messages.innerHTML += '<div class="message assistant" id="' + responseId + '"><span class="cursor">▊</span></div>';
            
            // Clear input
            input.value = '';
            
            // Scroll to bottom
            const container = document.getElementById('chat-container');
            container.scrollTop = container.scrollHeight;
            
            // Start SSE streaming
            const eventSource = new EventSource('/api/query/stream?q=' + encodeURIComponent(query));
            const responseEl = document.getElementById(responseId);
            let fullResponse = '';
            
            eventSource.onmessage = function(event) {
                const data = JSON.parse(event.data);
                if (data.done) {
                    eventSource.close();
                    responseEl.innerHTML = fullResponse || 'No response';
                } else if (data.content) {
                    fullResponse += data.content;
                    responseEl.innerHTML = fullResponse + '<span class="cursor">▊</span>';
                    container.scrollTop = container.scrollHeight;
                }
            };
            
            eventSource.onerror = function(err) {
                eventSource.close();
                if (!fullResponse) {
                    responseEl.innerHTML = '<span class="error">Connection error</span>';
                } else {
                    responseEl.innerHTML = fullResponse;
                }
            };
        }
        
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// handleQueryStream handles SSE streaming queries.
func (s *Server) handleQueryStream(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Get relevant context from vector store
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		sendSSE(w, flusher, map[string]interface{}{"error": err.Error(), "done": true})
		return
	}

	results, err := s.vectorStore.Search(ctx, embedding, 5)
	if err != nil {
		sendSSE(w, flusher, map[string]interface{}{"error": err.Error(), "done": true})
		return
	}

	// Build prompt
	var contextParts []string
	for _, r := range results {
		contextParts = append(contextParts, fmt.Sprintf("[Source: %s]\n%s", r.SourceDoc, r.Chunk.Content))
	}

	prompt := buildPrompt(query, contextParts)

	// Stream response
	tokenCh, err := s.llm.GenerateStream(ctx, prompt, contextParts)
	if err != nil {
		sendSSE(w, flusher, map[string]interface{}{"error": err.Error(), "done": true})
		return
	}

	for token := range tokenCh {
		if token.Error != nil {
			sendSSE(w, flusher, map[string]interface{}{"error": token.Error.Error(), "done": true})
			return
		}
		sendSSE(w, flusher, map[string]interface{}{"content": token.Content, "done": token.Done})
	}
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

func buildPrompt(query string, context []string) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant. Answer based on the context.\n\nContext:\n")
	sb.WriteString(strings.Join(context, "\n\n"))
	sb.WriteString("\n\nQuestion: ")
	sb.WriteString(query)
	sb.WriteString("\n\nAnswer:")
	return sb.String()
}

// handleQuery processes a non-streaming query.
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var query string
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var req struct{ Query string `json:"query"` }
		json.NewDecoder(r.Body).Decode(&req)
		query = req.Query
	} else {
		r.ParseForm()
		query = r.FormValue("query")
	}

	if query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	chatReq := &entities.ChatRequest{Query: query}
	resp, err := s.queryUseCase.Query(r.Context(), chatReq)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="message error">Error: ` + err.Error() + `</div>`))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="message user">` + query + `</div><div class="message assistant">` + resp.Answer + `</div>`))
}

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

