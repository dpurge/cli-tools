package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// runTypstGolden is the shared table-driven harness used by every
// TestToTypst_* function below: each case's markdown input must produce
// the EXACT (byte-identical) Typst output, matching the determinism
// requirement (ASR-3) and "matches SPECS §4 exactly" (AC2). Only the
// public markdown.ToTypst API is exercised — escapeTypstMarkup,
// escapeTypstString and typographerEntities are unexported and are
// therefore verified only indirectly, through inputs chosen to make
// their effect on the rendered output unambiguous (mirrors how the
// existing HTML suite only ever calls through markdown.ToHTML).
func runTypstGolden(t *testing.T, tests []struct {
	name  string
	input string
	want  string
}) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToTypst([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToTypst(%q) unexpected error: %v", tc.input, err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToTypst(%q) mismatch\n got: %q\nwant: %q", tc.input, string(got), tc.want)
			}
		})
	}
}

// TestToTypst_Headings covers SPECS §4's Heading row: "="xLevel + " " +
// inline + "\n\n".
func TestToTypst_Headings(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "levels 1 through 3",
			input: "# H1\n\n## H2\n\n### H3\n",
			want:  "= H1\n\n== H2\n\n=== H3\n\n",
		},
	})
}

// TestToTypst_ParagraphAndBreaks covers Paragraph ("\n\n" terminator) and
// Text's break handling (FR-6): a soft break emits a single space (never
// a bare newline, which would risk a stray Typst list marker on the next
// line), and a hard break — whether spelled as a trailing backslash or as
// CommonMark's two-trailing-spaces form — emits Typst's own hard-break
// shorthand `\ ` followed by a newline (empirically verified against the
// installed Typst 0.15.1 to force a line break independent of source
// newline placement).
func TestToTypst_ParagraphAndBreaks(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain paragraph",
			input: "Plain paragraph text.\n",
			want:  "Plain paragraph text\\.\n\n",
		},
		{
			name:  "soft break becomes a space, not a newline",
			input: "line one\nline two\n",
			want:  "line one line two\n\n",
		},
		{
			name:  "hard break via two trailing spaces",
			input: "line one  \nline two\n",
			want:  "line one\\ \nline two\n\n",
		},
		{
			name:  "hard break via trailing backslash",
			input: "line one\\\nline two\n",
			want:  "line one\\ \nline two\n\n",
		},
	})
}

// TestToTypst_Emphasis covers SPECS §4's function-form emphasis: Level 1
// -> #emph[...], Level 2 -> #strong[...] (Decision D2).
func TestToTypst_Emphasis(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "level 1 (em)", input: "*em*\n", want: "#emph[em]\n\n"},
		{name: "level 2 (strong)", input: "**strong**\n", want: "#strong[strong]\n\n"},
	})
}

// TestToTypst_CodeSpan covers #raw("...") emission, including the
// trailing-newline-becomes-space rule mirrored from
// renderer/html/html.go's renderCodeSpan for a code span spanning a soft
// line break.
func TestToTypst_CodeSpan(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "single line", input: "`code`\n", want: "#raw(\"code\")\n\n"},
		{name: "spans a line break, collapses to a space", input: "`a\nb`\n", want: "#raw(\"a b\")\n\n"},
	})
}

// TestToTypst_Link covers #link("dest")[label] (Title intentionally
// ignored, SPECS §4).
func TestToTypst_Link(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic link",
			input: "[text](https://example.com/a?b=1)\n",
			want:  "#link(\"https://example.com/a?b=1\")[text]\n\n",
		},
	})
}

// TestToTypst_AutoLink covers URL and email autolinks; an email autolink
// gets a "mailto:" prefix (SPECS §4, mirrors renderer/html/html.go's
// renderAutoLink).
func TestToTypst_AutoLink(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL autolink",
			input: "<https://example.com>\n",
			want:  "#link(\"https://example.com\")[https:\\/\\/example\\.com]\n\n",
		},
		{
			name:  "email autolink gets mailto: prefix",
			input: "<mail@example.com>\n",
			want:  "#link(\"mailto:mail@example.com\")[mail\\@example\\.com]\n\n",
		},
	})
}

