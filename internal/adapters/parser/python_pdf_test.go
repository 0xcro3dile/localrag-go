package parser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPythonPDFParser_Parse(t *testing.T) {
	// Mock Python service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/parse" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"text":  "Hello from PDF",
			"pages": 1,
		})
	}))
	defer server.Close()

	parser := NewPythonPDFParser(server.URL)
	text, err := parser.Parse(context.Background(), []byte("fake pdf"), "test.pdf")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if text != "Hello from PDF" {
		t.Errorf("unexpected text: %s", text)
	}
}

func TestPythonPDFParser_ServiceError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "parsing failed",
			"text":  "",
		})
	}))
	defer server.Close()

	parser := NewPythonPDFParser(server.URL)
	_, err := parser.Parse(context.Background(), []byte("bad"), "test.pdf")

	if err == nil {
		t.Error("should error on parse failure")
	}
}

func TestPythonPDFParser_SupportedFormats(t *testing.T) {
	parser := NewPythonPDFParser("")
	formats := parser.SupportedFormats()

	if len(formats) != 1 || formats[0] != "pdf" {
		t.Error("should support only pdf")
	}
}

func TestPythonPDFParser_DefaultURL(t *testing.T) {
	parser := NewPythonPDFParser("")
	if parser.serviceURL != "http://localhost:8081" {
		t.Error("should default to localhost:8081")
	}
}

func TestPythonPDFParser_IsServiceHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()

	parser := NewPythonPDFParser(server.URL)
	healthy := parser.IsServiceHealthy(context.Background())

	if !healthy {
		t.Error("should be healthy")
	}
}

func TestPythonPDFParser_UnhealthyService(t *testing.T) {
	parser := NewPythonPDFParser("http://localhost:99999")
	healthy := parser.IsServiceHealthy(context.Background())

	if healthy {
		t.Error("should be unhealthy")
	}
}
