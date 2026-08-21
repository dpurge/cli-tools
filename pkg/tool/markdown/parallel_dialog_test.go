package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// ---------------------------------------------------------------------
// AST / parser cases
//
// {start-parallel-dialog} combines {start-parallel}'s row/field grid
// (rows on "===", fields on "---", cap 3: source/translation/transcription)
// with {start-dialog}'s per-field turn/heading grammar: every present field
// must parse to exactly one turn (Header+Content) or heading (ItemHeader).
// Unlike {start-parallel}, translation is mandatory (a row with no "---"
// is an error) and transcription (when present) is turn-structured too,
// not raw markdown.
// ---------------------------------------------------------------------

// TestToHTML_ParallelDialog_Golden asserts the exact wrapper/row/cell shape:
// a .parallel-dialog wrapper, one .parallel-dialog-row per row, a .main cell
// (source, plus a stacked .parallel-dialog-transcription when present) and a
// .secondary cell (translation) — mirroring renderParallel's column shape,
// with each field's turn rendered as a .parallel-dialog-item(header+content).
func TestToHTML_ParallelDialog_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "named turns, two fields",
			input: "{start-parallel-dialog}\n" +
				"@Ion:\n  Buna dimineata!\n" +
				"---\n" +
				"@Jan:\n  Dzien dobry!\n" +
				"{end-parallel-dialog}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">R</span></div>\n" +
				"<div class=\"parallel-dialog\">\n" +
				"<div class=\"parallel-dialog-row\">\n" +
				"<div class=\"parallel-dialog-cell main\">\n" +
				"<div class=\"parallel-dialog-source\" dir=\"ltr\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">Ion:</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Buna dimineata!</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-dialog-cell secondary\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">Jan:</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Dzien dobry!</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// "--:" turns (anonymous continuation, always em dash — no
			// speaker lookup) plus a transcription field: field 3 is
			// turn-structured exactly like source, stacked below it inside
			// .main, with dir="ltr" pinned (matches {start-parallel}'s own
			// transcription placement).
			name: "anonymous turns with transcription (field 3 is turn-structured)",
			input: "{start-parallel-dialog script=latn}\n" +
				"--:\n  Buna ziua!\n" +
				"---\n" +
				"--:\n  Dzien dobry!\n" +
				"---\n" +
				"--:\n  Buna ziua translit\n" +
				"{end-parallel-dialog}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">R</span></div>\n" +
				"<div class=\"parallel-dialog s-latn\">\n" +
				"<div class=\"parallel-dialog-row\">\n" +
				"<div class=\"parallel-dialog-cell main\">\n" +
				"<div class=\"parallel-dialog-source\" dir=\"ltr\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">—</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Buna ziua!</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-dialog-transcription\" dir=\"ltr\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">—</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Buna ziua translit</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-dialog-cell secondary\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">—</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Dzien dobry!</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// Title row: both fields are bare headings instead of turns —
			// each renders as its own <hN>, no .parallel-dialog-item wrapper.
			name: "title row (per-field heading, not a turn)",
			input: "{start-parallel-dialog}\n" +
				"# Chapter One\n" +
				"---\n" +
				"# Rozdzial pierwszy\n" +
				"{end-parallel-dialog}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">R</span></div>\n" +
				"<div class=\"parallel-dialog\">\n" +
				"<div class=\"parallel-dialog-row\">\n" +
				"<div class=\"parallel-dialog-cell main\">\n" +
				"<div class=\"parallel-dialog-source\" dir=\"ltr\">\n" +
				"<h1>Chapter One</h1>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-dialog-cell secondary\">\n" +
				"<h1>Rozdzial pierwszy</h1>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// script=arab: SOURCE column dir="rtl" (marker-driven, mirrors
			// {start-parallel}'s ASR-1 column rule); secondary carries no
			// dir (translation inherits book direction).
			name: "script=arab: source dir=rtl, secondary has no dir",
			input: "{start-parallel-dialog script=arab}\n" +
				"@Ion:\n  Marhaba.\n" +
				"---\n" +
				"@Jan:\n  Hej.\n" +
				"{end-parallel-dialog}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">R</span></div>\n" +
				"<div class=\"parallel-dialog s-arab\">\n" +
				"<div class=\"parallel-dialog-row\">\n" +
				"<div class=\"parallel-dialog-cell main\">\n" +
				"<div class=\"parallel-dialog-source\" dir=\"rtl\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">Ion:</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Marhaba.</p>\n</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-dialog-cell secondary\">\n" +
				"<div class=\"parallel-dialog-item\">\n" +
				"<div class=\"parallel-dialog-header\">Jan:</div>\n" +
				"<div class=\"parallel-dialog-content\"><p>Hej.</p>\n</div>\n" +
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

