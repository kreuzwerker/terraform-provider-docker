package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenImageImportSourceLocalFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "import.tar")
	content := []byte("hello import")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %s", err)
	}

	reader, err := openImageImportSource(context.Background(), filePath)
	if err != nil {
		t.Fatalf("openImageImportSource returned error: %s", err)
	}
	defer reader.Close() // nolint:errcheck

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read local import source: %s", err)
	}

	if string(data) != string(content) {
		t.Fatalf("expected %q, got %q", string(content), string(data))
	}
}

func TestOpenImageImportSourceHTTPURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello from http"))
	}))
	defer server.Close()

	reader, err := openImageImportSource(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("openImageImportSource returned error: %s", err)
	}
	defer reader.Close() // nolint:errcheck

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read HTTP import source: %s", err)
	}

	if string(data) != "hello from http" {
		t.Fatalf("expected HTTP body %q, got %q", "hello from http", string(data))
	}
}