// TestToTypst_Image covers #image("dest") with alt text dropped (Decision
// D6, v1 minimal) and no trailing children walked.
func TestToTypst_Image(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "image, alt dropped",
			input: "![alt](assets/pic.png)\n",
			want:  "#image(\"assets/pic.png\")\n\n",
		},
	})
}

// TestToTypst_List covers #list(...)/#enum(...) function form (Decision
// D2) for unordered/ordered lists, plus a nested list inside a tight
// list item.
func TestToTypst_List(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unordered (tight)",
			input: "- one\n- two\n",
			want:  "#list(\n[one\n],\n[two\n],\n)\n\n",
		},
		{
			name:  "ordered (tight)",
			input: "1. one\n2. two\n",
			want:  "#enum(\n[one\n],\n[two\n],\n)\n\n",
		},
		{
			name:  "nested list inside a list item",
			input: "- one\n  - nested\n- two\n",
			want:  "#list(\n[one\n#list(\n[nested\n],\n)\n\n],\n[two\n],\n)\n\n",
		},
	})
}

// TestToTypst_Blockquote covers #quote(block: true)[...].
func TestToTypst_Blockquote(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "single paragraph", input: "> quoted text\n", want: "#quote(block: true)[quoted text\n\n]\n\n"},
	})
}

// TestToTypst_CodeBlocks covers fenced code (with and without an info
// string, SPECS §4: lang omitted when empty) and indented code, all
// emitted via #raw(block: true, ...).
func TestToTypst_CodeBlocks(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced with language",
			input: "```go\nfmt.Println(1)\n```\n",
			want:  "#raw(block: true, lang: \"go\", \"fmt.Println(1)\\n\")\n\n",
		},
		{
			name:  "fenced without language",
			input: "```\nplain\n```\n",
			want:  "#raw(block: true, \"plain\\n\")\n\n",
		},
		{
			name:  "indented code block",
			input: "    x := 1\n",
			want:  "#raw(block: true, \"x := 1\\n\")\n\n",
		},
	})
}

// TestToTypst_ThematicBreak covers the exact `#line(length: 100%)\n`
// emission (SPECS §4: single trailing newline, no blank-line padding).
func TestToTypst_ThematicBreak(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "thematic break", input: "---\n", want: "#line(length: 100%)\n"},
	})
}

// TestToTypst_Table covers #table(columns: (1fr,...), align: (...), <cells>) with
// all three GFM alignments plus the default (AlignNone -> Typst `auto`),
// and confirms the header row's cells precede the body row's cells in
// the emitted positional-argument order. Fractional tracks make all markdown
// tables fill the available text width (D12 global approach).
func TestToTypst_Table(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "left/center/right alignment, header then body cells",
			input: "| L | C | R |\n|:--|:-:|--:|\n| a | b | c |\n",
			want:  "#table(columns: (1fr, 1fr, 1fr), align: (left, center, right),\n[L],\n[C],\n[R],\n[a],\n[b],\n[c],\n)\n\n",
		},
		{
			name:  "no alignment marker maps to auto",
			input: "| A | B |\n|---|---|\n| 1 | 2 |\n",
			want:  "#table(columns: (1fr, 1fr), align: (auto, auto),\n[A],\n[B],\n[1],\n[2],\n)\n\n",
		},
	})
}

// TestToTypst_Strikethrough covers #strike[...].
func TestToTypst_Strikethrough(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "struck text", input: "~~gone~~\n", want: "#strike[gone]\n\n"},
	})
}

