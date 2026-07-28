package scanbook

import (
	"path/filepath"
	"testing"
)

// TestPdfOutputPath covers the .pdf extension handling: appended when absent,
// preserved (never doubled, any case) when already present.
func TestPdfOutputPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"my-book", "my-book.pdf"},
		{"my-book.pdf", "my-book.pdf"},
		{"MY-BOOK.PDF", "MY-BOOK.PDF"},
		{"book.PdF", "book.PdF"},
		{"a/b/book", "a/b/book.pdf"},
	}
	for _, tc := range cases {
		if got := pdfOutputPath(tc.in); got != tc.want {
			t.Errorf("pdfOutputPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestConvertPagesToPdfMissingInput errors before touching ImageMagick when
// the input directory does not exist.
func TestConvertPagesToPdfMissingInput(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := convertPagesToPdf(missing, "out", "png"); err == nil {
		t.Error("expected an error for a missing input directory, got nil")
	}
}

// TestConvertPagesToPdfNoPages errors (before ImageMagick) when the input
// directory exists but holds no pages of the requested format.
func TestConvertPagesToPdfNoPages(t *testing.T) {
	dir := t.TempDir() // exists, but empty
	if _, err := convertPagesToPdf(dir, "out", "png"); err == nil {
		t.Error("expected an error when no matching pages exist, got nil")
	}
}
