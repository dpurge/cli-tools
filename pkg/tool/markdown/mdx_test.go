package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// runMdxGolden is the shared table-driven harness used by every
// TestToMDX_* function below: each case's markdown input must produce the
// EXACT (byte-identical) MDX output for a fixed lang="lat"/script="latn"
// pair, matching the determinism requirement (ASR-3) and "matches SPECS
// §4/§5 exactly" (AC2). Only the public markdown.ToMDX API is exercised —
// mirrors typst_test.go's runTypstGolden harness for the existing Typst
// suite.
func runMdxGolden(t *testing.T, tests []struct {
	name  string
	input string
	want  string
}) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToMDX([]byte(tc.input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX(%q) unexpected error: %v", tc.input, err)
			}
			if string(got) != tc.want {
				t.Fatalf("ToMDX(%q) mismatch\n got: %q\nwant: %q", tc.input, string(got), tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// §4.1 standard node coverage
// ---------------------------------------------------------------------

// TestToMDX_Headings covers SPECS §4.1's Heading row ("#"xLevel + " " +
// inline + "\n\n") and confirms a heading-only document (no trailing
// prose) never gets a dangling empty <Text> — flush is a no-op when no
// prose has been accumulated.
func TestToMDX_Headings(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "levels 1 through 3, no trailing prose",
			input: "# H1\n\n## H2\n\n### H3\n",
			want:  "# H1\n\n## H2\n\n### H3\n",
		},
		{
			name:  "heading only, no body at all",
			input: "### H3\n",
			want:  "### H3\n",
		},
		{
			name:  "leading H1 is kept in the body (not stripped as a title)",
			input: "# H\n\nhi",
			want:  "# H\n\n<Text lang=\"lat\" script=\"latn\">\n\nhi\n\n</Text>\n",
		},
	})
}

// TestToMDX_ParagraphAndBreaks covers Paragraph's "\n\n" terminator and
// Text's break handling (D2): a soft break emits a literal "\n" (preserves
// the authored per-line layout) and a hard break (either a trailing
// backslash or CommonMark's two-trailing-spaces form) emits "  \n".
func TestToMDX_ParagraphAndBreaks(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain paragraph, no heading",
			input: "just prose\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\njust prose\n\n</Text>\n",
		},
		{
			name:  "soft break becomes a literal newline",
			input: "line one\nline two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\nline two\n\n</Text>\n",
		},
		{
			name:  "hard break via two trailing spaces",
			input: "line one  \nline two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one  \nline two\n\n</Text>\n",
		},
		{
			name:  "hard break via trailing backslash",
			input: "line one\\\nline two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one  \nline two\n\n</Text>\n",
		},
	})
}

// TestToMDX_Emphasis covers the canonical "*"/"**" markup delimiters
// (Level 1 -> "*", Level 2 -> "**", SPECS §4.1).
func TestToMDX_Emphasis(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "level 1 (em) and level 2 (strong)",
			input: "*em* and **strong**\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n*em* and **strong**\n\n</Text>\n",
		},
	})
}

// TestToMDX_CodeSpan covers inline code-span re-emission, including the
// backtick-delimiter widening rule (one more backtick than the longest
// run already in the content) and the trailing-newline-becomes-space
// rule for a span crossing a soft line break (mirrors
// typst_render.go's renderCodeSpanTypst).
func TestToMDX_CodeSpan(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line, no widening needed",
			input: "`code`\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n`code`\n\n</Text>\n",
		},
		{
			name:  "spans a line break, collapses to a space",
			input: "`a\nb`\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n`a b`\n\n</Text>\n",
		},
		{
			name:  "content contains a backtick run, delimiter widens past it",
			input: "`` a ``` b `` \n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n````a ``` b````\n\n</Text>\n",
		},
	})
}

// TestToMDX_Link covers "[text](escDest [\"escTitle\"])" (SPECS §4.1),
// including escapeMdxUrl wrapping a destination that (via the source's
// own CommonMark angle-bracket destination form) contains a raw space.
func TestToMDX_Link(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic link with title",
			input: "[text](https://example.com/a?b=1 \"a title\")\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n[text](https://example.com/a?b=1 \"a title\")\n\n</Text>\n",
		},
		{
			name:  "destination with a raw space round-trips via escapeMdxUrl's angle-bracket wrap",
			input: "[a](</has space>)\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n[a](</has space>)\n\n</Text>\n",
		},
	})
}

// TestToMDX_AutoLink covers URL and email autolinks re-emitted as a
// normal "[label](dest)" link (SPECS §4.1: NOT CommonMark's own "<url>"
// form, which would reintroduce the exact "<" JSX-element hazard
// escapeMdxText otherwise guards against); an email autolink gets a
// "mailto:" prefix.
func TestToMDX_AutoLink(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL and email autolinks",
			input: "<https://example.com>\n<mail@example.com>\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\n" +
				"[https://example.com](https://example.com)\n" +
				"[mail@example.com](mailto:mail@example.com)\n\n</Text>\n",
		},
	})
}

// TestToMDX_Image covers "![escAlt](escDest)" (SPECS §4.1): unlike the
// Typst renderer (which drops alt text), MDX keeps it, escaped like any
// other prose text.
func TestToMDX_Image(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "image, alt kept and escaped",
			input: "![a {b} c](pic.png)\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n![a \\{b\\} c](pic.png)\n\n</Text>\n",
		},
	})
}

// TestToMDX_List covers "- "/"N. " markers (SPECS §4.1), including a
// nested list correctly indented under its parent item (renderListItem's
// buffer-then-indent mechanism, mdx_render.go).
func TestToMDX_List(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unordered (tight)",
			input: "- one\n- two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n- one\n- two\n\n</Text>\n",
		},
		{
			name:  "ordered (tight)",
			input: "1. one\n2. two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n1. one\n2. two\n\n</Text>\n",
		},
		{
			name:  "nested list inside a list item, indented under it",
			input: "- one\n  - nested\n- two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n- one\n  - nested\n- two\n\n</Text>\n",
		},
	})
}