// TestToHTML_ParallelDialog_MultipleRows verifies that two "==="-separated
// records produce two independent rows.
func TestToHTML_ParallelDialog_MultipleRows(t *testing.T) {
	input := "{start-parallel-dialog}\n" +
		"@Ion:\n  Row1 src.\n---\n@Jan:\n  Row1 tr.\n" +
		"===\n" +
		"@Ion:\n  Row2 src.\n---\n@Jan:\n  Row2 tr.\n" +
		"{end-parallel-dialog}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	rowCount := strings.Count(string(got), `class="parallel-dialog-row"`)
	if rowCount != 2 {
		t.Errorf("expected 2 .parallel-dialog-row divs, got %d", rowCount)
	}
	for _, want := range []string{"Row1 src.", "Row1 tr.", "Row2 src.", "Row2 tr."} {
		if !strings.Contains(string(got), want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

// TestParallelDialog_As_Rejected mirrors TestParallel_As_Rejected: both
// columns' languages are already fixed by field position, so as= is
// rejected entirely.
func TestParallelDialog_As_Rejected(t *testing.T) {
	input := "{start-parallel-dialog as=source}\n@Ion:\n  Hi.\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-parallel-dialog}, got nil")
	}
	wantErr := "as= not applicable to {start-parallel-dialog}: its field languages are fixed"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}

// TestParallelDialog_MissingTranslation_Errors verifies that a row with no
// "---" (translation field absent) is an error — unlike plain
// {start-parallel}, translation is mandatory here.
func TestParallelDialog_MissingTranslation_Errors(t *testing.T) {
	input := "{start-parallel-dialog}\n@Ion:\n  Hi.\n{end-parallel-dialog}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for a row with no translation field, got nil")
	}
	if !strings.Contains(err.Error(), "missing its translation field") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "missing its translation field")
	}
}

// TestParallelDialog_MultipleItemsInField_Errors verifies that a field
// holding more than one turn is an error — unlike a {start-dialog} block, a
// parallel-dialog field never holds a run of several turns.
func TestParallelDialog_MultipleItemsInField_Errors(t *testing.T) {
	input := "{start-parallel-dialog}\n@Ion:\n  Hi.\n@Ion:\n  Again.\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for a field with two turns, got nil")
	}
	if !strings.Contains(err.Error(), "exactly one turn or heading, got 2") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "exactly one turn or heading, got 2")
	}
}

// TestParallelDialog_BadIndentation_Errors mirrors {start-dialog}'s D3
// bad-indentation error, scoped to a single field.
func TestParallelDialog_BadIndentation_Errors(t *testing.T) {
	input := "{start-parallel-dialog}\n@Ion:\nBadly indented\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected a bad-indentation error, got nil")
	}
	if !strings.Contains(err.Error(), "wrong line indentation for parallel-dialog item: Badly indented") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "wrong line indentation for parallel-dialog item: Badly indented")
	}
}

// TestParallelDialog_EmptyField_TreatedAsBlankTurn documents that a
// genuinely empty middle field ("---\n\n---") is tolerated as a blank turn
// (Header="", Content="") rather than an error — mirroring
// TestParseParallel_EmptyTranslationWithTranscription's "empty field
// tolerated" precedent for plain {start-parallel}.
func TestParallelDialog_EmptyField_TreatedAsBlankTurn(t *testing.T) {
	input := "{start-parallel-dialog}\n@Ion:\n  Hi.\n---\n\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error for an empty middle field: %v", err)
	}
	if !strings.Contains(string(got), "parallel-dialog-transcription") {
		t.Error("expected a .parallel-dialog-transcription wrapper even for a blank turn")
	}
}

