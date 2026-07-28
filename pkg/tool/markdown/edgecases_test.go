package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_UnterminatedVocabulary_FallsThroughToParagraph covers SPECS §5
// "Unterminated-block fallback (parity-critical)": a {start-vocabulary}
// block with no matching {end-vocabulary} must NOT swallow the rest of the
// document into a garbage wrapper div, and must NOT panic. Instead the
// {start-vocabulary} line (and everything after it, since there is no
// terminator to bound the block) falls through to plain paragraph text
// containing the literal marker.
func TestToHTML_UnterminatedVocabulary_FallsThroughToParagraph(t *testing.T) {
	input := "{start-vocabulary}\n你好\nno end marker here\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	want := "<p>{start-vocabulary}\n你好\nno end marker here</p>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	if !strings.Contains(string(got), "{start-vocabulary}") {
		t.Fatalf("expected literal marker text to survive as plain paragraph text, got: %q", string(got))
	}
}

// TestToHTML_BadDialogIndentation_ReturnsError covers SPECS §5 D3: a content
// line inside {start-dialog}...{end-dialog} that is neither empty, nor a
// header, nor 2-space-indented must surface as a real, non-nil error from
// ToHTML (not a panic, and not the original ported code's log.Fatal).
func TestToHTML_BadDialogIndentation_ReturnsError(t *testing.T) {
	input := "{start-dialog}\n--:\nbadline\n{end-dialog}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected a non-nil error for bad dialog indentation, got nil (output: %q)", string(got))
	}
	if got != nil {
		t.Fatalf("ToHTML() expected nil output alongside the error, got: %q", string(got))
	}
}

// TestVocabulary_EmptyPhraseAfterEqualsSplit_DocumentsPreservedPanic
// documents (does NOT fix) a pre-existing bug carried over from the ported
// gomarkdown code (see parser.go parseVocabularyItems): a vocabulary line
// that reduces to the empty string after the "= translation" split (e.g.
// "= foo") causes an out-of-range slice index and panics. This is a
// deliberately preserved divergence per the approved migration spec, not a
// regression introduced by this test suite. Verified empirically: it
// panics with "runtime error: slice bounds out of range [-1:]".
func TestVocabulary_EmptyPhraseAfterEqualsSplit_DocumentsPreservedPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic (pre-existing, preserved behavior) but ToHTML did not panic")
		}
		t.Logf("documented preserved panic: %v", r)
	}()

	_, _ = markdown.ToHTML([]byte("{start-vocabulary}\n= foo\n{end-vocabulary}\n"))
}