// TestToMDX_Blockquote covers the "> " line-prefix mechanism (SPECS
// §4.1): multi-paragraph content, and a nested block (a List, and a
// nested ThematicBreak) inside the quote.
func TestToMDX_Blockquote(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single paragraph",
			input: "> quoted text\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n> quoted text\n\n</Text>\n",
		},
		{
			name:  "multi-paragraph, blank quoted line is a bare '>'",
			input: "> para1\n>\n> para2\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n> para1\n>\n> para2\n\n</Text>\n",
		},
		{
			name:  "nested list inside a blockquote",
			input: "> - one\n> - two\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n> - one\n> - two\n\n</Text>\n",
		},
		{
			name:  "nested thematic break inside a blockquote renders as '---' (only top-level is dropped)",
			input: "> a\n>\n> ---\n>\n> b\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n> a\n>\n> ---\n>\n> b\n\n</Text>\n",
		},
	})
}

// TestToMDX_CodeBlocks covers fenced code (with/without an info string)
// and indented code, all re-emitted as fenced blocks (SPECS §4.1); content
// is never escaped (it is code, not prose).
func TestToMDX_CodeBlocks(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "fenced with language",
			input: "```go\nfmt.Println(1)\n```\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n```go\nfmt.Println(1)\n```\n\n</Text>\n",
		},
		{
			name:  "indented code block, no language token",
			input: "    x := 1\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n```\nx := 1\n```\n\n</Text>\n",
		},
	})
}

// TestToMDX_ThematicBreak_TopLevelDropped covers Decision D3: a top-level
// thematic break is a group boundary that flushes the accumulated prose
// into its own <Text> and is itself never rendered — it is the ebook's
// own vocabulary/reading separator, and no phraseforge lesson places an
// <hr> between blocks.
func TestToMDX_ThematicBreak_TopLevelDropped(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "splits surrounding prose into two Text groups, '---' itself is dropped",
			input: "hi\n\n---\n\nbye\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\nhi\n\n</Text>\n\n" +
				"<Text lang=\"lat\" script=\"latn\">\n\nbye\n\n</Text>\n",
		},
	})
}

// TestToMDX_Table covers the GFM pipe-table re-emission (SPECS §4.1): the
// header row, the alignment delimiter row (built from the correct source
// — see the GOLDMARK API DISCREPANCY note on renderTableHeader,
// mdx_render.go, since TableHeader.Alignments itself is never populated
// by goldmark), and the body row.
func TestToMDX_Table(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "left/center/right alignment",
			input: "| L | C | R |\n|:--|:-:|--:|\n| a | b | c |\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\n" +
				"| L | C | R |\n| :-- | :-: | --: |\n| a | b | c |\n\n</Text>\n",
		},
		{
			name:  "no alignment marker maps to a plain '---'",
			input: "| A | B |\n|---|---|\n| 1 | 2 |\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\n" +
				"| A | B |\n| --- | --- |\n| 1 | 2 |\n\n</Text>\n",
		},
	})
}

// TestToMDX_Strikethrough covers "~~"children"~~".
func TestToMDX_Strikethrough(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{name: "struck text", input: "~~gone~~\n", want: "<Text lang=\"lat\" script=\"latn\">\n\n~~gone~~\n\n</Text>\n"},
	})
}

// TestToMDX_DefinitionList covers the common one-term/one-description
// case ("Term\n: Description\n\n", PHP-Markdown-Extra syntax, SPECS
// §4.1); multi-term/multi-description pairings are a documented,
// deliberately out-of-scope simplification (mdx_render.go's
// renderDefinitionList comment), mirroring the Typst suite's identical
// caveat.
func TestToMDX_DefinitionList(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "one term, one description",
			input: "Term\n: Definition text.\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nTerm\n: Definition text.\n\n</Text>\n",
		},
	})
}

// TestToMDX_RawHTML_EmitsNothing covers the SPECS §4.1 fallback row for
// RawHTML/HTMLBlock: raw HTML is never registered, so it disappears
// (children walked, but there ARE none) while sibling text is unaffected —
// mirroring the HTML renderer's Unsafe=false omission, and critically
// (SPECS §5.1 point 2/ASR-4) never passing a raw "<...>" through into
// MDX/JSX unescaped.
func TestToMDX_RawHTML_EmitsNothing(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "inline tag dropped, sibling text kept",
			input: "Some <b>raw html</b> inline.\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nSome raw html inline.\n\n</Text>\n",
		},
		{
			name:  "html block dropped entirely",
			input: "Before.\n\n<div>block html</div>\n\nAfter.\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nBefore.\n\nAfter.\n\n</Text>\n",
		},
	})
}

// TestToMDX_TypographerEntities covers all 9 unique entity->Unicode
// mappings the Typographer extension can produce (typographerEntities,
// typst_escape.go, reused as-is per SPECS §3.2/ASR-2).
func TestToMDX_TypographerEntities(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "left/right single and double quotes, en/em dash, ellipsis, angle quotes",
			input: "'quoted' and \"double\" and word -- word and word --- word and word ... and <<x>> done\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\n" +
				"‘quoted’ and “double” and word – word and word — word and word … and «x» done\n\n</Text>\n",
		},
	})
}

// ---------------------------------------------------------------------
// §3.1/§7.5 <Text> grouping (two-level orchestration)
// ---------------------------------------------------------------------

