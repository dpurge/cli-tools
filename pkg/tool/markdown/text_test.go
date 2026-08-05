package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Text_Golden asserts the exact wrapper class and dir attribute
// produced by renderTextblock for each as= value (SPECS §7.1, §9.1, M3).
// Direction rule (D9): as=transcription pinned ltr; source/translation/grammar
// derive from the block's own script. as=source maps to class "text" (OI-8).
func TestToHTML_Text_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "source LTR script (latn) → class text dir ltr",
			input: "{start-text as=source script=latn}\nHello world.\n{end-text}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"text s-latn\" dir=\"ltr\">\n" +
				"<p>Hello world.</p>\n" +
				"</div>\n",
		},
		{
			name:  "source RTL script (arab) → class text dir rtl",
			input: "{start-text as=source script=arab}\nمرحبا\n{end-text}\n",
			want: "<div class=\"block-marker\" dir=\"rtl\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"text s-arab\" dir=\"rtl\">\n" +
				"<p>مرحبا</p>\n" +
				"</div>\n",
		},
		{
			name:  "transcription always ltr regardless of script",
			input: "{start-text as=transcription script=arab}\nmarhaban\n{end-text}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"transcription s-arab as-transcription\" dir=\"ltr\">\n" +
				"<p>marhaban</p>\n" +
				"</div>\n",
		},
		{
			name:  "translation LTR script (latn) → class translation dir ltr",
			input: "{start-text as=translation script=latn}\nHello in Latin.\n{end-text}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"translation s-latn as-translation\" dir=\"ltr\">\n" +
				"<p>Hello in Latin.</p>\n" +
				"</div>\n",
		},
		{
			name:  "translation RTL script (arab) → class translation dir rtl (D9)",
			input: "{start-text as=translation script=arab}\nترجمة\n{end-text}\n",
			want: "<div class=\"block-marker\" dir=\"rtl\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"translation s-arab as-translation\" dir=\"rtl\">\n" +
				"<p>ترجمة</p>\n" +
				"</div>\n",
		},
		{
			name:  "grammar LTR script (latn) → class grammar dir ltr",
			input: "{start-text as=grammar script=latn}\nProse explanation.\n{end-text}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"grammar s-latn as-grammar\" dir=\"ltr\">\n" +
				"<p>Prose explanation.</p>\n" +
				"</div>\n",
		},
		{
			name:  "source without script attr → dir ltr (default)",
			input: "{start-text as=source}\nDefault direction.\n{end-text}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">T</span></div>\n" +
				"<div class=\"text\" dir=\"ltr\">\n" +
				"<p>Default direction.</p>\n" +
				"</div>\n",
		},
		{
			name:  "heading inside source block is recursed through ToHTML",
			input: "{start-text as=source}\n# Section Title\n\nBody text.\n{end-text}\n",
			want: "<div class=\"text\" dir=\"ltr\">\n" +
				"<h1 id=\"section-title\" class=\"block-title\"><span class=\"ct-badge\">T</span>Section Title</h1>\n" +
				"<p>Body text.</p>\n" +
				"</div>\n",
		},
		{
			name:  "grammar with a table: table is inside div.grammar",
			input: "{start-text as=grammar script=latn}\n| A | B |\n|---|---|\n| 1 | 2 |\n{end-text}\n",
			want: "<div class=\"grammar s-latn as-grammar\" dir=\"ltr\">\n" +
				"<table>\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToHTML() unexpected error: %v", err)
			}
			// Last test uses substring check (table inner markup varies).
			if tc.name == "grammar with a table: table is inside div.grammar" {
				if !strings.Contains(string(got), "<div class=\"grammar s-latn as-grammar\" dir=\"ltr\">") {
					t.Fatalf("ToHTML() missing grammar wrapper\n got: %q", string(got))
				}
				if !strings.Contains(string(got), "<table>") {
					t.Fatalf("ToHTML() missing table inside grammar block\n got: %q", string(got))
				}
				return
			}
			if string(got) != tc.want {
				t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), tc.want)
			}
		})
	}
}