// TestToTypst_DefinitionList covers the common one-term/one-description
// case of #terms(terms.item[term][desc], ...) (SPECS §4; multi-term or
// multi-description pairings are a documented, deliberately out-of-scope
// simplification per typst_render.go's renderDefinitionListTypst comment).
func TestToTypst_DefinitionList(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "one term, one description",
			input: "Term\n: Definition text.\n",
			want:  "#terms(\nterms.item[Term][Definition text\\.\n],\n)\n\n",
		},
	})
}

// TestToTypst_RawHTML_EmitsNothing covers the SPECS §4 fallback row for
// RawHTML/HTMLBlock: an inline HTML tag disappears but its sibling text
// is unaffected, while an HTML block (whose content lives in Lines(), not
// child text nodes) disappears in its entirety — mirroring the HTML
// renderer's Unsafe=false "<!-- raw HTML omitted -->" behavior, except
// Typst has no comment placeholder to emit, so nothing at all is written.
func TestToTypst_RawHTML_EmitsNothing(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "inline tag dropped, sibling text kept",
			input: "Some <b>raw html</b> inline.\n",
			want:  "Some raw html inline\\.\n\n",
		},
		{
			name:  "html block dropped entirely",
			input: "Before.\n\n<div>block html</div>\n\nAfter.\n",
			want:  "Before\\.\n\nAfter\\.\n\n",
		},
	})
}

// TestToTypst_Vocabulary_Golden covers the #vocabulary(...) custom-block
// mapping (SPECS §4/§7), including empty (dropped) fields all rendering
// as empty string literals.
func TestToTypst_Vocabulary_Golden(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full line: phrase, grammar, transcription, translation",
			input: "{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n" +
				"#vocabulary(dir: ltr, script: \"\",\n" +
				"  (phrase: \"你好\", grammar: \"noun\", transcription: \"nǐ hǎo\", translation: \"hello\"),\n" +
				")\n\n",
		},
		{
			name:  "phrase only, other fields empty",
			input: "{start-vocabulary}\n再见\n{end-vocabulary}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n" +
				"#vocabulary(dir: ltr, script: \"\",\n" +
				"  (phrase: \"再见\", grammar: \"\", transcription: \"\", translation: \"\"),\n" +
				")\n\n",
		},
	})
}

// TestToTypst_Dialog_Golden covers the #dialog(...) custom-block mapping,
// including that a "--:" header renders as "—" and "@Bob:" keeps its
// trailing colon, and that dialog content recurses through ToTypst (a
// **bold** run inside dialog content is rendered via #strong[...], not
// left as literal markdown).
func TestToTypst_Dialog_Golden(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "anonymous and named turns, content recurses through ToTypst",
			input: "{start-dialog}\n" +
				"--:\n" +
				"  Hello **there**.\n" +
				"@Bob:\n" +
				"  Hi!\n" +
				"{end-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"D\")]\n\n" +
				"#dialog(dir: ltr, script: \"\", role: \"source\",\n" +
				"  (header: \"—\", content: [Hello #strong[there]\\.\n\n]),\n" +
				"  (header: \"Bob:\", content: [Hi!\n\n]),\n" +
				")\n\n",
		},
		{
			// RTL integration golden: script=arab → blockDirection("arab")="rtl"
			// → badgeOnlyTypst("D","rtl") emits align(right)[...] (NFR-4).
			// The badge block begins with #block(..., align(right)[#_ctbadge("D")])
			// and #dialog carries dir: rtl + script: "arab".
			name:  "script=arab: RTL align(right) badge and dialog dir: rtl",
			input: "{start-dialog script=arab}\n--:\n  مرحبا.\n{end-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge(\"D\")])\n\n" +
				"#dialog(dir: rtl, script: \"arab\", role: \"source\",\n" +
				"  (header: \"—\", content: [مرحبا\\.\n\n]),\n" +
				")\n\n",
		},
	})
}