// TestToMDX_TextGrouping covers AC3: a contiguous run of top-level prose
// blocks yields exactly ONE <Text>; a Heading, custom block, or top-level
// ThematicBreak is a group boundary; a leading H1 is kept in the body; a
// document with prose before its first heading gets its own leading
// <Text>; a prose-only document (no heading at all) is a single <Text>.
func TestToMDX_TextGrouping(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "contiguous paragraphs group into exactly one Text",
			input: "Just one paragraph.\n\nAnother paragraph.\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nJust one paragraph.\n\nAnother paragraph.\n\n</Text>\n",
		},
		{
			name:  "prose before the first heading gets its own Text",
			input: "Intro para.\n\n# Heading\n\nBody.\n",
			want: "<Text lang=\"lat\" script=\"latn\">\n\nIntro para.\n\n</Text>\n\n" +
				"# Heading\n\n<Text lang=\"lat\" script=\"latn\">\n\nBody.\n\n</Text>\n",
		},
		{
			name:  "heading, then a '---' boundary splits two prose runs into two Texts",
			input: "# H\n\nP1.\n\nP2.\n\n---\n\nP3.\n",
			want: "# H\n\n<Text lang=\"lat\" script=\"latn\">\n\nP1.\n\nP2.\n\n</Text>\n\n" +
				"<Text lang=\"lat\" script=\"latn\">\n\nP3.\n\n</Text>\n",
		},
		{
			name:  "a custom block splits surrounding prose into separate Texts",
			input: "# Title\n\nSome prose here.\n\n{start-vocabulary}\nx = y\n{end-vocabulary}\n\nMore prose.\n",
			want: "# Title\n\n<Text lang=\"lat\" script=\"latn\">\n\nSome prose here.\n\n</Text>\n\n" +
				"```vocabulary lang=lat script=latn\nx = y\n```\n\n" +
				"<Text lang=\"lat\" script=\"latn\">\n\nMore prose.\n\n</Text>\n",
		},
	})
}

// ---------------------------------------------------------------------
// §4.2-§4.4 custom-block fences
// ---------------------------------------------------------------------

// TestToMDX_Vocabulary_Golden covers the vocabulary fence (SPECS §4.2):
// full-field lines, and every permutation of an omitted field.
func TestToMDX_Vocabulary_Golden(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full: phrase, grammar, transcription, translation",
			input: "{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\n你好 {noun} [nǐ hǎo] = hello\n```\n",
		},
		{
			name:  "no grammar, no transcription: 'p = t'",
			input: "{start-vocabulary}\nrana, -ae {N f} = żaba\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\nrana, -ae {N f} = żaba\n```\n",
		},
		{
			name:  "phrase and translation only, no grammar/transcription",
			input: "{start-vocabulary}\np = t\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\np = t\n```\n",
		},
		{
			name:  "phrase, transcription, translation, no grammar: 'p [x] = t'",
			input: "{start-vocabulary}\np [x] = t\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\np [x] = t\n```\n",
		},
		{
			name:  "all four fields: 'p {g} [x] = t'",
			input: "{start-vocabulary}\np {g} [x] = t\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\np {g} [x] = t\n```\n",
		},
		{
			name:  "phrase only, every other field empty",
			input: "{start-vocabulary}\n再见\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\n再见\n```\n",
		},
		{
			name:  "multiple items, one per line",
			input: "{start-vocabulary}\nrana, -ae {N f} = żaba\naqua, -ae {N f} = woda\n{end-vocabulary}\n",
			want:  "```vocabulary lang=lat script=latn\nrana, -ae {N f} = żaba\naqua, -ae {N f} = woda\n```\n",
		},
	})
}

// TestToMDX_Dialog_Golden covers the dialog fence (SPECS §4.3): an
// anonymous "--:" turn and a named "@Name:" turn, 2-space content
// re-indentation, a multi-line turn with an internal blank line, and that
// fence content is literal (a "**bold**" run inside a turn is emitted
// verbatim, not re-rendered).
func TestToMDX_Dialog_Golden(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "anonymous and named turns, content is literal (not re-rendered)",
			input: "{start-dialog}\n" +
				"--:\n" +
				"  Hello **there**.\n" +
				"@Bob:\n" +
				"  Hi!\n" +
				"{end-dialog}\n",
			want: "```dialog lang=lat script=latn\n" +
				"--:\n" +
				"  Hello **there**.\n" +
				"@Bob:\n" +
				"  Hi!\n" +
				"```\n",
		},
		{
			name: "multi-line turn with an internal blank line",
			input: "{start-dialog}\n" +
				"@Bob:\n" +
				"  Line1\n" +
				"\n" +
				"  Line2\n" +
				"{end-dialog}\n",
			want: "```dialog lang=lat script=latn\n" +
				"@Bob:\n" +
				"  Line1\n" +
				"\n" +
				"  Line2\n" +
				"```\n",
		},
	})
}

// TestToMDX_Dialog_Err asserts that a Dialog block with a bad-indentation
// content line (Dialog.Err, ast.go) surfaces the parse-time error out of
// ToMDX instead of silently producing output — mirroring renderDialog's
// WalkStop handling in the HTML/Typst paths (renderer.go, typst_render.go).
func TestToMDX_Dialog_Err(t *testing.T) {
	input := "{start-dialog}\n@Bob:\nBadly indented line\n{end-dialog}\n"

	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err == nil {
		t.Fatalf("ToMDX(%q) expected a non-nil error for a badly indented dialog line, got nil (output: %q)", input, got)
	}
	wantErr := "Wrong line indentation for dialog item: Badly indented line"
	if err.Error() != wantErr {
		t.Fatalf("ToMDX(%q) error = %q, want %q", input, err.Error(), wantErr)
	}
	if len(got) != 0 {
		t.Fatalf("ToMDX(%q) expected no output alongside the error, got: %q", input, got)
	}
}