// TestParallelDialog_BlankLinesAroundSeparators_NoOp is a regression test
// for a real-world report: a "---"/"==="-separated field that itself has a
// blank line right before/after the separator (e.g. because the source was
// written with a paragraph break for readability) used to error with "field
// must contain exactly one turn or heading, got 2 item(s)". The extra
// leading blank line surviving into the next field was buffered before any
// "@X:" header was seen, and flush() turned it into a spurious zero-content
// item as soon as the real header arrived. A blank line INSIDE a turn
// (between two indented content lines, preserved as a paragraph break) was
// never affected and still isn't.
func TestParallelDialog_BlankLinesAroundSeparators_NoOp(t *testing.T) {
	input := "{start-parallel-dialog}\n" +
		"@Ion:\n" +
		"  Row1 src line1.\n" +
		"\n" +
		"  Row1 src line2.\n" +
		"\n" +
		"---\n" +
		"\n" +
		"@Jan:\n" +
		"  Row1 tr.\n" +
		"\n" +
		"===\n" +
		"\n" +
		"@Ion:\n" +
		"  Row2 src.\n" +
		"---\n" +
		"@Jan:\n" +
		"  Row2 tr.\n" +
		"\n" +
		"{end-parallel-dialog}\n"

	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error for blank lines around separators: %v", err)
	}
	rowCount := strings.Count(string(got), `class="parallel-dialog-row"`)
	if rowCount != 2 {
		t.Errorf("expected 2 .parallel-dialog-row divs, got %d", rowCount)
	}
	for _, want := range []string{"Row1 src line1.", "Row1 src line2.", "Row1 tr.", "Row2 src.", "Row2 tr."} {
		if !strings.Contains(string(got), want) {
			t.Errorf("expected output to contain %q", want)
		}
	}
}

// ---------------------------------------------------------------------
// Typst emission (Go-side dict-string assertions)
// ---------------------------------------------------------------------

// TestToTypst_ParallelDialog_Golden asserts the exact #parallel-dialog(...)
// dict emission: source-dir:/script: parameters, per-field item dicts
// reusing dialog()'s exact (header:,content:) / (kind:,level:,text:) shape,
// and transcription: present ONLY when the row supplies a third field
// (key omission, not empty content — mirrors renderParallelTypst's
// "transcription" in r idiom).
func TestToTypst_ParallelDialog_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "two fields, no transcription key",
			input: "{start-parallel-dialog}\n" +
				"@Ion:\n  Buna dimineata!\n---\n@Jan:\n  Dzien dobry!\n" +
				"{end-parallel-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"R\")]\n\n" +
				"#parallel-dialog(source-dir: ltr, script: \"\",\n" +
				"  (source: (header: \"Ion:\", content: [Buna dimineata!\n\n]), translation: (header: \"Jan:\", content: [Dzien dobry!\n\n])),\n" +
				")\n\n",
		},
		{
			name: "three fields — transcription key present",
			input: "{start-parallel-dialog script=latn}\n" +
				"--:\n  Buna ziua!\n---\n--:\n  Dzien dobry!\n---\n--:\n  Buna ziua translit\n" +
				"{end-parallel-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"R\")]\n\n" +
				"#parallel-dialog(source-dir: ltr, script: \"latn\",\n" +
				"  (source: (header: \"—\", content: [Buna ziua!\n\n]), translation: (header: \"—\", content: [Dzien dobry!\n\n]), transcription: (header: \"—\", content: [Buna ziua translit\n\n])),\n" +
				")\n\n",
		},
		{
			name:  "title row — kind/level/text dict, no header/content",
			input: "{start-parallel-dialog}\n# Chapter One\n---\n# Rozdzial pierwszy\n{end-parallel-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"R\")]\n\n" +
				"#parallel-dialog(source-dir: ltr, script: \"\",\n" +
				"  (source: (kind: \"header\", level: 1, text: \"Chapter One\"), translation: (kind: \"header\", level: 1, text: \"Rozdzial pierwszy\")),\n" +
				")\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToTypst([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToTypst() unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToTypst() mismatch\n got: %q\nwant: %q", string(got), tc.want)
			}
		})
	}
	// Explicit key-omission check on the two-field case: "transcription:" must be
	// genuinely absent, not present-with-empty-content.
	got, err := markdown.ToTypst([]byte(tests[0].input))
	if err != nil {
		t.Fatalf("ToTypst() unexpected error: %v", err)
	}
	if strings.Contains(string(got), "transcription:") {
		t.Error("two-field row must have no transcription: key at all (key omission, not empty content)")
	}
}

