package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestCRLFDialog guards the CRLF fix: a dialog in a CRLF-encoded source must
// parse identically to its LF form. Before the fix the trailing "\r" left on
// each split line made the "--:" header fail to match, erroring the build.
// ToMDX is included to cover bug #1 (mdx.go missed normalizeNewlines before
// the fix, unlike ToHTML/ToTypst which already called it).
func TestCRLFDialog(t *testing.T) {
	lf := "{start-dialog}\n--:\n  Hello there.\n{end-dialog}\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	convs := []struct {
		name string
		fn   func([]byte) ([]byte, error)
	}{
		{"ToHTML", markdown.ToHTML},
		{"ToTypst", markdown.ToTypst},
		{"ToMDX", func(b []byte) ([]byte, error) { return markdown.ToMDX(b, "eng", "latn") }},
	}
	for _, c := range convs {
		lfOut, err := c.fn([]byte(lf))
		if err != nil {
			t.Fatalf("%s LF error: %v", c.name, err)
		}
		crlfOut, err := c.fn([]byte(crlf))
		if err != nil {
			t.Fatalf("%s CRLF error: %v", c.name, err)
		}
		if string(lfOut) != string(crlfOut) {
			t.Errorf("%s CRLF output differs from LF:\n LF=%q\n CRLF=%q", c.name, lfOut, crlfOut)
		}
	}
}

// TestCRLFBlocks guards the CRLF fix for the item-based blocks models and
// questions (review finding F9). The central normalizeNewlines already covers
// them; this pins that coverage so a future move of normalization downstream
// cannot silently reintroduce the trailing-"\r" parse hazard for these blocks.
func TestCRLFBlocks(t *testing.T) {
	blocks := []struct {
		name string
		lf   string
	}{
		{"models", "{start-models}\nrun [rʌn] = biec\nwalk = iść\n{end-models}\n"},
		{"questions", "{start-questions}\nWhat is your name?\nWhere are you from? = Poland\nHow old are you?\n{end-questions}\n"},
	}
	convs := []struct {
		name string
		fn   func([]byte) ([]byte, error)
	}{
		{"ToHTML", markdown.ToHTML},
		{"ToTypst", markdown.ToTypst},
		{"ToMDX", func(b []byte) ([]byte, error) { return markdown.ToMDX(b, "eng", "latn") }},
	}
	for _, blk := range blocks {
		crlf := strings.ReplaceAll(blk.lf, "\n", "\r\n")
		for _, c := range convs {
			lfOut, err := c.fn([]byte(blk.lf))
			if err != nil {
				t.Fatalf("%s %s LF error: %v", blk.name, c.name, err)
			}
			crlfOut, err := c.fn([]byte(crlf))
			if err != nil {
				t.Fatalf("%s %s CRLF error: %v", blk.name, c.name, err)
			}
			if string(lfOut) != string(crlfOut) {
				t.Errorf("%s %s CRLF output differs from LF:\n LF=%q\n CRLF=%q", blk.name, c.name, lfOut, crlfOut)
			}
		}
	}
}

// TestCRLFParallel guards the parallel-block CRLF fix (amendment F2): a
// parallel block using the "===" row separator must parse identically
// whether the source is LF or CRLF encoded.
func TestCRLFParallel(t *testing.T) {
	lf := "{start-parallel}\nMain cell.\n---\nSecondary cell.\n===\nRow two main only.\n{end-parallel}\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	convs := []struct {
		name string
		fn   func([]byte) ([]byte, error)
	}{
		{"ToHTML", markdown.ToHTML},
		{"ToTypst", markdown.ToTypst},
		{"ToMDX", func(b []byte) ([]byte, error) { return markdown.ToMDX(b, "eng", "latn") }},
	}
	for _, c := range convs {
		lfOut, err := c.fn([]byte(lf))
		if err != nil {
			t.Fatalf("%s LF error: %v", c.name, err)
		}
		crlfOut, err := c.fn([]byte(crlf))
		if err != nil {
			t.Fatalf("%s CRLF error: %v", c.name, err)
		}
		if string(lfOut) != string(crlfOut) {
			t.Errorf("%s CRLF output differs from LF:\n LF=%q\n CRLF=%q", c.name, lfOut, crlfOut)
		}
	}
}