// TestToMDX_Parallel_Golden covers the parallel fence with the NEW field semantics
// (ASR-1/ASR-3): every "---" is now a field separator (SplitN cap 3), so the
// fence body reconstructs each row's fields joined by "\n---\n" without any
// surrounding blank lines (TrimSpace is applied per field during parsing).
// Row 2 (source-only) emits no "---" separator at all.
func TestToMDX_Parallel_Golden(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			// Was: "LAST '---' split in main cell, plus a row with empty secondary"
			// Now (ASR-3): two "---" lines → 3 fields (source/translation/transcription),
			// each TrimSpace'd and rejoined with "\n---\n". Row 2 source-only → no "---".
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
			want: "```parallel lang=lat script=latn\n" +
				"First para.\n" +
				"---\n" +
				"Second para in main.\n" +
				"---\n" +
				"Secondary cell.\n" +
				"===\n" +
				"Row2 main only.\n" +
				"```\n",
		},
	})
}

// ---------------------------------------------------------------------
// SPECS §12.5 — MDX round-trip (parallel-specific)
// ---------------------------------------------------------------------

// TestToMDX_Parallel_OneField verifies that a source-only row round-trips as
// its source text with no "---" separator in the fence body.
func TestToMDX_Parallel_OneField(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "source only, no --- in fence body",
			input: "{start-parallel}\nsource text\n{end-parallel}\n",
			want:  "```parallel lang=lat script=latn\nsource text\n```\n",
		},
	})
}

// TestToMDX_Parallel_TwoFields verifies that a two-field row (source + translation)
// round-trips with a single "\n---\n" joining the two fields.
func TestToMDX_Parallel_TwoFields(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "source and translation joined by ---",
			input: "{start-parallel}\nsource\n---\ntranslation\n{end-parallel}\n",
			want:  "```parallel lang=lat script=latn\nsource\n---\ntranslation\n```\n",
		},
	})
}

// TestToMDX_Parallel_ThreeFields verifies that a three-field row (source +
// translation + transcription) round-trips with two "\n---\n" separators.
func TestToMDX_Parallel_ThreeFields(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "source, translation, and transcription joined by --- ---",
			input: "{start-parallel}\nsource\n---\ntranslation\n---\ntranscription\n{end-parallel}\n",
			want:  "```parallel lang=lat script=latn\nsource\n---\ntranslation\n---\ntranscription\n```\n",
		},
	})
}

// TestToMDX_Parallel_RoundTripStable proves parse → ToMDX → parse is lossless
// for all three field counts: the re-parsed fence body produces HTML identical to
// the original input's HTML, confirming the same ParallelRow fields are recovered
// (SPECS §12.5). Verified via public API only (ToMDX + ToHTML).
func TestToMDX_Parallel_RoundTripStable(t *testing.T) {
	// Step-by-step scenario: for each field count, verify that serializing to MDX
	// and re-parsing the fence body produces identical ParallelRows (evidenced by
	// byte-identical HTML output). Mirrors TestToMDX_ModelsRoundTrip pattern.
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "one field",
			input: "{start-parallel}\nSource text.\n{end-parallel}\n",
		},
		{
			name:  "two fields",
			input: "{start-parallel}\nSource text.\n---\nTranslation text.\n{end-parallel}\n",
		},
		{
			name:  "three fields",
			input: "{start-parallel}\nSource text.\n---\nTranslation text.\n---\nTranscription text.\n{end-parallel}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass 1: original input → MDX fence.
			gotMDX, err := markdown.ToMDX([]byte(tc.input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX() unexpected error: %v", err)
			}
			// Extract the fence body between the opening and closing backtick lines.
			body := strings.TrimPrefix(string(gotMDX), "```parallel lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "\n```\n")

			// Pass 2: re-wrap the body and compare HTML output.
			originalHTML, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("original ToHTML() unexpected error: %v", err)
			}
			reparsedHTML, err := markdown.ToHTML([]byte("{start-parallel}\n" + body + "\n{end-parallel}\n"))
			if err != nil {
				t.Fatalf("reparsed ToHTML() unexpected error: %v", err)
			}
			if string(reparsedHTML) != string(originalHTML) {
				t.Fatalf("round-trip mismatch for %q:\n got:  %q\nwant: %q",
					tc.name, string(reparsedHTML), string(originalHTML))
			}
		})
	}
}

// TestToMDX_Models_Golden covers the models fence (mirrors
// TestToMDX_Vocabulary_Golden, minus the `{grammar}` field): full-field
// lines and every permutation of an omitted field.
func TestToMDX_Models_Golden(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "phrase, transcription and translation",
			input: "{start-models}\nrun [rʌn] = biec\n{end-models}\n",
			want:  "```models lang=lat script=latn\nrun [rʌn] = biec\n```\n",
		},
		{
			name:  "phrase and transcription only",
			input: "{start-models}\nrun [rʌn]\n{end-models}\n",
			want:  "```models lang=lat script=latn\nrun [rʌn]\n```\n",
		},
		{
			name:  "phrase and translation only",
			input: "{start-models}\nrun = biec\n{end-models}\n",
			want:  "```models lang=lat script=latn\nrun = biec\n```\n",
		},
		{
			name:  "phrase only",
			input: "{start-models}\n再见\n{end-models}\n",
			want:  "```models lang=lat script=latn\n再见\n```\n",
		},
		{
			name: "multiple items, one per line",
			input: "{start-models}\n" +
				"run [rʌn] = biec\n" +
				"walk = iść\n" +
				"{end-models}\n",
			want: "```models lang=lat script=latn\nrun [rʌn] = biec\nwalk = iść\n```\n",
		},
	})
}

// TestToMDX_Questions_Golden covers the questions fence: a question-only
// line (no " = answer" suffix) and a question+answer line.
func TestToMDX_Questions_Golden(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "question only, no answer",
			input: "{start-questions}\nWhat is your name?\n{end-questions}\n",
			want:  "```questions lang=lat script=latn\nWhat is your name?\n```\n",
		},
		{
			name:  "question and answer",
			input: "{start-questions}\nWhere are you from? = Poland\n{end-questions}\n",
			want:  "```questions lang=lat script=latn\nWhere are you from? = Poland\n```\n",
		},
		{
			name: "mixed block, multiple items",
			input: "{start-questions}\n" +
				"Q1\n" +
				"Q2 = A2\n" +
				"{end-questions}\n",
			want: "```questions lang=lat script=latn\nQ1\nQ2 = A2\n```\n",
		},
	})
}

// TestToMDX_FenceWidening covers mdxFence widening the vocabulary/dialog/
// parallel fence delimiter past a backtick run already present in the
// (literal, unescaped) fence body (SPECS §5.1.3).
func TestToMDX_FenceWidening(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "vocabulary body containing a triple-backtick run widens the fence to 4 backticks",
			input: "{start-vocabulary}\na ``` b = c\n{end-vocabulary}\n",
			want:  "````vocabulary lang=lat script=latn\na ``` b = c\n````\n",
		},
		{
			name:  "dialog body containing a triple-backtick run widens the fence to 4 backticks",
			input: "{start-dialog}\n--:\n  a ``` b\n{end-dialog}\n",
			want:  "````dialog lang=lat script=latn\n--:\n  a ``` b\n````\n",
		},
		{
			name:  "parallel body containing a triple-backtick run widens the fence to 4 backticks",
			input: "{start-parallel}\na ``` b\n{end-parallel}\n",
			want:  "````parallel lang=lat script=latn\na ``` b\n````\n",
		},
	})
}

