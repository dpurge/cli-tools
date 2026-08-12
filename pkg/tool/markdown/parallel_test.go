package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Parallel_Golden asserts the EXACT wrapper output for the
// {start-parallel}...{end-parallel} block per the NEW SPECS (ASR-1, ASR-3):
//
//   - The primary column (.parallel-cell.main) always contains .parallel-source
//     with dir="<source direction from marker>". When TranscriptionRaw is
//     non-empty, .parallel-transcription dir="ltr" is stacked below inside .main.
//   - The secondary column (.parallel-cell.secondary) carries the TRANSLATION
//     (book language, NO dir attribute — ASR-1 reversal from the old behavior).
//   - Every lone "---" now splits fields, not preserves a thematic break inside
//     the source (ASR-3 split-on-every-"---", cap 3).
//
// The first sub-case repurposes the old "thematic-break-preservation" test (the
// original parallel_test.go:23-44) to document the ASR-3 behavior change; see
// also TestParseParallel_SourceThematicBreakNowSplits for an explicit guard.
func TestToHTML_Parallel_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			// Was: "main cell contains thematic break, plus secondary after LAST ---"
			// Now (ASR-3): TWO "---" lines → 3 fields:
			//   source="First para.", translation="Second para in main.",
			//   transcription="Secondary cell."
			// The thematic break no longer survives into the source's HTML.
			name: "source, translation, transcription (former thematic-break-in-source case, ASR-3)",
			input: "{start-parallel}\n" +
				"First para.\n" +
				"\n" +
				"---\n" +
				"\n" +
				"Second para in main.\n" +
				"---\n" +
				"Secondary cell.\n" +
				"{end-parallel}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
				"<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<div class=\"parallel-source\" dir=\"ltr\">\n" +
				"<p>First para.</p>\n" +
				"\n</div>\n" +
				"<div class=\"parallel-transcription\" dir=\"ltr\">\n" +
				"<p>Secondary cell.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-cell secondary\">\n" +
				"<p>Second para in main.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// Source-only row: .parallel-source inside .main; no .secondary.
			name: "row with no secondary cell",
			input: "{start-parallel}\n" +
				"Just main cell text.\n" +
				"{end-parallel}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
				"<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<div class=\"parallel-source\" dir=\"ltr\">\n" +
				"<p>Just main cell text.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// Two rows: first has source+translation (no transcription); second is
			// source-only. .secondary carries NO dir attribute (ASR-1 reversal).
			name: "two rows",
			input: "{start-parallel}\n" +
				"Row1 main.\n" +
				"---\n" +
				"Row1 secondary.\n" +
				"===\n" +
				"Row2 main only.\n" +
				"{end-parallel}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
				"<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<div class=\"parallel-source\" dir=\"ltr\">\n" +
				"<p>Row1 main.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-cell secondary\">\n" +
				"<p>Row1 secondary.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<div class=\"parallel-source\" dir=\"ltr\">\n" +
				"<p>Row2 main only.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// script=arab: the SOURCE column gets dir="rtl" (ASR-1 reversal — the old
			// behavior put dir="rtl" on the SECONDARY). The .secondary carries NO dir.
			name: "script=arab: source dir=rtl; secondary has no dir (ASR-1 reversal)",
			input: "{start-parallel script=arab}\n" +
				"Main text.\n" +
				"---\n" +
				"Secondary text.\n" +
				"{end-parallel}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
				"<div class=\"parallel s-arab\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<div class=\"parallel-source\" dir=\"rtl\">\n" +
				"<p>Main text.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-cell secondary\">\n" +
				"<p>Secondary text.</p>\n" +
				"\n</div>\n" +
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

// TestToHTML_Parallel_MainCellFirstCharNotDropped is a focused regression test:
// the source cell text starts immediately after the marker line, and its first
// character must appear verbatim in the output.
func TestToHTML_Parallel_MainCellFirstCharNotDropped(t *testing.T) {
	input := "{start-parallel}\nZebra leads the main cell.\n{end-parallel}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "Zebra leads the main cell.") {
		t.Fatalf("main cell's first character was dropped (D2 regression), got: %q", string(got))
	}
}

// TestParallel_As_Rejected covers SPECS §5: parallel's main/secondary
// columns already carry both languages, so as= is rejected entirely, with
// a clearer message than the pre-unification blanket rejection.
func TestParallel_As_Rejected(t *testing.T) {
	input := "{start-parallel as=source}\nMain.\n{end-parallel}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-parallel}, got nil")
	}
	wantErr := "as= not applicable to {start-parallel}: its field languages are fixed"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}

