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

// TestToHTML_UnterminatedModels_FallsThroughToParagraph mirrors
// TestToHTML_UnterminatedVocabulary_FallsThroughToParagraph for the new
// {start-models} block (AC-2.4): with no matching {end-models}, the
// {start-models} line must fall through to plain paragraph text instead
// of swallowing the rest of the document or panicking.
func TestToHTML_UnterminatedModels_FallsThroughToParagraph(t *testing.T) {
	input := "{start-models}\nrun [rʌn]\nno end marker here\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	want := "<p>{start-models}\nrun [rʌn]\nno end marker here</p>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestToHTML_UnterminatedQuestions_FallsThroughToParagraph mirrors
// TestToHTML_UnterminatedVocabulary_FallsThroughToParagraph for the new
// {start-questions} block (AC-2.4).
func TestToHTML_UnterminatedQuestions_FallsThroughToParagraph(t *testing.T) {
	input := "{start-questions}\nQ1 = A1\nno end marker here\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	want := "<p>{start-questions}\nQ1 = A1\nno end marker here</p>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestModels_EmptyPhraseAfterBracketSplit_NoPanic covers the agreed
// migration decision for FR-2A.5 (models diverges deliberately from
// vocabulary's preserved empty-phrase panic): a models line that reduces
// to the empty string after the trailing "[...]" split (e.g. a
// transcription-only line "[abc]", with no phrase text before the
// bracket) MUST NOT panic, and the phrase span is simply omitted from the
// rendered output.
func TestModels_EmptyPhraseAfterBracketSplit_NoPanic(t *testing.T) {
	got, err := markdown.ToHTML([]byte("{start-models}\n[abc]\n{end-models}\n"))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	want := "<div class=\"models\">\n" +
		"<div class=\"models-group\">\n" +
		"<div class=\"models-item paired\">\n" +
		"<div class=\"models-col1\">\n" +
		"</div>\n" +
		"<div class=\"models-col2\">\n" +
		"<span class=\"models-transcription\">abc</span>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n"
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestToTypstAndToMDX_ModelsAndQuestions_NoPanicSmoke is the Gate P
// panic-safety smoke test (CRITICAL per the migration plan): KindModels
// and KindQuestions hold the HIGHEST node-kind ordinals in this package,
// so if either kind were registered on the HTML path but missing from
// typstNodeRenderer or mdxNodeRenderer's RegisterFuncs, a document
// containing that block would panic (index out of range) under
// ToTypst/ToMDX instead of failing softly. This asserts both conversions
// succeed with non-empty output for a document containing BOTH new
// blocks together.
func TestToTypstAndToMDX_ModelsAndQuestions_NoPanicSmoke(t *testing.T) {
	input := "{start-models}\n" +
		"run [rʌn] = biec\n" +
		"{end-models}\n" +
		"{start-questions}\n" +
		"Q1 = A1\n" +
		"{end-questions}\n"

	typstOut, err := markdown.ToTypst([]byte(input))
	if err != nil {
		t.Fatalf("ToTypst() unexpected error: %v", err)
	}
	if len(typstOut) == 0 {
		t.Fatalf("ToTypst() expected non-empty output")
	}

	mdxOut, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if len(mdxOut) == 0 {
		t.Fatalf("ToMDX() expected non-empty output")
	}

	htmlOut, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if len(htmlOut) == 0 {
		t.Fatalf("ToHTML() expected non-empty output")
	}
}