// ---------------------------------------------------------------------
// §5.2/§5.3 escaping
// ---------------------------------------------------------------------

// TestToMDX_EscapeMdxText covers escapeMdxText's full metachar set (SPECS
// §5.2) indirectly through ToMDX: every character in `\ { } < backtick *
// _ [ ]` comes out backslash-escaped, while the deliberately-NOT-escaped
// set (". , ! ? ( ) : ; # + - = ~ > /") passes through untouched
// mid-line.
func TestToMDX_EscapeMdxText(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "braces are build-breaking, escaped",
			input: "a {b} c\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\na \\{b\\} c\n\n</Text>\n",
		},
		{
			name:  "a bare '<' is build-breaking, escaped",
			input: "x < y\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nx \\< y\n\n</Text>\n",
		},
		{
			name:  "a source-escaped literal asterisk is re-escaped, not turned into emphasis",
			input: "\\*lit\\*\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\n\\*lit\\*\n\n</Text>\n",
		},
		{
			name:  "mid-line safe punctuation is never escaped",
			input: "a. b, c! d? e(f) g:h;i #j +k -l =m ~n >o /p\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\na. b, c! d? e(f) g:h;i #j +k -l =m ~n >o /p\n\n</Text>\n",
		},
	})
}

// TestToMDX_EscapeMdxLineStart covers escapeMdxLineStart (SPECS §5.3): a
// physical source line that does NOT interrupt the enclosing paragraph
// under CommonMark's own block-parsing rules (so it remains a literal
// Text segment reached via a soft break) but WOULD be misread as a block
// marker if emitted at a bare MDX line start, gets a defensive backslash.
// Each case is deliberately chosen so the marker character does NOT
// actually open a new block in the SOURCE (an ordinal other than 1, a
// dash/hash/plus/equals/tilde with no following space, or a
// source-escaped ">"), proving these hazards are real: without the
// escape, a line like "2. x" would be misread as an ordinal list item by
// a downstream MDX/remark parser even though the ORIGINAL author never
// intended one.
func TestToMDX_EscapeMdxLineStart(t *testing.T) {
	runMdxGolden(t, []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ordinal other than 1 cannot interrupt the paragraph, stays literal text needing escape",
			input: "line one\n2. x\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n2\\. x\n\n</Text>\n",
		},
		{
			name:  "dash with no following space is not a bullet, stays literal text needing escape",
			input: "line one\n-foo\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\-foo\n\n</Text>\n",
		},
		{
			name:  "a source-escaped '>' stays literal text (never opened a blockquote) and round-trips",
			input: "line one\n\\> escaped quote\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\> escaped quote\n\n</Text>\n",
		},
		{
			name:  "hash with no following space is not an ATX heading, stays literal text needing escape",
			input: "line one\n#foo\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\#foo\n\n</Text>\n",
		},
		{
			name:  "single tilde (not a 3+ run) is not a code fence, stays literal text needing escape",
			input: "line one\n~not a fence\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\~not a fence\n\n</Text>\n",
		},
		{
			name:  "plus with no following space is not a bullet, stays literal text needing escape",
			input: "line one\n+foo\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\+foo\n\n</Text>\n",
		},
		{
			name:  "leading '=' (not a setext underline on its own line here) needing escape",
			input: "line one\n=foo\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n\\=foo\n\n</Text>\n",
		},
		{
			name:  "ordinal followed by ')' also cannot interrupt the paragraph, stays literal text needing escape",
			input: "line one\n2) foo\n",
			want:  "<Text lang=\"lat\" script=\"latn\">\n\nline one\n2\\) foo\n\n</Text>\n",
		},
	})
}

// TestToMDX_EscapeMdxAttr covers escapeMdxAttr (SPECS §5.5): lang/script
// passed into the "<Text lang=\"...\" script=\"...\">" JSX attribute
// values are double-quote-escaped ('\' -> '\\', '"' -> '\"'), the same
// minimal rule SPECS §5.4 uses for YAML frontmatter scalars.
func TestToMDX_EscapeMdxAttr(t *testing.T) {
	got, err := markdown.ToMDX([]byte("hi\n"), `la"t`, `lat\n`)
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	want := "<Text lang=\"la\\\"t\" script=\"lat\\\\n\">\n\nhi\n\n</Text>\n"
	if string(got) != want {
		t.Fatalf("ToMDX() = %q, want %q", string(got), want)
	}
}

