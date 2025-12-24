// Package loader provides document loading adapters.
package loader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	httpPkg "net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xcro3dile/localrag-go/internal/domain/entities"
)

// TextLoader loads plain text documents (.txt, .md).
type TextLoader struct{}

// NewTextLoader creates a new text document loader.
func NewTextLoader() *TextLoader {
	return &TextLoader{}
}

// Load reads a text document from the given path.
func (l *TextLoader) Load(ctx context.Context, path string) (*entities.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return &entities.Document{
		ID:        generateDocID(path),
		Name:      filepath.Base(path),
		Path:      path,
		Content:   string(content),
		CreatedAt: info.ModTime(),
		UpdatedAt: time.Now(),
	}, nil
}

// SupportedExtensions returns file extensions this loader handles.
func (l *TextLoader) SupportedExtensions() []string {
	return []string{".txt", ".md", ".markdown"}
}

// PDFLoader loads PDF documents via Python service.
type PDFLoader struct {
	serviceURL string
}

// NewPDFLoader creates a PDF loader that calls Python service.
func NewPDFLoader() *PDFLoader {
	return &PDFLoader{serviceURL: "http://localhost:8081"}
}

// NewPDFLoaderWithURL creates a PDF loader with custom service URL.
func NewPDFLoaderWithURL(url string) *PDFLoader {
	return &PDFLoader{serviceURL: url}
}

// Load reads a PDF via Python service.
func (l *PDFLoader) Load(ctx context.Context, path string) (*entities.Document, error) {
	// Read PDF file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Call Python service
	text, err := l.parsePDF(ctx, data)
	if err != nil {
		// Fallback: return empty doc with error note
		text = "[PDF parsing failed: " + err.Error() + "]"
	}

	info, _ := os.Stat(path)
	modTime := time.Now()
	if info != nil {
		modTime = info.ModTime()
	}

	return &entities.Document{
		ID:        generateDocID(path),
		Name:      filepath.Base(path),
		Path:      path,
		Content:   text,
		CreatedAt: modTime,
		UpdatedAt: time.Now(),
	}, nil
}

// parsePDF calls Python service for extraction.
func (l *PDFLoader) parsePDF(ctx context.Context, data []byte) (string, error) {
	req, err := httpPkg.NewRequestWithContext(ctx, "POST", l.serviceURL+"/parse", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &httpPkg.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Text  string `json:"text"`
		Error string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("pdf parse: %s", result.Error)
	}
	return result.Text, nil
}

// SupportedExtensions returns file extensions.
func (l *PDFLoader) SupportedExtensions() []string {
	return []string{".pdf"}
}

// MultiLoader combines multiple loaders.
type MultiLoader struct {
	loaders map[string]interface{ Load(context.Context, string) (*entities.Document, error) }
}

// NewMultiLoader creates a loader that handles multiple file types.
func NewMultiLoader() *MultiLoader {
	return &MultiLoader{
		loaders: map[string]interface{ Load(context.Context, string) (*entities.Document, error) }{
			".txt":      NewTextLoader(),
			".md":       NewTextLoader(),
			".markdown": NewTextLoader(),
			".pdf":      NewPDFLoader(),
		},
	}
}

// Load dispatches to the appropriate loader based on extension.
func (m *MultiLoader) Load(ctx context.Context, path string) (*entities.Document, error) {
	ext := strings.ToLower(filepath.Ext(path))
	loader, ok := m.loaders[ext]
	if !ok {
		// Default to text loader
		loader = NewTextLoader()
	}
	return loader.Load(ctx, path)
}

// SupportedExtensions returns all supported extensions.
func (m *MultiLoader) SupportedExtensions() []string {
	exts := make([]string, 0, len(m.loaders))
	for ext := range m.loaders {
		exts = append(exts, ext)
	}
	return exts
}

// generateDocID creates a deterministic ID for a document.
func generateDocID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:8])
}

// cleanPDFContent removes binary garbage from text.
func cleanPDFContent(content string) string {
	var cleaned strings.Builder
	for _, r := range content {
		if r >= 32 && r < 127 || r == '\n' || r == '\t' {
			cleaned.WriteRune(r)
		}
	}
	return strings.TrimSpace(cleaned.String())
}