// TestToTypst_Dialog_Err asserts that a Dialog block with a bad-indentation
// content line (D3, ast.go's Dialog.Err) surfaces the parse-time error out
// of ToTypst instead of silently producing (possibly truncated) output —
// mirroring renderDialog's WalkStop handling in the HTML path
// (renderer.go:65-67).
func TestToTypst_Dialog_Err(t *testing.T) {
	input := "{start-dialog}\n@Bob:\nBadly indented line\n{end-dialog}\n"

	got, err := markdown.ToTypst([]byte(input))
	if err == nil {
		t.Fatalf("ToTypst(%q) expected a non-nil error for a badly indented dialog line, got nil (output: %q)", input, got)
	}
	wantErr := "Wrong line indentation for dialog item: Badly indented line"
	if err.Error() != wantErr {
		t.Fatalf("ToTypst(%q) error = %q, want %q", input, err.Error(), wantErr)
	}
	if len(got) != 0 {
		t.Fatalf("ToTypst(%q) expected no output alongside the error, got: %q", input, got)
	}
}

// TestToTypst_Parallel_Golden covers the #parallel(...) custom-block mapping
// with the NEW (post-ASR-1/ASR-3) semantics:
//   - Parameter renamed: `source-dir:` (was `secondary-dir:`).
//   - Row dict keys: `source:` (was `main:`), `translation:` (was `secondary:`),
//     plus optional `transcription:` when TranscriptionRaw != "".
//   - Every "---" is a field separator (ASR-3): two "---" lines → 3 fields.
//   - Empty `translation: []` is emitted when TranslationRaw=="" (unchanged).
func TestToTypst_Parallel_Golden(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			// Was: "LAST '---' split in main cell, plus a row with empty secondary"
			// Now (ASR-3): two "---" lines split into 3 fields:
			//   source="First para.", translation="Second para in main.",
			//   transcription="Secondary cell."
			// Row 2 is source-only → translation: [] emitted (no transcription key).
			name: "three-field row then source-only row (ASR-3 new split behavior)",
			input: "{start-parallel}\n" +
				"First para.\n" +
				"\n" +
				"---\n" +
				"\n" +
				"Second para in main.\n" +
				"---\n" +
				"Secondary cell.\n" +
				"===\n" +
				"Row2 main only.\n" +
				"{end-parallel}\n",
			want: "#parallel(source-dir: ltr, script: \"\",\n" +
				"  (source: [First para\\.\n\n], translation: [Second para in main\\.\n\n], transcription: [Secondary cell\\.\n\n]),\n" +
				"  (source: [Row2 main only\\.\n\n], translation: []),\n" +
				")\n\n",
		},
		{
			// script=arab: the SOURCE column now gets source-dir: rtl (ASR-1 reversal —
			// the old behavior drove the SECONDARY column's direction from the marker).
			name: "script=arab emits source-dir: rtl (ASR-1 reversal)",
			input: "{start-parallel script=arab}\n" +
				"Main.\n" +
				"---\n" +
				"Secondary.\n" +
				"{end-parallel}\n",
			want: "#parallel(source-dir: rtl, script: \"arab\",\n" +
				"  (source: [Main\\.\n\n], translation: [Secondary\\.\n\n]),\n" +
				")\n\n",
		},
	})
}

// ---------------------------------------------------------------------
// SPECS §12.3 — Typst emission (Go-side dict-string assertions)
// The Typst compile gate lives in pkg/ebook/typst_gate_test.go (already done).
// These tests verify the Go-side string emission only.
// ---------------------------------------------------------------------