// TestToMDX_Dialog_Err_PropagatesThroughBlockquote proves a Dialog.Err
// still surfaces out of ToMDX even when the Dialog block is nested inside
// a Blockquote — i.e. reached via renderChildrenToBuf's own recursive
// r.self.Render call (mdx_render.go) rather than the top-level
// orchestration's direct r.Render call (mdx.go) — confirming the error
// propagates correctly across that extra recursion layer instead of being
// silently swallowed.
func TestToMDX_Dialog_Err_PropagatesThroughBlockquote(t *testing.T) {
	input := "> {start-dialog}\n> @Bob:\n> Badly indented\n> {end-dialog}\n"

	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err == nil {
		t.Fatalf("ToMDX(%q) expected a non-nil error, got nil (output: %q)", input, got)
	}
	wantErr := "Wrong line indentation for dialog item: Badly indented"
	if err.Error() != wantErr {
		t.Fatalf("ToMDX(%q) error = %q, want %q", input, err.Error(), wantErr)
	}
	if len(got) != 0 {
		t.Fatalf("ToMDX(%q) expected no output alongside the error, got: %q", input, got)
	}
}

// ---------------------------------------------------------------------
// §6 Title
// ---------------------------------------------------------------------

func TestTitle(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		hasErr bool
	}{
		{name: "H1 present", input: "# H\n\nbody\n", want: "H"},
		{name: "H1 absent, only an H2", input: "## H2 only\n\nbody\n", want: ""},
		{name: "no heading at all", input: "no heading here\n", want: ""},
		{name: "H1 with emphasis, markup dropped from the plain title", input: "# *X*\n\nbody\n", want: "X"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.Title([]byte(tc.input))
			if (err != nil) != tc.hasErr {
				t.Fatalf("Title(%q) error = %v, hasErr = %v", tc.input, err, tc.hasErr)
			}
			if got != tc.want {
				t.Fatalf("Title(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// FileToMDX (mirrors filetohtml_test.go / typst_test.go's FileToTypst tests)
// ---------------------------------------------------------------------

func TestFileToMDX_ConvertsTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")

	content := "# Title\n\nSome **bold** text.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp markdown file: %v", err)
	}

	got, err := markdown.FileToMDX(path, "lat", "latn")
	if err != nil {
		t.Fatalf("FileToMDX() unexpected error: %v", err)
	}
	want := "# Title\n\n<Text lang=\"lat\" script=\"latn\">\n\nSome **bold** text.\n\n</Text>\n"
	if got != want {
		t.Fatalf("FileToMDX() = %q, want %q", got, want)
	}
}

func TestFileToMDX_NonExistentPath_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.md")

	got, err := markdown.FileToMDX(path, "lat", "latn")
	if err == nil {
		t.Fatalf("FileToMDX(%q) expected a non-nil error, got nil (output: %q)", path, got)
	}
	if got != "" {
		t.Fatalf("FileToMDX(%q) expected empty output alongside the error, got: %q", path, got)
	}
}

// ---------------------------------------------------------------------
// §10 round-trip assertions: emitted vocabulary/dialog fence bodies,
// re-parsed by a TEST PORT of phraseforge-web's own parsing rules, yield
// the original VocabularyItem/DialogItem fields (SPECS §4.2/§4.3).
// ---------------------------------------------------------------------

// portParseStructuredEntry is a Go test port of phraseforge-web's
// src/components/Lesson/StructuredBody.tsx parseStructuredEntry: split
// the line at the first "="  (trimming one optional space on each side,
// mirroring the TS regex /\s=\s|=/), extract the FIRST "{...}" as
// grammar, extract the LAST "[...]" as transcription, and collapse
// whatever remains (whitespace-normalized) to source/phrase.
func portParseStructuredEntry(line string) (source, grammar, transcription, translation string) {
	left := line
	if i := strings.Index(line, "="); i != -1 {
		left = strings.TrimSpace(line[:i])
		translation = strings.TrimSpace(line[i+1:])
	}
	source = left

	if gi := strings.Index(source, "{"); gi != -1 {
		if gj := strings.Index(source[gi:], "}"); gj != -1 {
			grammar = strings.TrimSpace(source[gi+1 : gi+gj])
			source = source[:gi] + " " + source[gi+gj+1:]
		}
	}
	if ti := strings.LastIndex(source, "["); ti != -1 {
		if tj := strings.Index(source[ti:], "]"); tj != -1 {
			transcription = strings.TrimSpace(source[ti+1 : ti+tj])
			source = source[:ti] + source[ti+tj+1:]
		}
	}
	source = strings.Join(strings.Fields(source), " ")
	if source == "" {
		source = left
	}
	return source, grammar, transcription, translation
}