// ---------------------------------------------------------------------
// SPECS §12.1 — AST / parser cases
//
// These verify parseParallelRows field-splitting behavior via ToHTML.
// The HTML output directly reflects which AST fields were populated
// (SourceRaw → .parallel-source, TranslationRaw → .parallel-cell.secondary,
// TranscriptionRaw → .parallel-transcription). All inputs omit lang=/script=
// so the scriptClass and blockDirection results are unambiguous (both empty
// → "ltr" and "" respectively, ASR-5 fallback).
// ---------------------------------------------------------------------

// TestParseParallel_OneField verifies that a single-field record (no "---")
// produces ParallelRow{SourceRaw: "source text"} with other fields "".
// Evidence: only .parallel-source; no .secondary; no .parallel-transcription.
func TestParseParallel_OneField(t *testing.T) {
	input := "{start-parallel}\nsource text\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source text</p>\n" +
		"\n</div>\n" +
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
	if strings.Contains(string(got), "parallel-cell secondary") {
		t.Error("one-field row must have no .parallel-cell secondary (TranslationRaw==\"\")")
	}
	if strings.Contains(string(got), "parallel-transcription") {
		t.Error("one-field row must have no .parallel-transcription (TranscriptionRaw==\"\")")
	}
}

// TestParseParallel_TwoFields verifies that "source\n---\ntranslation" yields
// SourceRaw="source", TranslationRaw="translation", TranscriptionRaw="".
func TestParseParallel_TwoFields(t *testing.T) {
	input := "{start-parallel}\nsource\n---\ntranslation\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>translation</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"</div>\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	if strings.Contains(string(got), "parallel-transcription") {
		t.Error("two-field row must have no .parallel-transcription (TranscriptionRaw==\"\")")
	}
}

