package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// ---------------------------------------------------------------------
// AST / parser cases
//
// {start-section} holds a compulsory H1 title, an optional "- Author"
// list (rendered comma-separated), and an optional lone "(YYYY)" year
// line. Unlike vocabulary/dialog/parallel, it has no repeating item
// grammar: at most one of each element, in any order, with every other
// line a grammar error.
// ---------------------------------------------------------------------

// TestToHTML_Section_Golden asserts the exact wrapper/title/authors/year
// shape, including the s-<script> class hook and the "S" content badge.
func TestToHTML_Section_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "title, single author, year",
			input: "{start-section lang=spa script=latn}\n" +
				"# All Spanish Method\n" +
				"\n" +
				"- Guillermo Hall\n" +
				"\n" +
				"(1918)\n" +
				"{end-section}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">S</span></div>\n" +
				"<div class=\"section s-latn\" dir=\"ltr\">\n" +
				"<h1 class=\"section-title\">All Spanish Method</h1>\n" +
				"<p class=\"section-authors\">Guillermo Hall</p>\n" +
				"<p class=\"section-year\">1918</p>\n" +
				"</div>\n",
		},
		{
			name:  "title only — authors/year omitted entirely",
			input: "{start-section}\n# Bare Title\n{end-section}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">S</span></div>\n" +
				"<div class=\"section\" dir=\"ltr\">\n" +
				"<h1 class=\"section-title\">Bare Title</h1>\n" +
				"</div>\n",
		},
		{
			name: "multiple authors — comma-joined",
			input: "{start-section}\n" +
				"# Title Here\n" +
				"\n" +
				"- Author One\n" +
				"- Author Two\n" +
				"{end-section}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">S</span></div>\n" +
				"<div class=\"section\" dir=\"ltr\">\n" +
				"<h1 class=\"section-title\">Title Here</h1>\n" +
				"<p class=\"section-authors\">Author One, Author Two</p>\n" +
				"</div>\n",
		},
		{
			// script=arab: dir="rtl" (marker-driven, mirrors every other block).
			name:  "script=arab: dir=rtl",
			input: "{start-section script=arab}\n# Title\n{end-section}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">S</span></div>\n" +
				"<div class=\"section s-arab\" dir=\"rtl\">\n" +
				"<h1 class=\"section-title\">Title</h1>\n" +
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

// TestSection_MissingTitle_Errors verifies the compulsory-title rule.
func TestSection_MissingTitle_Errors(t *testing.T) {
	input := "{start-section}\n- Solo Author\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for a section with no title, got nil")
	}
	if !strings.Contains(err.Error(), "missing its compulsory title") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "missing its compulsory title")
	}
}

// TestSection_NonLevel1Title_Errors verifies the title must be a level-1
// heading, not any other level.
func TestSection_NonLevel1Title_Errors(t *testing.T) {
	input := "{start-section}\n## Not H1\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for a non-level-1 title, got nil")
	}
	if !strings.Contains(err.Error(), "must be a level-1 heading") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "must be a level-1 heading")
	}
}

// TestSection_DuplicateTitle_Errors verifies at most one title is allowed.
func TestSection_DuplicateTitle_Errors(t *testing.T) {
	input := "{start-section}\n# First\n# Second\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for two titles, got nil")
	}
	if !strings.Contains(err.Error(), "more than one title") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "more than one title")
	}
}

// TestSection_DuplicateYear_Errors verifies at most one year line is allowed.
func TestSection_DuplicateYear_Errors(t *testing.T) {
	input := "{start-section}\n# T\n\n(1918)\n\n(1920)\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for two year lines, got nil")
	}
	if !strings.Contains(err.Error(), "more than one year") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "more than one year")
	}
}

// TestSection_NonNumericYear_Errors verifies the year must be digits only
// (parens stripped), not arbitrary parenthesized text.
func TestSection_NonNumericYear_Errors(t *testing.T) {
	input := "{start-section}\n# T\n\n(19xx)\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for a non-numeric year, got nil")
	}
	if !strings.Contains(err.Error(), "year must be numeric") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "year must be numeric")
	}
}

// TestSection_UnrecognizedLine_Errors verifies a section block holds only
// the title/authors/year shapes — arbitrary prose is a grammar error.
func TestSection_UnrecognizedLine_Errors(t *testing.T) {
	input := "{start-section}\n# T\n\nSome stray prose.\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatal("ToHTML() expected an error for an unrecognized line, got nil")
	}
	if !strings.Contains(err.Error(), "unrecognized line in section block") {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), "unrecognized line in section block")
	}
}

