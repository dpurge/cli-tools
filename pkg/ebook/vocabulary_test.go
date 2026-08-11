package ebook

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetVocabularySkipsHeader verifies SPECS §9 / D6: getVocabulary drops
// ATX header lines (# through ######) from inside a {start-vocabulary} block
// so they produce no CSV row. Data rows are returned unchanged.
//
// Edge cases covered:
//   - ## (level 2) and ### (level 3) and ###### (level 6) are all skipped.
//   - ####### (7 hashes) is NOT a valid ATX header and must NOT be skipped.
func TestGetVocabularySkipsHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.md")
	content := "{start-vocabulary}\n" +
		"phrase1 = translation1\n" +
		"## Section Heading\n" +
		"phrase2 = translation2\n" +
		"### Sub-heading\n" +
		"###### Deepest heading\n" +
		"####### NOT a header\n" +
		"phrase3 = translation3\n" +
		"{end-vocabulary}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	lines, err := getVocabulary(path)
	if err != nil {
		t.Fatalf("getVocabulary: %v", err)
	}
	// The three ATX header lines (##, ###, ######) must be absent.
	// The 7-hash line (NOT a header per ATX) and the data lines must be present.
	want := []string{
		"phrase1 = translation1",
		"phrase2 = translation2",
		"####### NOT a header",
		"phrase3 = translation3",
	}
	if len(lines) != len(want) {
		t.Fatalf("getVocabulary returned %d lines %v, want %d %v", len(lines), lines, len(want), want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("getVocabulary[%d] = %q, want %q", i, lines[i], w)
		}
	}
}

// TestGetVocabularyDataOnlyUnchanged verifies ASR-3: a vocabulary block with
// no header lines returns the data lines byte-identical to the pre-change
// output. The header-skip path must be a no-op when no headers are present.
func TestGetVocabularyDataOnlyUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vocab.md")
	content := "{start-vocabulary}\n" +
		"phrase1 = translation1\n" +
		"phrase2 = translation2\n" +
		"{end-vocabulary}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	lines, err := getVocabulary(path)
	if err != nil {
		t.Fatalf("getVocabulary: %v", err)
	}
	want := []string{
		"phrase1 = translation1",
		"phrase2 = translation2",
	}
	if len(lines) != len(want) {
		t.Fatalf("getVocabulary returned %d lines %v, want %d %v", len(lines), lines, len(want), want)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("getVocabulary[%d] = %q, want %q", i, lines[i], w)
		}
	}
}
