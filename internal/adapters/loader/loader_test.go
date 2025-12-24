package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTextLoader_LoadTxtFile(t *testing.T) {
	// Create temp file
	dir, _ := os.MkdirTemp("", "loader-test-*")
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("Hello World"), 0644)

	loader := NewTextLoader()
	doc, err := loader.Load(context.Background(), path)

	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if doc.Content != "Hello World" {
		t.Errorf("unexpected content: %s", doc.Content)
	}
	if doc.Name != "test.txt" {
		t.Errorf("unexpected name: %s", doc.Name)
	}
}

func TestTextLoader_SupportedExtensions(t *testing.T) {
	loader := NewTextLoader()
	exts := loader.SupportedExtensions()

	if len(exts) == 0 {
		t.Error("should support extensions")
	}

	found := false
	for _, e := range exts {
		if e == ".txt" {
			found = true
		}
	}
	if !found {
		t.Error(".txt should be supported")
	}
}

func TestMultiLoader_DispatchByExtension(t *testing.T) {
	dir, _ := os.MkdirTemp("", "loader-test-*")
	defer os.RemoveAll(dir)

	// Create test files
	txtPath := filepath.Join(dir, "test.txt")
	mdPath := filepath.Join(dir, "test.md")
	os.WriteFile(txtPath, []byte("txt content"), 0644)
	os.WriteFile(mdPath, []byte("# Markdown"), 0644)

	loader := NewMultiLoader()

	txt, _ := loader.Load(context.Background(), txtPath)
	md, _ := loader.Load(context.Background(), mdPath)

	if txt.Content != "txt content" {
		t.Error("txt not loaded correctly")
	}
	if md.Content != "# Markdown" {
		t.Error("md not loaded correctly")
	}
}

func TestMultiLoader_AllExtensions(t *testing.T) {
	loader := NewMultiLoader()
	exts := loader.SupportedExtensions()

	if len(exts) < 3 {
		t.Errorf("expected at least 3 extensions, got %d", len(exts))
	}
}

func TestLoader_NonexistentFile(t *testing.T) {
	loader := NewTextLoader()
	_, err := loader.Load(context.Background(), "/nonexistent/file.txt")

	if err == nil {
		t.Error("should error on nonexistent file")
	}
}
