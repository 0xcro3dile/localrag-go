// Package parser provides document parsing adapters.
// Clean Architecture: Adapter implementing ports.DocumentParser.
// Calls external Python service for PDF extraction.
package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// PythonPDFParser implements ports.DocumentParser using Python subprocess.
// Dependency Inversion: Usecases depend on DocumentParser interface, not this.
type PythonPDFParser struct {
	serviceURL string
	client     *http.Client
	pythonCmd  *exec.Cmd
}

// NewPythonPDFParser creates a new PDF parser that calls Python service.
func NewPythonPDFParser(serviceURL string) *PythonPDFParser {
	if serviceURL == "" {
		serviceURL = "http://localhost:8081"
	}
	return &PythonPDFParser{
		serviceURL: serviceURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// parseResponse is the Python service response format.
type parseResponse struct {
	Text    string `json:"text"`
	Pages   int    `json:"pages"`
	Library string `json:"library,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Parse extracts text from PDF bytes via Python service.
func (p *PythonPDFParser) Parse(ctx context.Context, data []byte, filename string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", p.serviceURL+"/parse", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling PDF service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var result parseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("PDF parse error: %s", result.Error)
	}

	return result.Text, nil
}

// SupportedFormats returns formats this parser handles.
func (p *PythonPDFParser) SupportedFormats() []string {
	return []string{"pdf"}
}

// StartService starts the Python PDF service as a subprocess.
// Returns a cleanup function to stop the service.
func (p *PythonPDFParser) StartService(pythonPath string) (func(), error) {
	// Find the pdf_service.py
	scriptPath := filepath.Join(pythonPath, "pdf_service.py")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("pdf_service.py not found at %s", scriptPath)
	}

	// Start Python process
	p.pythonCmd = exec.Command("python3", scriptPath)
	p.pythonCmd.Stdout = os.Stdout
	p.pythonCmd.Stderr = os.Stderr

	if err := p.pythonCmd.Start(); err != nil {
		return nil, fmt.Errorf("starting Python service: %w", err)
	}

	// Wait for service to be ready
	time.Sleep(1 * time.Second)

	cleanup := func() {
		if p.pythonCmd != nil && p.pythonCmd.Process != nil {
			p.pythonCmd.Process.Kill()
		}
	}

	return cleanup, nil
}

// IsServiceHealthy checks if the Python service is running.
func (p *PythonPDFParser) IsServiceHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", p.serviceURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