func TestToMDX_VocabularyRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		item struct{ phrase, grammar, transcription, translation string }
	}{
		{name: "all four fields", item: struct{ phrase, grammar, transcription, translation string }{"rana, -ae", "N f", "", "żaba"}},
		{name: "phrase and transcription and translation", item: struct{ phrase, grammar, transcription, translation string }{"你好", "noun", "nǐ hǎo", "hello"}},
		{name: "phrase only", item: struct{ phrase, grammar, transcription, translation string }{"再见", "", "", ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var line strings.Builder
			line.WriteString(tc.item.phrase)
			if tc.item.grammar != "" {
				line.WriteString(" {" + tc.item.grammar + "}")
			}
			if tc.item.transcription != "" {
				line.WriteString(" [" + tc.item.transcription + "]")
			}
			if tc.item.translation != "" {
				line.WriteString(" = " + tc.item.translation)
			}
			input := "{start-vocabulary}\n" + line.String() + "\n{end-vocabulary}\n"

			got, err := markdown.ToMDX([]byte(input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX(%q) unexpected error: %v", input, err)
			}
			// Extract the fence body (the single line between the fence markers).
			body := strings.TrimPrefix(string(got), "```vocabulary lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "\n```\n")

			source, grammar, transcription, translation := portParseStructuredEntry(body)
			if source != tc.item.phrase || grammar != tc.item.grammar || transcription != tc.item.transcription || translation != tc.item.translation {
				t.Fatalf("round-trip mismatch for %q:\n got  phrase=%q grammar=%q transcription=%q translation=%q\n want phrase=%q grammar=%q transcription=%q translation=%q",
					body, source, grammar, transcription, translation,
					tc.item.phrase, tc.item.grammar, tc.item.transcription, tc.item.translation)
			}
		})
	}
}

// portParseDialogTurn is a Go test port of phraseforge-web's
// src/remark/lessonElements.ts turn-parsing rules (namedTurnPattern
// "^@(.+?):\\s*$", anonymousTurnPattern "^--:\\s*$", indentPattern
// "^(?:    |  |\\t)(.*)$"): given ONE marker line and the turn's raw
// (still-indented) body lines, return the turn's speaker (""  for an
// anonymous "--:" turn) and its de-indented content lines.
func portParseDialogTurn(marker string, bodyLines []string) (speaker string, content []string) {
	switch {
	case marker == "--:":
		speaker = ""
	default:
		speaker = strings.TrimSuffix(strings.TrimPrefix(marker, "@"), ":")
	}
	for _, line := range bodyLines {
		switch {
		case strings.HasPrefix(line, "    "):
			content = append(content, line[4:])
		case strings.HasPrefix(line, "  "):
			content = append(content, line[2:])
		case strings.HasPrefix(line, "\t"):
			content = append(content, line[1:])
		case line == "":
			content = append(content, "")
		}
	}
	return speaker, content
}

// TestToMDX_DialogRoundTrip proves the emitted dialog fence body's marker
// lines and 2-space-indented content re-parse (under the phraseforge
// turn-parsing rules ported above) to the original DialogItem
// Header/Content: an anonymous turn's marker recovers "--:", a named
// turn's marker recovers "Name" (matching the DialogItem.Header's
// "Name:" with the trailing colon stripped, per namedTurnPattern's own
// capture group), and the de-indented content lines rejoin to the
// original Content.
func TestToMDX_DialogRoundTrip(t *testing.T) {
	input := "{start-dialog}\n" +
		"--:\n" +
		"  Hello there.\n" +
		"@Bob:\n" +
		"  Line1\n" +
		"\n" +
		"  Line2\n" +
		"{end-dialog}\n"

	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX(%q) unexpected error: %v", input, err)
	}
	body := strings.TrimPrefix(string(got), "```dialog lang=lat script=latn\n")
	body = strings.TrimSuffix(body, "\n```\n")
	lines := strings.Split(body, "\n")

	// Turn 1: "--:" marker, single-line content "Hello there.".
	speaker1, content1 := portParseDialogTurn(lines[0], lines[1:2])
	if speaker1 != "" || strings.Join(content1, "\n") != "Hello there." {
		t.Fatalf("turn 1 round-trip: speaker=%q content=%q, want speaker=\"\" content=\"Hello there.\"", speaker1, content1)
	}

	// Turn 2: "@Bob:" marker, two content lines around a blank line.
	speaker2, content2 := portParseDialogTurn(lines[2], lines[3:6])
	if speaker2 != "Bob" || strings.Join(content2, "\n") != "Line1\n\nLine2" {
		t.Fatalf("turn 2 round-trip: speaker=%q content=%q, want speaker=\"Bob\" content=\"Line1\\n\\nLine2\"", speaker2, content2)
	}
}

// TestToMDX_ModelsRoundTrip proves parse -> ToMDX fence -> re-parse is
// idempotent for the models block: the fence body extracted from ToMDX's
// output, re-wrapped in {start-models}...{end-models} and converted back
// to HTML, must render identically to the original input's own HTML
// (public-API-only proof that parseModelsItems recovers the same items).
func TestToMDX_ModelsRoundTrip(t *testing.T) {
	lines := []string{
		"run [rʌn] = biec",
		"run [rʌn]",
		"run = biec",
		"再见",
	}
	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			input := "{start-models}\n" + line + "\n{end-models}\n"

			gotMDX, err := markdown.ToMDX([]byte(input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX(%q) unexpected error: %v", input, err)
			}
			body := strings.TrimPrefix(string(gotMDX), "```models lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "\n```\n")

			originalHTML, err := markdown.ToHTML([]byte(input))
			if err != nil {
				t.Fatalf("ToHTML(%q) unexpected error: %v", input, err)
			}
			reparsedHTML, err := markdown.ToHTML([]byte("{start-models}\n" + body + "\n{end-models}\n"))
			if err != nil {
				t.Fatalf("ToHTML(%q) unexpected error: %v", body, err)
			}
			if string(reparsedHTML) != string(originalHTML) {
				t.Fatalf("round-trip mismatch for %q:\n got  %q\n want %q", line, string(reparsedHTML), string(originalHTML))
			}
		})
	}
}

// TestToMDX_QuestionsRoundTrip proves parse -> ToMDX fence -> re-parse is
// idempotent for the questions block, mirroring
// TestToMDX_ModelsRoundTrip.
func TestToMDX_QuestionsRoundTrip(t *testing.T) {
	lines := []string{
		"What is your name?",
		"Where are you from? = Poland",
		"Q = A = B",
	}
	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			input := "{start-questions}\n" + line + "\n{end-questions}\n"

			gotMDX, err := markdown.ToMDX([]byte(input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX(%q) unexpected error: %v", input, err)
			}
			body := strings.TrimPrefix(string(gotMDX), "```questions lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "\n```\n")

			originalHTML, err := markdown.ToHTML([]byte(input))
			if err != nil {
				t.Fatalf("ToHTML(%q) unexpected error: %v", input, err)
			}
			reparsedHTML, err := markdown.ToHTML([]byte("{start-questions}\n" + body + "\n{end-questions}\n"))
			if err != nil {
				t.Fatalf("ToHTML(%q) unexpected error: %v", body, err)
			}
			if string(reparsedHTML) != string(originalHTML) {
				t.Fatalf("round-trip mismatch for %q:\n got  %q\n want %q", line, string(reparsedHTML), string(originalHTML))
			}
		})
	}
}

// ---------------------------------------------------------------------
// SPECS §12.5 — MDX round-trip non-regression
// ---------------------------------------------------------------------

// TestMDXVocabularyHeaderRoundTrip asserts:
//  1. The EXACT serialized fence-line string for a vocabulary header item —
//     specifically that no trailing " = " separator is appended (F7 / SPECS §10).
//  2. parse→ToMDX→parse is lossless: re-parsing the fence body produces the
//     same HTML as the original input.
func TestMDXVocabularyHeaderRoundTrip(t *testing.T) {
	input := "{start-vocabulary}\nphrase1 = t1\n## Heading\nphrase2 = t2\n{end-vocabulary}\n"

	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}

	// (1) Exact serialized string: the header must appear as "## Heading" with
	// no trailing " = " or other data-item suffix.
	want := "```vocabulary lang=lat script=latn\n" +
		"phrase1 = t1\n" +
		"## Heading\n" +
		"phrase2 = t2\n" +
		"```\n"
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch\n got: %q\nwant: %q", string(got), want)
	}

	// (2) parse→ToMDX→parse losslessness: extract the fence body, re-wrap,
	// convert to HTML, and compare to the original's HTML.
	body := strings.TrimPrefix(string(got), "```vocabulary lang=lat script=latn\n")
	body = strings.TrimSuffix(body, "\n```\n")

	originalHTML, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	reparsedHTML, err := markdown.ToHTML([]byte("{start-vocabulary}\n" + body + "\n{end-vocabulary}\n"))
	if err != nil {
		t.Fatalf("ToHTML() round-trip unexpected error: %v", err)
	}
	if string(reparsedHTML) != string(originalHTML) {
		t.Fatalf("round-trip HTML mismatch\n got: %q\nwant: %q", string(reparsedHTML), string(originalHTML))
	}
}