// TestToTypst_Text_Golden asserts the exact #textblock call emitted for each
// as= value (SPECS §7.1, M3). Typst function uses role: param (as is a
// reserved Typst keyword). Direction rule (D9) applied.
func TestToTypst_Text_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "source LTR script (latn) → role source dir ltr",
			input: "{start-text as=source script=latn}\nHello world.\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"source\", dir: ltr, script: \"latn\", [\nHello world\\.\n\n])\n\n",
		},
		{
			name:  "source RTL script (arab) → role source dir rtl",
			input: "{start-text as=source script=arab}\nمرحبا\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge(\"T\")])\n\n#textblock(role: \"source\", dir: rtl, script: \"arab\", [\nمرحبا\n\n])\n\n",
		},
		{
			name:  "transcription always ltr (pinned romanization, D9)",
			input: "{start-text as=transcription script=arab}\nmarhaban\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"transcription\", dir: ltr, script: \"arab\", [\nmarhaban\n\n])\n\n",
		},
		{
			name:  "translation LTR script (latn) → dir ltr",
			input: "{start-text as=translation script=latn}\nHello in Latin.\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"translation\", dir: ltr, script: \"latn\", [\nHello in Latin\\.\n\n])\n\n",
		},
		{
			name:  "translation RTL script (arab) → dir rtl (D9)",
			input: "{start-text as=translation script=arab}\nترجمة\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge(\"T\")])\n\n#textblock(role: \"translation\", dir: rtl, script: \"arab\", [\nترجمة\n\n])\n\n",
		},
		{
			name:  "grammar LTR script (latn) → dir ltr",
			input: "{start-text as=grammar script=latn}\nProse explanation.\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"grammar\", dir: ltr, script: \"latn\", [\nProse explanation\\.\n\n])\n\n",
		},
		{
			name:  "source no script → dir ltr (default)",
			input: "{start-text as=source}\nDefault direction.\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"source\", dir: ltr, script: \"\", [\nDefault direction\\.\n\n])\n\n",
		},
		{
			name:  "heading inside source block is recursed via ToTypst",
			input: "{start-text as=source script=latn}\n# Section\n\nBody.\n{end-text}\n",
			want:  "#textblock(role: \"source\", dir: ltr, script: \"latn\", [\n= #_ctbadge(\"T\") Section\n\nBody\\.\n\n])\n\n",
		},
		{
			// Grammar tables are emitted with integer columns so that book.typ's
			// grammar-scoped show rule can read it.columns and produce
			// (1fr,) * it.columns fractional tracks, making the table full-width.
			name:  "grammar with a table: integer columns emitted (book.typ transforms to fractional)",
			input: "{start-text as=grammar script=latn}\n| A | B |\n|---|---|\n| 1 | 2 |\n{end-text}\n",
			want:  "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n#textblock(role: \"grammar\", dir: ltr, script: \"latn\", [\n#table(columns: (1fr, 1fr), align: (auto, auto),\n[A],\n[B],\n[1],\n[2],\n)\n\n])\n\n",
		},
	}

	runTypstGolden(t, tests)
}