// TestParseParallel_ThreeFields verifies that "source\n---\ntranslation\n---\ntranscription"
// populates all three fields (SPECS §3.2, §5.1).
func TestParseParallel_ThreeFields(t *testing.T) {
	input := "{start-parallel}\nsource\n---\ntranslation\n---\ntranscription\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source</p>\n" +
		"\n</div>\n" +
		"<div class=\"parallel-transcription\" dir=\"ltr\">\n" +
		"<p>transcription</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>translation</p>\n" +
		"\n</div>\n" +
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

// TestParseParallel_FourPlusChunksAbsorbed verifies the ASR-3 absorb policy:
// SplitN(s, "\n---\n", 3) caps at 3 fields; a 4th "---" stays verbatim inside
// TranscriptionRaw. In "a\n---\nb\n---\nc\n---\nd" → SourceRaw="a",
// TranslationRaw="b", TranscriptionRaw="c\n---\nd".
//
// Note: "c\n---" is a CommonMark setext heading (level 2), so TranscriptionRaw
// renders as <h2>c</h2><p>d</p>, not as a thematic break. The extra "---" is
// absorbed as authoring content, not treated as an error (§5.2 graceful degradation).
func TestParseParallel_FourPlusChunksAbsorbed(t *testing.T) {
	input := "{start-parallel}\na\n---\nb\n---\nc\n---\nd\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>a</p>\n" +
		"\n</div>\n" +
		"<div class=\"parallel-transcription\" dir=\"ltr\">\n" +
		"<h2 id=\"c\">c</h2>\n" +
		"<p>d</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>b</p>\n" +
		"\n</div>\n" +
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

// TestParseParallel_EmptyTranslationWithTranscription verifies §5.3 edge case:
// an empty field 2 with a non-empty field 3 is tolerated (not a parse error).
// "s\n---\n\n---\nt" → SourceRaw="s", TranslationRaw="" (empty), TranscriptionRaw="t".
// TranslationRaw=="" → no .secondary emitted; TranscriptionRaw=="t" → .parallel-transcription IS present.
func TestParseParallel_EmptyTranslationWithTranscription(t *testing.T) {
	input := "{start-parallel}\ns\n---\n\n---\nt\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>s</p>\n" +
		"\n</div>\n" +
		"<div class=\"parallel-transcription\" dir=\"ltr\">\n" +
		"<p>t</p>\n" +
		"\n</div>\n" +
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
	if strings.Contains(string(got), "parallel-cell secondary") {
		t.Error("empty TranslationRaw must produce no .parallel-cell secondary (§5.3 tolerated)")
	}
}

// TestParseParallel_MultipleRows verifies that two "==="-separated records produce
// two independent ParallelRows, each rendered in its own .parallel-row.
func TestParseParallel_MultipleRows(t *testing.T) {
	input := "{start-parallel}\nRow A\n===\nRow B\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>Row A</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>Row B</p>\n" +
		"\n</div>\n" +
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
	rowCount := strings.Count(string(got), `class="parallel-row"`)
	if rowCount != 2 {
		t.Errorf("expected 2 .parallel-row divs, got %d", rowCount)
	}
}

// TestParseParallel_SourceThematicBreakNowSplits documents the ASR-3 behavior
// change that replaces the OLD thematic-break-preservation trick (LastIndex,
// parallel_test.go:23-44 original). The old behavior kept a "---" inside the
// source verbatim (rendering as <hr />) when it was not the LAST "---" in a
// record. The NEW behavior (SplitN cap 3) treats every "---" as a field separator.
//
// Input with TWO "---" lines → 3 fields:
//
//	field 1 (source):       "First para."
//	field 2 (translation):  "Second para in main."
//	field 3 (transcription): "Secondary cell."
//
// The <hr /> that used to appear inside the source is now GONE. This is
// intentional (ASR-3, HITL-approved D2) — real-world impact was nil per the
// pre-delivery backward-compat check in coding-flow-state.md.
func TestParseParallel_SourceThematicBreakNowSplits(t *testing.T) {
	input := "{start-parallel}\n" +
		"First para.\n" +
		"\n" +
		"---\n" +
		"\n" +
		"Second para in main.\n" +
		"---\n" +
		"Secondary cell.\n" +
		"{end-parallel}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}

	// Source must NOT contain a thematic break (the "---" is now a field separator).
	if strings.Contains(string(got), "<hr />") {
		t.Error("source cell must not contain <hr /> — '---' is now a field separator (ASR-3), not a thematic break within the source")
	}
	// Field 1 rendered as source.
	if !strings.Contains(string(got), "<p>First para.</p>") {
		t.Error("source must contain 'First para.'")
	}
	// Field 2 went to translation (.secondary), not merged with source.
	if !strings.Contains(string(got), "<p>Second para in main.</p>") {
		t.Error("translation must contain 'Second para in main.' (the former second paragraph)")
	}
	// Field 3 is transcription, stacked inside .main.
	if !strings.Contains(string(got), "parallel-transcription") {
		t.Error("transcription wrapper must be present (field 3 = 'Secondary cell.')")
	}
	if !strings.Contains(string(got), "<p>Secondary cell.</p>") {
		t.Error("transcription must contain 'Secondary cell.'")
	}
}

// ---------------------------------------------------------------------
// SPECS §12.2 — HTML render contract
//
// These verify the specific rendering contracts for dir attribute placement,
// wrapper class names, and per-field HTML structure. They use the same
// ToHTML API as the parser tests but focus on rendering semantics.
// ---------------------------------------------------------------------

// TestRenderParallelHTML_OneField verifies: .parallel-source dir="ltr" inside
// .main; no .parallel-transcription; no .secondary. Backward class names preserved
// (§6 backward CSS compat): "parallel", "parallel-row", "parallel-cell main".
func TestRenderParallelHTML_OneField(t *testing.T) {
	input := "{start-parallel}\nsource text\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source text</p>\n" +
		"\n</div>\n" +
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
	// Verify backward-compat class names (§6).
	for _, cls := range []string{`class="parallel"`, `class="parallel-row"`, `class="parallel-cell main"`} {
		if !strings.Contains(string(got), cls) {
			t.Errorf("expected backward-compat class %q in output", cls)
		}
	}
	if strings.Contains(string(got), "parallel-transcription") {
		t.Error("one-field row must not have .parallel-transcription")
	}
	if strings.Contains(string(got), "parallel-cell secondary") {
		t.Error("one-field row must not have .parallel-cell secondary")
	}
}

// TestRenderParallelHTML_TwoFields verifies: .main with .parallel-source dir="ltr";
// .secondary present with dir ABSENT (translation = book language, ASR-1 reversal).
// The dir-absence is verified both via exact string match and an explicit substring
// check per SPECS §12.2 (reviewer-driven correction).
func TestRenderParallelHTML_TwoFields(t *testing.T) {
	input := "{start-parallel}\nsource\n---\ntranslation\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>translation</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"</div>\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// Explicit dir-absence check on the secondary cell substring (SPECS §12.2):
	// the translation is book language → dir must be genuinely absent (not dir="").
	const secondaryMarker = `class="parallel-cell secondary"`
	idx := strings.Index(string(got), secondaryMarker)
	if idx == -1 {
		t.Fatal("expected .parallel-cell secondary in output")
	}
	rest := string(got)[idx:]
	endDiv := strings.Index(rest, "</div>")
	secondaryCellSubstr := rest[:endDiv]
	if strings.Contains(secondaryCellSubstr, "dir=") {
		t.Fatalf("dir= must be absent from .parallel-cell secondary (ASR-1 reversal), got: %q", secondaryCellSubstr)
	}
	if strings.Contains(string(got), "parallel-transcription") {
		t.Error("two-field row must not have .parallel-transcription")
	}
}

// TestRenderParallelHTML_ThreeFields verifies: .main contains .parallel-source then
// .parallel-transcription dir="ltr" (pinned, ASR-6 — not inherited from marker);
// .secondary present with dir ABSENT (ASR-1).
func TestRenderParallelHTML_ThreeFields(t *testing.T) {
	input := "{start-parallel}\nsource\n---\ntranslation\n---\ntranscription\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"ltr\">\n" +
		"<p>source</p>\n" +
		"\n</div>\n" +
		"<div class=\"parallel-transcription\" dir=\"ltr\">\n" +
		"<p>transcription</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>translation</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"</div>\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// Transcription dir pinned "ltr" regardless of the marker script (ASR-6).
	if !strings.Contains(string(got), `class="parallel-transcription" dir="ltr"`) {
		t.Error("expected .parallel-transcription dir=\"ltr\" (pinned per ASR-6, matches vocabulary/models)")
	}
	// Secondary has no dir (ASR-1).
	const secondaryMarker = `class="parallel-cell secondary"`
	idx := strings.Index(string(got), secondaryMarker)
	if idx == -1 {
		t.Fatal("expected .parallel-cell secondary in output")
	}
	rest := string(got)[idx:]
	endDiv := strings.Index(rest, "</div>")
	secondaryCellSubstr := rest[:endDiv]
	if strings.Contains(secondaryCellSubstr, "dir=") {
		t.Fatalf("dir= must be absent from .parallel-cell secondary (ASR-1), got: %q", secondaryCellSubstr)
	}
}

// TestRenderParallelHTML_RTLSource verifies that script=arab causes the SOURCE
// column to receive dir="rtl" (the marker's script drives source direction, ASR-4),
// while .secondary still has NO dir attribute (ASR-1). This is the exact reversal
// of the old behavior where script=arab put dir="rtl" on the SECONDARY cell.
func TestRenderParallelHTML_RTLSource(t *testing.T) {
	input := "{start-parallel script=arab}\nsource\n---\ntranslation\n{end-parallel}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">P</span></div>\n" +
		"<div class=\"parallel s-arab\">\n" +
		"<div class=\"parallel-row\">\n" +
		"<div class=\"parallel-cell main\">\n" +
		"<div class=\"parallel-source\" dir=\"rtl\">\n" +
		"<p>source</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"<div class=\"parallel-cell secondary\">\n" +
		"<p>translation</p>\n" +
		"\n</div>\n" +
		"</div>\n" +
		"</div>\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// Source is RTL per marker (ASR-4: column internal direction follows marker).
	if !strings.Contains(string(got), `class="parallel-source" dir="rtl"`) {
		t.Error("expected .parallel-source dir=\"rtl\" for script=arab")
	}
	// Secondary has no dir even when source is RTL (ASR-1).
	const secondaryMarker = `class="parallel-cell secondary"`
	idx := strings.Index(string(got), secondaryMarker)
	if idx == -1 {
		t.Fatal("expected .parallel-cell secondary in output")
	}
	rest := string(got)[idx:]
	endDiv := strings.Index(rest, "</div>")
	secondaryCellSubstr := rest[:endDiv]
	if strings.Contains(secondaryCellSubstr, "dir=") {
		t.Fatalf("dir= must be absent from .parallel-cell secondary even for RTL marker (ASR-1), got: %q", secondaryCellSubstr)
	}
}

// TestRenderParallelHTML_NoScriptFallback verifies ASR-5: a {start-parallel} marker
// without lang=/script= degrades gracefully — blockDirection("") == "ltr", so
// .parallel-source gets dir="ltr"; no s-<script> class on the container.
func TestRenderParallelHTML_NoScriptFallback(t *testing.T) {
	input := "{start-parallel}\nsource\n---\ntranslation\n{end-parallel}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	// blockDirection("") = "ltr" → source is LTR (ASR-5 fallback).
	if !strings.Contains(string(got), `class="parallel-source" dir="ltr"`) {
		t.Errorf("expected .parallel-source dir=\"ltr\" for a no-script marker (ASR-5), got: %q", string(got))
	}
	// No scriptClass → no "s-" class on container.
	if strings.Contains(string(got), `class="parallel s-`) {
		t.Errorf("expected no s-<script> class for empty script, got: %q", string(got))
	}
	// Secondary still no dir.
	const secondaryMarker = `class="parallel-cell secondary"`
	idx := strings.Index(string(got), secondaryMarker)
	if idx == -1 {
		t.Fatal("expected .parallel-cell secondary in output")
	}
	rest := string(got)[idx:]
	endDiv := strings.Index(rest, "</div>")
	secondaryCellSubstr := rest[:endDiv]
	if strings.Contains(secondaryCellSubstr, "dir=") {
		t.Fatalf("dir= must be absent from .parallel-cell secondary for no-script marker, got: %q", secondaryCellSubstr)
	}
}