// TestToTypst_Parallel_TwoFields asserts the exact Go-side Typst emission for a
// two-field parallel block (source + translation, NO transcription key):
//   - `source-dir:` param (not `secondary-dir:`)
//   - row dict: `(source: [...], translation: [...])`
//   - `transcription:` key is genuinely ABSENT (key omission, not empty content)
func TestToTypst_Parallel_TwoFields(t *testing.T) {
	input := "{start-parallel script=latn}\nsource text\n---\ntranslation text\n{end-parallel}\n"
	want := "#parallel(source-dir: ltr, script: \"latn\",\n" +
		"  (source: [source text\n\n], translation: [translation text\n\n]),\n" +
		")\n\n"

	got, err := markdown.ToTypst([]byte(input))
	if err != nil {
		t.Fatalf("ToTypst() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToTypst() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// Assert the param rename explicitly: old name must not appear.
	if strings.Contains(string(got), "secondary-dir:") {
		t.Error("must not contain old param name 'secondary-dir:' (renamed to 'source-dir:', ASR-1)")
	}
	// Assert transcription key is genuinely absent (not just empty).
	// book.typ detects transcription presence via "transcription" in r (dict-key membership).
	if strings.Contains(string(got), "transcription:") {
		t.Error("two-field row must have NO 'transcription:' key in the dict (key omission, not empty content)")
	}
}

// TestToTypst_Parallel_ThreeFields asserts the Go-side Typst emission for a
// three-field parallel block: the `transcription:` key is present in the row dict,
// using the pinned script="latn" resolution at Typst runtime (ASR-6).
func TestToTypst_Parallel_ThreeFields(t *testing.T) {
	input := "{start-parallel script=latn}\nsource\n---\ntranslation\n---\ntranscription\n{end-parallel}\n"
	want := "#parallel(source-dir: ltr, script: \"latn\",\n" +
		"  (source: [source\n\n], translation: [translation\n\n], transcription: [transcription\n\n]),\n" +
		")\n\n"

	got, err := markdown.ToTypst([]byte(input))
	if err != nil {
		t.Fatalf("ToTypst() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToTypst() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// transcription key must be present (book.typ: "transcription" in r).
	if !strings.Contains(string(got), "transcription:") {
		t.Error("three-field row must have 'transcription:' key in the dict")
	}
}

// TestToTypst_Parallel_RTLSource asserts that script=arab produces `source-dir: rtl`
// (the source direction follows the marker, ASR-4). Assert the old param name is absent.
func TestToTypst_Parallel_RTLSource(t *testing.T) {
	input := "{start-parallel script=arab}\nsource\n---\ntranslation\n{end-parallel}\n"
	want := "#parallel(source-dir: rtl, script: \"arab\",\n" +
		"  (source: [source\n\n], translation: [translation\n\n]),\n" +
		")\n\n"

	got, err := markdown.ToTypst([]byte(input))
	if err != nil {
		t.Fatalf("ToTypst() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToTypst() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// The param rename must be complete everywhere.
	if strings.Contains(string(got), "secondary-dir:") {
		t.Error("must not contain old param name 'secondary-dir:' anywhere in output")
	}
}

// TestToTypst_EscapeTypstMarkup covers escapeTypstMarkup's full metachar
// set (SPECS §5.2) indirectly through ToTypst: every character in
// `\ # [ ] * _ ` $ < > @ ~ = + - / .` must come out backslash-escaped.
// The very first token in the input is a literal, UNCONSUMED backslash
// (CommonMark only treats `\` as an escape when followed by ASCII
// punctuation; a trailing space is not punctuation, so the backslash
// survives into the Text node's value both before and after
// unescapeMarkdownBackslash) — so it is itself a metachar requiring
// protection, which is why it comes out doubled (`\\`) rather than
// singled: escapeTypstMarkup escapes that one real backslash into two.
func TestToTypst_EscapeTypstMarkup(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "every markup metachar, including a literal input backslash",
			input: "\\ # [ ] * _ ` $ < > @ ~ = + - / .\n",
			want:  "\\\\ \\# \\[ \\] \\* \\_ \\` \\$ \\< \\> \\@ \\~ \\= \\+ \\- \\/ \\.\n\n",
		},
		{
			name:  "a markdown backslash-escaped backslash still renders as one literal backslash",
			input: "a\\\\b\n",
			want:  "a\\\\b\n\n",
		},
	})
}

// TestToTypst_EscapeTypstMarkup_LineStartEscapedOrdinal is the SPECS §5.2
// review-fix regression case: `1\. text` must come out with the `.`
// escaped, or Typst would parse the resulting line-leading "1." as an
// enum marker (empirically confirmed against Typst 0.15.1 — see the
// package-level notes in typst_escape.go for the underlying goldmark
// AST discrepancy this depends on: the backslash is NOT already stripped
// out of ast.Text.Value by the parser).
func TestToTypst_EscapeTypstMarkup_LineStartEscapedOrdinal(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "escaped ordinal at line start stays plain text",
			input: "1\\. text\n",
			want:  "1\\. text\n\n",
		},
	})
}

