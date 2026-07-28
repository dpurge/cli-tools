package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestFileToHTML_ConvertsTempFile writes a markdown file into an isolated
// t.TempDir() (hermetic: no repo files touched, nothing left behind) and
// asserts FileToHTML reads and converts it correctly.
func TestFileToHTML_ConvertsTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")

	content := "# Title\n\nSome **bold** text.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp markdown file: %v", err)
	}

	got, err := markdown.FileToHTML(path)
	if err != nil {
		t.Fatalf("FileToHTML() unexpected error: %v", err)
	}
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Fatalf("FileToHTML() = %q, want it to contain <strong>bold</strong>", got)
	}
	if !strings.Contains(got, "<h1") {
		t.Fatalf("FileToHTML() = %q, want it to contain an <h1> heading", got)
	}
}

// TestFileToHTML_NonExistentPath_ReturnsError asserts FileToHTML surfaces a
// non-nil error (from the underlying os.ReadFile) for a path that does not
// exist, rather than panicking.
func TestFileToHTML_NonExistentPath_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.md")

	got, err := markdown.FileToHTML(path)
	if err == nil {
		t.Fatalf("FileToHTML(%q) expected a non-nil error, got nil (output: %q)", path, got)
	}
	if got != "" {
		t.Fatalf("FileToHTML(%q) expected empty output alongside the error, got: %q", path, got)
	}
}
