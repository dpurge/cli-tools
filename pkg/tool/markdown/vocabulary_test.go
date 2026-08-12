package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Vocabulary_Golden asserts the EXACT (byte-identical) wrapper
// and span output for the {start-vocabulary}...{end-vocabulary} block per
// SPECS §4.1. Vocabulary content is raw and unescaped, so these are golden,
// full-string comparisons rather than substring checks.
func TestToHTML_Vocabulary_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full line: phrase, grammar, transcription, translation",
			input: "{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\" dir=\"ltr\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">hello</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase only",
			input: "{start-vocabulary}\n你好\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + translation",
			input: "{start-vocabulary}\n谢谢 = thank you\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">thank you</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + grammar",
			input: "{start-vocabulary}\n早上好 {greeting}\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">早上好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">greeting</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// RTL integration golden: script=arab → blockDirection("arab")="rtl"
			// → the vocabulary wrapper carries dir="rtl" + s-arab class. The badge
			// itself is always left-side (FR-2: no dir attribute on the badge div).
			name:  "script=arab: RTL badge and wrapper",
			input: "{start-vocabulary lang=ar script=arab}\nمرحبا = hello\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary s-arab\" dir=\"rtl\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">مرحبا</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">hello</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "multiple items",
			input: "{start-vocabulary}\n" +
				"你好 {noun} [nǐ hǎo] = hello\n" +
				"再见\n" +
				"谢谢 = thank you\n" +
				"{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\" dir=\"ltr\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">hello</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">再见</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">thank you</span>\n" +
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

// TestVocabulary_As_Rejected covers SPECS §5: vocabulary's field languages
// are fixed (phrase=foreign, translation=base), so as= is rejected
// entirely — even an in-grammar value like "source" — with a clearer
// message than the pre-unification blanket rejection.
func TestVocabulary_As_Rejected(t *testing.T) {
	input := "{start-vocabulary as=source}\nphrase = translation\n{end-vocabulary}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-vocabulary}, got nil")
	}
	wantErr := "as= not applicable to {start-vocabulary}: its field languages are fixed"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}

// ---------------------------------------------------------------------
// SPECS §12.1 — Parser round-trip (verified through HTML rendering)
// ---------------------------------------------------------------------

// TestParseVocabularyHeaderItem asserts that a ## Heading mid-block produces
// an ItemHeader at the correct positional index, with surrounding data items
// intact (SPECS §12.1, ASR-2).
func TestParseVocabularyHeaderItem(t *testing.T) {
	input := "{start-vocabulary}\nphrase1 = t1\n## Heading\nphrase2 = t2\n{end-vocabulary}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
		"<div class=\"vocabulary\" dir=\"ltr\">\n" +
		"<div class=\"vocabulary-item\">\n" +
		"<span class=\"vocabulary-phrase\">phrase1</span>\n" +
		"<span class=\"vocabulary-translation\" dir=\"ltr\">t1</span>\n" +
		"</div>\n" +
		"<h2>Heading</h2>\n" +
		"<div class=\"vocabulary-item\">\n" +
		"<span class=\"vocabulary-phrase\">phrase2</span>\n" +
		"<span class=\"vocabulary-translation\" dir=\"ltr\">t2</span>\n" +
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

// TestParseVocabularyHeaderLevels asserts that levels 1–6 are recognised as
// block headers, and that 7 '#' characters are NOT a header (fall through to
// phrase item). Covers SPECS §12.1 and §3.1 level range N1.
func TestParseVocabularyHeaderLevels(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "level 1: single '#'",
			input: "{start-vocabulary}\n# H1\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<h1>H1</h1>\n" +
				"</div>\n",
		},
		{
			name:  "level 2: '##'",
			input: "{start-vocabulary}\n## H2\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<h2>H2</h2>\n" +
				"</div>\n",
		},
		{
			name:  "level 6: '######'",
			input: "{start-vocabulary}\n###### H6\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<h6>H6</h6>\n" +
				"</div>\n",
		},
		{
			// 7 '#' must NOT be a header — falls through to phrase item (SPECS §3.1).
			name:  "level 7: 7 '#' is NOT a header, treated as phrase",
			input: "{start-vocabulary}\n####### H7\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">####### H7</span>\n" +
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

// TestParseVocabularyNoNote asserts D1: a standalone (x) line inside a
// vocabulary block is NOT recognised as a note — it is treated as a phrase
// data item, with the parentheses preserved verbatim in the phrase field
// (SPECS §12.1, §3.3).
func TestParseVocabularyNoNote(t *testing.T) {
	input := "{start-vocabulary}\n(hello)\n{end-vocabulary}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
		"<div class=\"vocabulary\" dir=\"ltr\">\n" +
		"<div class=\"vocabulary-item\">\n" +
		"<span class=\"vocabulary-phrase\">(hello)</span>\n" +
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

// TestRenderVocabularyHeaderHTML asserts exact <hN> emission for block
// headers: no `id` attribute (ASR-6), raw text written (ASR-4), and boundary
// levels h1 and h6 each covered explicitly (SPECS §12.2 N1).
func TestRenderVocabularyHeaderHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "h2 in mid-block, no id attribute",
			input: "{start-vocabulary}\nphrase1 = t1\n## Heading\nphrase2 = t2\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">phrase1</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">t1</span>\n" +
				"</div>\n" +
				"<h2>Heading</h2>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">phrase2</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">t2</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// Boundary case: h1 (level 1) — SPECS §12.2 N1.
			name:  "h1 boundary: level 1",
			input: "{start-vocabulary}\n# H1\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<h1>H1</h1>\n" +
				"</div>\n",
		},
		{
			// Boundary case: h6 (level 6) — SPECS §12.2 N1.
			name:  "h6 boundary: level 6",
			input: "{start-vocabulary}\n###### H6\n{end-vocabulary}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
				"<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<h6>H6</h6>\n" +
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
			// ASR-6: no id attribute on block headers.
			if strings.Contains(string(got), ` id=`) {
				t.Fatalf("ToHTML() emitted an id attribute on a block header (ASR-6 violated): %q", string(got))
			}
		})
	}
}

// TestRenderVocabularyNoHeaderUnchanged is the ASR-3 byte-identical regression
// control: a vocabulary block with no header/note lines must produce output
// that is byte-identical to the pre-change golden (SPECS §12.2, ASR-3).
func TestRenderVocabularyNoHeaderUnchanged(t *testing.T) {
	input := "{start-vocabulary}\n你好\n{end-vocabulary}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">V</span></div>\n" +
		"<div class=\"vocabulary\" dir=\"ltr\">\n" +
		"<div class=\"vocabulary-item\">\n" +
		"<span class=\"vocabulary-phrase\">你好</span>\n" +
		"</div>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch (ASR-3 regression)\n got: %q\nwant: %q", string(got), want)
	}
}