// TestToTypst_EscapeTypstString covers escapeTypstString's `"` and `\`
// handling (SPECS §5.2) indirectly, via a vocabulary phrase field (a
// Typst string-literal context, per SPECS §4's Vocabulary row) — chosen
// because, unlike markup context, string context must NOT also escape
// markup metachars such as `/` or `.` (SPECS §5.1.3).
func TestToTypst_EscapeTypstString(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "quote and backslash escaped, markup metachars (/ .) are NOT",
			input: "{start-vocabulary}\na\"b\\c/d.e\n{end-vocabulary}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n" +
				"#vocabulary(dir: ltr, script: \"\",\n" +
				"  (phrase: \"a\\\"b\\\\c/d.e\", grammar: \"\", transcription: \"\", translation: \"\"),\n" +
				")\n\n",
		},
	})
}

// TestToTypst_TypographerEntities covers all 9 unique entity->Unicode
// mappings (typographerEntities, typst_escape.go) that the Typographer
// extension can produce (Apostrophe and RightSingleQuote share the same
// "&rsquo;" default, so a single apostrophe case exercises both).
func TestToTypst_TypographerEntities(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "left/right single and double quotes, en/em dash, ellipsis, angle quotes",
			input: "'quoted' and \"double\" and word -- word and word --- word and word ... and <<x>> done\n",
			want:  "‘quoted’ and “double” and word – word and word — word and word … and «x» done\n\n",
		},
		{
			name:  "apostrophe (shares &rsquo; with RightSingleQuote)",
			input: "it's fine\n",
			want:  "it’s fine\n\n",
		},
	})
}

// TestToTypst_Models_Golden covers the #models(...) custom-block mapping
// (like #vocabulary but minus the `grammar` field), including empty
// (dropped) fields all rendering as empty string literals.
func TestToTypst_Models_Golden(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "phrase only, other fields empty",
			input: "{start-models}\n你好\n{end-models}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"M\")]\n\n" +
				"#models(dir: ltr, script: \"\",\n" +
				"  (phrase: \"你好\", transcription: \"\", translation: \"\"),\n" +
				")\n\n",
		},
		{
			name:  "phrase, transcription and translation",
			input: "{start-models}\nrun [rʌn] = biec\n{end-models}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"M\")]\n\n" +
				"#models(dir: ltr, script: \"\",\n" +
				"  (phrase: \"run\", transcription: \"rʌn\", translation: \"biec\"),\n" +
				")\n\n",
		},
	})
}

// TestToTypst_Questions_Golden covers the #questions(...) custom-block
// mapping: a question-only line emits an empty `answer` string literal.
func TestToTypst_Questions_Golden(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "question only, no answer",
			input: "{start-questions}\nWhat is your name?\n{end-questions}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"Q\")]\n\n" +
				"#questions(dir: ltr, script: \"\", role: \"source\",\n" +
				"  (question: \"What is your name?\", answer: \"\"),\n" +
				")\n\n",
		},
		{
			name:  "question and answer",
			input: "{start-questions}\nWhere are you from? = Poland\n{end-questions}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"Q\")]\n\n" +
				"#questions(dir: ltr, script: \"\", role: \"source\",\n" +
				"  (question: \"Where are you from?\", answer: \"Poland\"),\n" +
				")\n\n",
		},
	})
}