// TestToMDX_Text_Golden asserts the <Text> emission for each as= value
// (SPECS §7.7, OI-7, M3). as="source" omits the as= attribute (phraseforge
// corpus default). lang/script come from the node; fall back to call-level
// when absent.
func TestToMDX_Text_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		lang  string
		script string
		want  string
	}{
		{
			name:   "source omits as= attr",
			input:  "{start-text as=source lang=arb script=arab}\nمرحبا\n{end-text}\n",
			lang:   "arb",
			script: "arab",
			want:   "<Text lang=\"arb\" script=\"arab\">\n\nمرحبا\n\n</Text>",
		},
		{
			name:   "transcription includes as= attr",
			input:  "{start-text as=transcription lang=arb script=arab}\nmarhaban\n{end-text}\n",
			lang:   "arb",
			script: "arab",
			want:   "<Text as=\"transcription\" lang=\"arb\" script=\"arab\">\n\nmarhaban\n\n</Text>",
		},
		{
			name:   "translation includes as= attr",
			input:  "{start-text as=translation lang=arb script=latn}\nHello.\n{end-text}\n",
			lang:   "arb",
			script: "latn",
			want:   "<Text as=\"translation\" lang=\"arb\" script=\"latn\">\n\nHello.\n\n</Text>",
		},
		{
			name:   "grammar includes as= attr",
			input:  "{start-text as=grammar lang=arb script=latn}\nNoun.\n{end-text}\n",
			lang:   "arb",
			script: "latn",
			want:   "<Text as=\"grammar\" lang=\"arb\" script=\"latn\">\n\nNoun.\n\n</Text>",
		},
		{
			name:   "no node attrs fall back to call-level lang/script",
			input:  "{start-text as=source}\nFallback.\n{end-text}\n",
			lang:   "lat",
			script: "latn",
			want:   "<Text lang=\"lat\" script=\"latn\">\n\nFallback.\n\n</Text>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToMDX([]byte(tc.input), tc.lang, tc.script)
			if err != nil {
				t.Fatalf("ToMDX() unexpected error: %v", err)
			}
			if !strings.Contains(string(got), tc.want) {
				t.Fatalf("ToMDX() mismatch\n got: %q\nwant substring: %q", string(got), tc.want)
			}
		})
	}
}

// TestToMDX_Block_AttributeAwareFence asserts that vocabulary/dialog/
// parallel/models/questions MDX fences use the node's parsed lang/script
// when present, and fall back to the call-level values when absent (SPECS
// §7.7, M3 attribute-aware fences).
func TestToMDX_Block_AttributeAwareFence(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		lang   string
		script string
		want   string
	}{
		{
			name:   "vocabulary node attrs override call-level",
			input:  "{start-vocabulary lang=arb script=arab}\nكتاب = book\n{end-vocabulary}\n",
			lang:   "lat",
			script: "latn",
			want:   "```vocabulary lang=arb script=arab\n",
		},
		{
			name:   "vocabulary no attrs uses call-level",
			input:  "{start-vocabulary}\np = q\n{end-vocabulary}\n",
			lang:   "lat",
			script: "latn",
			want:   "```vocabulary lang=lat script=latn\n",
		},
		{
			name:   "models node attrs override call-level",
			input:  "{start-models lang=arb script=arab}\nكتاب [kitāb] = book\n{end-models}\n",
			lang:   "lat",
			script: "latn",
			want:   "```models lang=arb script=arab\n",
		},
		{
			name:   "questions node attrs override call-level",
			input:  "{start-questions lang=arb script=arab}\nسؤال = answer\n{end-questions}\n",
			lang:   "lat",
			script: "latn",
			want:   "```questions lang=arb script=arab\n",
		},
		{
			name:   "dialog node attrs override call-level",
			input:  "{start-dialog lang=arb script=arab}\n--:\n  مرحبا\n{end-dialog}\n",
			lang:   "lat",
			script: "latn",
			want:   "```dialog lang=arb script=arab\n",
		},
		{
			name:   "dialog no attrs uses call-level",
			input:  "{start-dialog}\n--:\n  Hello\n{end-dialog}\n",
			lang:   "lat",
			script: "latn",
			want:   "```dialog lang=lat script=latn\n",
		},
		{
			name:   "parallel node attrs override call-level",
			input:  "{start-parallel lang=arb script=arab}\nمرحبا\n{end-parallel}\n",
			lang:   "lat",
			script: "latn",
			want:   "```parallel lang=arb script=arab\n",
		},
		{
			name:   "parallel no attrs uses call-level",
			input:  "{start-parallel}\nHello\n{end-parallel}\n",
			lang:   "lat",
			script: "latn",
			want:   "```parallel lang=lat script=latn\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToMDX([]byte(tc.input), tc.lang, tc.script)
			if err != nil {
				t.Fatalf("ToMDX() unexpected error: %v", err)
			}
			if !strings.Contains(string(got), tc.want) {
				t.Fatalf("ToMDX() missing expected fence header\n got: %q\nwant substring: %q", string(got), tc.want)
			}
		})
	}
}