// TestToTypst_ParallelDialog_Err verifies that a parse-time error (bad
// indentation, missing translation field, as= rejection, etc.) surfaces out
// of ToTypst with no output, mirroring TestToTypst_Dialog_Err/
// TestToTypst_Parallel's error handling.
func TestToTypst_ParallelDialog_Err(t *testing.T) {
	input := "{start-parallel-dialog}\n@Ion:\nBadly indented\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n"

	got, err := markdown.ToTypst([]byte(input))
	if err == nil {
		t.Fatalf("ToTypst(%q) expected a non-nil error, got nil (output: %q)", input, got)
	}
	if len(got) != 0 {
		t.Fatalf("ToTypst(%q) expected no output alongside the error, got: %q", input, got)
	}
}

// ---------------------------------------------------------------------
// MDX round-trip
// ---------------------------------------------------------------------

// TestToMDX_ParallelDialog_Golden asserts the exact `parallel-dialog` fence
// shape: rows joined by "===", fields by "---", each turn re-serialized as
// its original "@Speaker:"/"--:" marker line plus 2-space-indented content —
// literal fence body, never re-run through a renderer.
func TestToMDX_ParallelDialog_Golden(t *testing.T) {
	input := "{start-parallel-dialog}\n" +
		"@Ion:\n  Row1 src.\n---\n@Jan:\n  Row1 tr.\n" +
		"===\n" +
		"@Ion:\n  Row2 src.\n---\n@Jan:\n  Row2 tr.\n" +
		"{end-parallel-dialog}\n"
	want := "<Text lang=\"en\" script=\"latn\">\n\n" +
		"```parallel-dialog lang=en script=latn\n" +
		"@Ion:\n  Row1 src.\n---\n@Jan:\n  Row1 tr.\n" +
		"===\n" +
		"@Ion:\n  Row2 src.\n---\n@Jan:\n  Row2 tr.\n" +
		"```\n\n</Text>\n"

	got, err := markdown.ToMDX([]byte(input), "en", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestToMDX_ParallelDialog_RoundTripStable verifies that serializing to MDX
// and re-parsing the fence body produces an identical block, evidenced by
// byte-identical HTML output — mirrors TestToMDX_Parallel_RoundTripStable.
func TestToMDX_ParallelDialog_RoundTripStable(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "two fields",
			input: "{start-parallel-dialog}\n@Ion:\n  Hi.\n---\n@Jan:\n  Hej.\n{end-parallel-dialog}\n",
		},
		{
			name:  "three fields",
			input: "{start-parallel-dialog}\n--:\n  Hi.\n---\n--:\n  Hej.\n---\n--:\n  Hi translit\n{end-parallel-dialog}\n",
		},
		{
			name:  "title row",
			input: "{start-parallel-dialog}\n# Chapter One\n---\n# Rozdzial pierwszy\n{end-parallel-dialog}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMDX, err := markdown.ToMDX([]byte(tc.input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX() unexpected error: %v", err)
			}
			body := string(gotMDX)
			body = strings.TrimPrefix(body, "<Text lang=\"lat\" script=\"latn\">\n\n```parallel-dialog lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "```\n\n</Text>\n")
			body = strings.TrimSuffix(body, "\n")

			originalHTML, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("original ToHTML() unexpected error: %v", err)
			}
			reparsedHTML, err := markdown.ToHTML([]byte("{start-parallel-dialog}\n" + body + "\n{end-parallel-dialog}\n"))
			if err != nil {
				t.Fatalf("reparsed ToHTML() unexpected error: %v", err)
			}
			if string(originalHTML) != string(reparsedHTML) {
				t.Fatalf("round-trip mismatch\noriginal:  %q\nreparsed: %q", originalHTML, reparsedHTML)
			}
		})
	}
}