// ---------------------------------------------------------------------
// SPECS §12.3 — Typst snapshot (Go-side dict-string assertions only)
// The optional Typst compile gate lives in pkg/ebook/typst_gate_test.go.
// ---------------------------------------------------------------------

// TestRenderVocabularyHeaderTypst asserts that a vocabulary block header item
// emits the exact dict `(kind: "header", level: N, text: "…")` in the Typst
// output, while surrounding data items keep their existing dict shape with no
// `kind` key (SPECS §12.3, §7.1, ASR-3).
func TestRenderVocabularyHeaderTypst(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "header mid-block: surrounding data items keep existing shape",
			input: "{start-vocabulary}\nphrase1 = t1\n## Heading\nphrase2 = t2\n{end-vocabulary}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n" +
				"#vocabulary(dir: ltr, script: \"\",\n" +
				"  (phrase: \"phrase1\", grammar: \"\", transcription: \"\", translation: \"t1\"),\n" +
				"  (kind: \"header\", level: 2, text: \"Heading\"),\n" +
				"  (phrase: \"phrase2\", grammar: \"\", transcription: \"\", translation: \"t2\"),\n" +
				")\n\n",
		},
	})
}

// TestRenderDialogNoteTypst asserts that a dialog note item emits the exact
// dict `(kind: "note", text: "…")` in the Typst output, and that the
// surrounding dialog turn emits its existing dict shape with no `kind` key
// (SPECS §12.3, §7.1, ASR-3).
func TestRenderDialogNoteTypst(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "note after an anonymous turn: turn keeps existing shape",
			input: "{start-dialog}\n--:\n  Hello.\n(Note text)\n{end-dialog}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"D\")]\n\n" +
				"#dialog(dir: ltr, script: \"\", role: \"source\",\n" +
				"  (header: \"—\", content: [Hello\\.\n\n]),\n" +
				"  (kind: \"note\", text: \"Note text\"),\n" +
				")\n\n",
		},
	})
}

// TestRenderTypstDataOnlyUnchanged is the ASR-3 byte-identical regression
// control for the Typst renderer: a data-only block must emit NO `kind` key in
// any dict entry and produce output identical to the pre-change golden (SPECS
// §12.3, ASR-3).
func TestRenderTypstDataOnlyUnchanged(t *testing.T) {
	runTypstGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			// Mirrors TestToTypst_Vocabulary_Golden "phrase only" — byte-identical
			// to the pre-change golden (no `kind` key must appear).
			name:  "vocabulary data-only: no kind key in dict",
			input: "{start-vocabulary}\n再见\n{end-vocabulary}\n",
			want: "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n" +
				"#vocabulary(dir: ltr, script: \"\",\n" +
				"  (phrase: \"再见\", grammar: \"\", transcription: \"\", translation: \"\"),\n" +
				")\n\n",
		},
	})
}

// TestFileToTypst_ConvertsTempFile mirrors
// TestFileToHTML_ConvertsTempFile: writes a markdown file into an
// isolated t.TempDir() and asserts FileToTypst reads and converts it.
func TestFileToTypst_ConvertsTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")

	content := "# Title\n\nSome **bold** text.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp markdown file: %v", err)
	}

	got, err := markdown.FileToTypst(path)
	if err != nil {
		t.Fatalf("FileToTypst() unexpected error: %v", err)
	}
	want := "= Title\n\nSome #strong[bold] text\\.\n\n"
	if got != want {
		t.Fatalf("FileToTypst() = %q, want %q", got, want)
	}
}

// TestFileToTypst_NonExistentPath_ReturnsError mirrors
// TestFileToHTML_NonExistentPath_ReturnsError.
func TestFileToTypst_NonExistentPath_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.md")

	got, err := markdown.FileToTypst(path)
	if err == nil {
		t.Fatalf("FileToTypst(%q) expected a non-nil error, got nil (output: %q)", path, got)
	}
	if got != "" {
		t.Fatalf("FileToTypst(%q) expected empty output alongside the error, got: %q", path, got)
	}
}
