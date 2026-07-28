package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Interlinear_Inactive covers SPECS §4.4 / §7: the Interlinear
// block type is defined but deliberately NOT registered with the converter,
// so {start-interlinear}...{end-interlinear} must have no special handling —
// the markers fall through as ordinary paragraph text, exactly like any
// unknown text, with no error and no interlinear-specific wrapper.
func TestToHTML_Interlinear_Inactive(t *testing.T) {
	input := "{start-interlinear}\nword gloss\n{end-interlinear}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "{start-interlinear}") || !strings.Contains(s, "{end-interlinear}") {
		t.Fatalf("interlinear markers should survive as plain text (block is inactive), got: %q", s)
	}
	if strings.Contains(s, "interlinear\"") || strings.Contains(s, "class=\"interlinear") {
		t.Fatalf("interlinear block must not be rendered with any special wrapper, got: %q", s)
	}
}

// TestToHTML_Dialog_TrailingStarStripped covers the intentional dialog
// line-cleanup carried over from the ported code (parser.go: TrimRight(line,
// " *")): trailing spaces/asterisks on a dialog content line are removed
// before the line is treated as content.
func TestToHTML_Dialog_TrailingStarStripped(t *testing.T) {
	input := "{start-dialog}\n@Bob:\n  Hello there *\n{end-dialog}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "Hello there") {
		t.Fatalf("expected dialog content text, got: %q", s)
	}
	if strings.Contains(s, "*") {
		t.Fatalf("trailing ' *' should have been stripped from the dialog line, got: %q", s)
	}
}