// TestSection_As_Rejected mirrors TestParallel_As_Rejected: a section has no
// field languages, so as= is rejected entirely.
func TestSection_As_Rejected(t *testing.T) {
	input := "{start-section as=source}\n# T\n{end-section}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-section}, got nil")
	}
	wantErr := "as= not applicable to {start-section}: it has no field languages"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}

// ---------------------------------------------------------------------
// Typst emission (Go-side dict/call-string assertions)
// ---------------------------------------------------------------------

// TestToTypst_Section_Golden asserts the exact #section(...) call emission,
// including the LEADING `#pagebreak(weak: true)` (emitted before the badge,
// not inside book.typ's section() itself — see renderSectionTypst's doc
// comment for why) and the authors tuple/none-year shape.
func TestToTypst_Section_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "title, single author, year",
			input: "{start-section lang=spa script=latn}\n" +
				"# All Spanish Method\n\n- Guillermo Hall\n\n(1918)\n" +
				"{end-section}\n",
			want: "#pagebreak(weak: true)\n" +
				"#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"S\")]\n\n" +
				"#section(dir: ltr, script: \"latn\", title: \"All Spanish Method\", authors: (\"Guillermo Hall\", ), year: \"1918\")\n\n",
		},
		{
			name:  "title only — empty authors tuple, year: none",
			input: "{start-section}\n# Bare Title\n{end-section}\n",
			want: "#pagebreak(weak: true)\n" +
				"#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"S\")]\n\n" +
				"#section(dir: ltr, script: \"\", title: \"Bare Title\", authors: (), year: none)\n\n",
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
}

// TestToTypst_Section_Err verifies that a parse-time error (missing title,
// etc.) surfaces out of ToTypst with no output.
func TestToTypst_Section_Err(t *testing.T) {
	input := "{start-section}\n- Solo Author\n{end-section}\n"

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

// TestToMDX_Section_Golden asserts the exact `section` fence shape: a
// top-level fence (NOT wrapped in <Text>, unlike ParallelDialog — Section
// gets first-class top-level treatment like vocabulary/dialog/parallel/
// models/questions/text, mdx.go's ToMDX classification switch).
func TestToMDX_Section_Golden(t *testing.T) {
	input := "{start-section lang=spa script=latn}\n" +
		"# All Spanish Method\n\n- Guillermo Hall\n\n(1918)\n" +
		"{end-section}\n"
	want := "```section lang=spa script=latn\n" +
		"# All Spanish Method\n\n- Guillermo Hall\n\n(1918)\n" +
		"```\n"

	got, err := markdown.ToMDX([]byte(input), "en", "latn")
	if err != nil {
		t.Fatalf("ToMDX() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToMDX() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestToMDX_Section_RoundTripStable verifies that serializing to MDX and
// re-parsing the fence body produces an identical block, evidenced by
// byte-identical HTML output — mirrors TestToMDX_ParallelDialog_RoundTripStable.
func TestToMDX_Section_RoundTripStable(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "title, authors, year",
			input: "{start-section}\n# All Spanish Method\n\n- Guillermo Hall\n\n(1918)\n{end-section}\n",
		},
		{
			name:  "title only",
			input: "{start-section}\n# Bare Title\n{end-section}\n",
		},
		{
			name:  "multiple authors, no year",
			input: "{start-section}\n# Title\n\n- Author One\n- Author Two\n{end-section}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMDX, err := markdown.ToMDX([]byte(tc.input), "lat", "latn")
			if err != nil {
				t.Fatalf("ToMDX() unexpected error: %v", err)
			}
			body := string(gotMDX)
			body = strings.TrimPrefix(body, "```section lang=lat script=latn\n")
			body = strings.TrimSuffix(body, "```\n")
			body = strings.TrimSuffix(body, "\n")

			originalHTML, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("original ToHTML() unexpected error: %v", err)
			}
			reparsedHTML, err := markdown.ToHTML([]byte("{start-section}\n" + body + "\n{end-section}\n"))
			if err != nil {
				t.Fatalf("reparsed ToHTML() unexpected error: %v", err)
			}
			if string(originalHTML) != string(reparsedHTML) {
				t.Fatalf("round-trip mismatch\noriginal:  %q\nreparsed: %q", originalHTML, reparsedHTML)
			}
		})
	}
}
