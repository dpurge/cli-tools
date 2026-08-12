package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Models_Golden asserts the EXACT (byte-identical) wrapper and
// span output for the {start-models}...{end-models} block, covering all
// four render variants (agreed migration decision): phrase only (a plain,
// non-tabular line); phrase+transcription; phrase+translation; and
// phrase+transcription+translation (transcription stacked below the
// phrase in col1). Consecutive paired items are wrapped in a single
// "models-group" (mirrors "questions-group") so their columns align
// across items, matching the Typst side (book.typ grids every item
// together); a phrase-only item flushes the current group. Models content
// is raw and unescaped, so these are golden, full-string comparisons
// rather than substring checks.
func TestToHTML_Models_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "phrase only",
			input: "{start-models}\n你好\n{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-item\">\n" +
				"<span class=\"models-phrase\">你好</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// RTL integration golden: script=arab → blockDirection("arab")="rtl"
			// → the models wrapper carries dir="rtl" + s-arab class. The badge
			// itself is always left-side (FR-2: no dir attribute on the badge div).
			name:  "script=arab: RTL badge and wrapper",
			input: "{start-models script=arab}\nمرحبا\n{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models s-arab\" dir=\"rtl\">\n" +
				"<div class=\"models-item\">\n" +
				"<span class=\"models-phrase\">مرحبا</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + transcription, no translation",
			input: "{start-models}\nrun [rʌn]\n{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-transcription\" dir=\"ltr\">rʌn</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + translation, no transcription",
			input: "{start-models}\nrun = biec\n{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + transcription + translation: transcription stacked below phrase in col1",
			input: "{start-models}\nrun [rʌn] = biec\n{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"<br/>\n" +
				"<span class=\"models-transcription\" dir=\"ltr\">rʌn</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "multiple items, mixed variants: a solo phrase followed by a run of two paired items sharing one group",
			input: "{start-models}\n" +
				"你好\n" +
				"run [rʌn]\n" +
				"run = biec\n" +
				"{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-item\">\n" +
				"<span class=\"models-phrase\">你好</span>\n" +
				"</div>\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-transcription\" dir=\"ltr\">rʌn</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "a phrase-only item breaks the group, like a Q-only line breaks questions-group",
			input: "{start-models}\n" +
				"run [rʌn] = biec\n" +
				"再见\n" +
				"walk = iść\n" +
				"{end-models}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
				"<div class=\"models\" dir=\"ltr\">\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">run</span>\n" +
				"<br/>\n" +
				"<span class=\"models-transcription\" dir=\"ltr\">rʌn</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"models-item\">\n" +
				"<span class=\"models-phrase\">再见</span>\n" +
				"</div>\n" +
				"<div class=\"models-group\">\n" +
				"<div class=\"models-item paired\">\n" +
				"<div class=\"models-col1\">\n" +
				"<span class=\"models-phrase\">walk</span>\n" +
				"</div>\n" +
				"<div class=\"models-col2\">\n" +
				"<span class=\"models-translation\" dir=\"ltr\">iść</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToHTML() unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// SPECS §12.1 — Parser round-trip (verified through HTML rendering)
// ---------------------------------------------------------------------

// TestParseModelsHeaderAndNote asserts that a ## header and a (note) line
// inside a models block produce ItemHeader and ItemNote items at the correct
// positions, and that the models-group is flushed before the header (SPECS
// §12.1, §12.2, §6 group-flush).
func TestParseModelsHeaderAndNote(t *testing.T) {
	input := "{start-models}\nrun = biec\n## Section\n(Learn to run)\nwalk = iść\n{end-models}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
		"<div class=\"models\" dir=\"ltr\">\n" +
		// First paired item in its own group.
		"<div class=\"models-group\">\n" +
		"<div class=\"models-item paired\">\n" +
		"<div class=\"models-col1\">\n" +
		"<span class=\"models-phrase\">run</span>\n" +
		"</div>\n" +
		"<div class=\"models-col2\">\n" +
		"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n" +
		// Block header flushes the group and renders at full width.
		"<h2>Section</h2>\n" +
		// Block note renders at full width outside any group.
		"<p class=\"block-note\">Learn to run</p>\n" +
		// Second paired item opens a new group.
		"<div class=\"models-group\">\n" +
		"<div class=\"models-item paired\">\n" +
		"<div class=\"models-col1\">\n" +
		"<span class=\"models-phrase\">walk</span>\n" +
		"</div>\n" +
		"<div class=\"models-col2\">\n" +
		"<span class=\"models-translation\" dir=\"ltr\">iść</span>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// ---------------------------------------------------------------------
// SPECS §12.2 — HTML render
// ---------------------------------------------------------------------

// TestRenderModelsNoteHTML asserts that a models note item flushes any open
// models-group before emitting <p class="block-note">…</p>, and a new group
// begins for paired items that follow (SPECS §12.2, §6 group-flush).
func TestRenderModelsNoteHTML(t *testing.T) {
	input := "{start-models}\nrun = biec\n(practice)\nwalk = iść\n{end-models}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">M</span></div>\n" +
		"<div class=\"models\" dir=\"ltr\">\n" +
		"<div class=\"models-group\">\n" +
		"<div class=\"models-item paired\">\n" +
		"<div class=\"models-col1\">\n" +
		"<span class=\"models-phrase\">run</span>\n" +
		"</div>\n" +
		"<div class=\"models-col2\">\n" +
		"<span class=\"models-translation\" dir=\"ltr\">biec</span>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n" +
		// Note flushes the group and renders outside it.
		"<p class=\"block-note\">practice</p>\n" +
		// New group for the following paired item.
		"<div class=\"models-group\">\n" +
		"<div class=\"models-item paired\">\n" +
		"<div class=\"models-col1\">\n" +
		"<span class=\"models-phrase\">walk</span>\n" +
		"</div>\n" +
		"<div class=\"models-col2\">\n" +
		"<span class=\"models-translation\" dir=\"ltr\">iść</span>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestModels_As_Rejected covers SPECS §5: models' field languages are fixed
// (like vocabulary), so as= is rejected entirely, with a clearer message
// than the pre-unification blanket rejection.
func TestModels_As_Rejected(t *testing.T) {
	input := "{start-models as=source}\nrun = biec\n{end-models}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-models}, got nil")
	}
	wantErr := "as= not applicable to {start-models}: its field languages are fixed"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}