// TestMDXDialogNoteRoundTrip asserts:
//  1. The EXACT serialized fence-line string for a dialog note item — the
//     parentheses that were stripped during parsing are re-added (D5 / SPECS §10).
//  2. parse→ToMDX→parse is lossless.
func TestMDXDialogNoteRoundTrip(t *testing.T) {
	input := "{start-dialog}\n--:\n  Hello there.\n(A greeting)\n{end-dialog}\n"

	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}

	// (1) Exact serialized string: the note appears as "(A greeting)" with
	// parens restored, at block level (un-indented), distinct from turn content.
	want := "```dialog lang=lat script=latn\n" +
		"--:\n" +
		"  Hello there.\n" +
		"(A greeting)\n" +
		"```\n"
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch\n got: %q\nwant: %q", string(got), want)
	}

	// (2) parse→ToMDX→parse losslessness via HTML comparison.
	body := strings.TrimPrefix(string(got), "```dialog lang=lat script=latn\n")
	body = strings.TrimSuffix(body, "\n```\n")

	originalHTML, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	reparsedHTML, err := markdown.ToHTML([]byte("{start-dialog}\n" + body + "\n{end-dialog}\n"))
	if err != nil {
		t.Fatalf("ToHTML() round-trip unexpected error: %v", err)
	}
	if string(reparsedHTML) != string(originalHTML) {
		t.Fatalf("round-trip HTML mismatch\n got: %q\nwant: %q", string(reparsedHTML), string(originalHTML))
	}
}

// TestMDXDataOnlyUnchanged is the ASR-3 byte-identical regression control for
// the MDX renderer: a vocabulary block with no header/note lines must produce
// MDX fence output that is byte-identical to the pre-change golden (SPECS §12.5,
// ASR-3).
func TestMDXDataOnlyUnchanged(t *testing.T) {
	input := "{start-vocabulary}\np = t\n{end-vocabulary}\n"
	want := "```vocabulary lang=lat script=latn\np = t\n```\n"
	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch (ASR-3 regression)\n got: %q\nwant: %q", string(got), want)
	}
}

// ---------------------------------------------------------------------
// Phase-6 review recommendations R1 and R2
// ---------------------------------------------------------------------

// TestMDXHeaderEmptyTextTrailingSpace is the R1 explicit test case: a
// vocabulary header with empty text (e.g. "#### " — bare hashes + a single
// space) must be re-serialized with the trailing space preserved in the fence
// body (the serializer emits strings.Repeat("#", Level) + " " + Text where
// Text is ""), so the fence body line is "#### " not "####". This ensures
// parse→ToMDX round-trip is lossless for empty-text headers.
func TestMDXHeaderEmptyTextTrailingSpace(t *testing.T) {
	// "#### " is TrimSpace'd to "####" by the vocabulary parser, then
	// isBlockHeader("####") → level=4, text="", ok=true.
	// The MDX serializer emits: "####" + " " + "" = "#### " (with trailing space).
	input := "{start-vocabulary}\n#### \n{end-vocabulary}\n"
	want := "```vocabulary lang=lat script=latn\n#### \n```\n"
	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch (R1: empty-text header trailing space)\n got: %q\nwant: %q", string(got), want)
	}
}

// TestMDXNotePaddedWhitespaceStripped is the R2 explicit test case: a models
// note with padded inner whitespace — e.g. "( padded note )" — must be
// re-serialized as "(padded note)" (inner text is TrimSpace'd by isBlockNote
// per SPECS §3.2/D5), NOT as "( padded note )". The fence body must contain
// the trimmed form.
func TestMDXNotePaddedWhitespaceStripped(t *testing.T) {
	// isBlockNote("( padded note )") → inner = " padded note ", TrimSpace → "padded note".
	// MDX serializer: "(" + "padded note" + ")" = "(padded note)".
	input := "{start-models}\n( padded note )\n{end-models}\n"
	want := "```models lang=lat script=latn\n(padded note)\n```\n"
	got, err := markdown.ToMDX([]byte(input), "lat", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch (R2: padded note whitespace stripped)\n got: %q\nwant: %q", string(got), want)
	}
}
