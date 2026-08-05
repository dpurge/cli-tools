package markdown

// Unit tests for the marker.go helpers (SPECS FR-5/FR-7).
// These are in package markdown (not markdown_test) so that unexported
// functions — badgeOnlyTypst, badgeOnlyHTML, injectBadgeIntoFirstHeadingTypst,
// injectBadgeIntoFirstHeadingHTML — are accessible directly.

import "testing"

// TestBadgeOnlyTypst covers the standalone Typst badge block, both LTR and RTL.
func TestBadgeOnlyTypst(t *testing.T) {
	tests := []struct {
		name   string
		letter string
		dir    string
		want   string
	}{
		{
			name:   "LTR emits inline bracket form",
			letter: "T",
			dir:    "ltr",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n",
		},
		{
			name:   "LTR empty dir string treated as LTR",
			letter: "V",
			dir:    "",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n",
		},
		{
			name:   "RTL emits align(right) form",
			letter: "T",
			dir:    "rtl",
			want:   "#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge(\"T\")])\n\n",
		},
		{
			name:   "RTL letter D",
			letter: "D",
			dir:    "rtl",
			want:   "#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge(\"D\")])\n\n",
		},
		{
			name:   "all five letters LTR",
			letter: "M",
			dir:    "ltr",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"M\")]\n\n",
		},
		{
			name:   "Q letter LTR",
			letter: "Q",
			dir:    "ltr",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"Q\")]\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := badgeOnlyTypst(tc.letter, tc.dir)
			if got != tc.want {
				t.Fatalf("badgeOnlyTypst(%q, %q) = %q, want %q", tc.letter, tc.dir, got, tc.want)
			}
		})
	}
}

// TestBadgeOnlyHTML covers the standalone HTML badge div, both LTR and RTL.
func TestBadgeOnlyHTML(t *testing.T) {
	tests := []struct {
		name   string
		letter string
		dir    string
		want   string
	}{
		{
			name:   "LTR emits no dir attribute",
			letter: "V",
			dir:    "ltr",
			want:   `<div class="block-marker"><span class="ct-badge">V</span></div>` + "\n",
		},
		{
			name:   "empty dir treated as LTR (no dir attribute)",
			letter: "D",
			dir:    "",
			want:   `<div class="block-marker"><span class="ct-badge">D</span></div>` + "\n",
		},
		{
			name:   "RTL emits dir=rtl attribute",
			letter: "T",
			dir:    "rtl",
			want:   `<div class="block-marker" dir="rtl"><span class="ct-badge">T</span></div>` + "\n",
		},
		{
			name:   "RTL letter M",
			letter: "M",
			dir:    "rtl",
			want:   `<div class="block-marker" dir="rtl"><span class="ct-badge">M</span></div>` + "\n",
		},
		{
			name:   "Q letter LTR",
			letter: "Q",
			dir:    "ltr",
			want:   `<div class="block-marker"><span class="ct-badge">Q</span></div>` + "\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := badgeOnlyHTML(tc.letter, tc.dir)
			if got != tc.want {
				t.Fatalf("badgeOnlyHTML(%q, %q) = %q, want %q", tc.letter, tc.dir, got, tc.want)
			}
		})
	}
}

// TestInjectBadgeIntoFirstHeadingTypst covers the Typst heading injection:
// badge is inserted right after the "= " prefix of the first heading line.
func TestInjectBadgeIntoFirstHeadingTypst(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
		letter   string
		wantOut  string
		wantOK   bool
	}{
		{
			name:     "H1 heading present - badge injected after '= '",
			rendered: "= Section\n\nBody text\\.\n\n",
			letter:   "T",
			wantOut:  "= #_ctbadge(\"T\") Section\n\nBody text\\.\n\n",
			wantOK:   true,
		},
		{
			name:     "H2 heading present - badge injected after '== '",
			rendered: "== Subtitle\n\n",
			letter:   "V",
			wantOut:  "== #_ctbadge(\"V\") Subtitle\n\n",
			wantOK:   true,
		},
		{
			name:     "H3 heading present - badge injected after '=== '",
			rendered: "=== Deep\n\n",
			letter:   "D",
			wantOut:  "=== #_ctbadge(\"D\") Deep\n\n",
			wantOK:   true,
		},
		{
			name:     "heading not first line - badge still injected into first heading wherever it appears",
			rendered: "Preamble paragraph\\.\n\n== Second line heading\n\n",
			letter:   "T",
			wantOut:  "Preamble paragraph\\.\n\n== #_ctbadge(\"T\") Second line heading\n\n",
			wantOK:   true,
		},
		{
			name:     "no heading - returns false and string unchanged",
			rendered: "Plain paragraph without any heading\\.\n\n",
			letter:   "T",
			wantOut:  "Plain paragraph without any heading\\.\n\n",
			wantOK:   false,
		},
		{
			name:     "empty input - returns false",
			rendered: "",
			letter:   "T",
			wantOut:  "",
			wantOK:   false,
		},
		{
			// A raw Typst code block starts with '#raw(', not '=', so the
			// '= not a heading' text inside the raw() argument is NOT on a
			// line starting with '=', and must not be treated as a heading.
			name:     "code-fence false-positive: '= ...' inside #raw() is not a heading",
			rendered: "#raw(block: true, \"= not a heading\\n\")\n\n",
			letter:   "T",
			wantOut:  "#raw(block: true, \"= not a heading\\n\")\n\n",
			wantOK:   false,
		},
		{
			// A line that starts with '=' but has no space after the '=' run
			// must not be treated as a heading (e.g. an assignment in a raw block
			// or content like "=foo").
			name:     "'=' without trailing space is not a heading",
			rendered: "=nospace\n\n",
			letter:   "T",
			wantOut:  "=nospace\n\n",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotOK := injectBadgeIntoFirstHeadingTypst(tc.rendered, tc.letter)
			if gotOK != tc.wantOK {
				t.Fatalf("injectBadgeIntoFirstHeadingTypst() ok = %v, want %v", gotOK, tc.wantOK)
			}
			if gotOut != tc.wantOut {
				t.Fatalf("injectBadgeIntoFirstHeadingTypst() out =\n  %q\nwant:\n  %q", gotOut, tc.wantOut)
			}
		})
	}
}

// TestInjectBadgeIntoFirstHeadingHTML covers the HTML heading injection:
// 'block-title' class is added to the first <h1>/<h2>/<h3> tag, and the
// badge span is inserted immediately after its closing '>'.
func TestInjectBadgeIntoFirstHeadingHTML(t *testing.T) {
	tests := []struct {
		name     string
		rendered string
		letter   string
		wantOut  string
		wantOK   bool
	}{
		{
			name:     "h1 with id - block-title class appended, badge after >",
			rendered: `<h1 id="section">Title</h1>` + "\n",
			letter:   "T",
			wantOut:  `<h1 id="section" class="block-title"><span class="ct-badge">T</span>Title</h1>` + "\n",
			wantOK:   true,
		},
		{
			name:     "h2 with id - class added, badge inserted",
			rendered: `<h2 id="sub">Subtitle</h2>` + "\n",
			letter:   "V",
			wantOut:  `<h2 id="sub" class="block-title"><span class="ct-badge">V</span>Subtitle</h2>` + "\n",
			wantOK:   true,
		},
		{
			name:     "h3 with id - class added, badge inserted",
			rendered: `<h3 id="deep">Deep</h3>` + "\n",
			letter:   "D",
			wantOut:  `<h3 id="deep" class="block-title"><span class="ct-badge">D</span>Deep</h3>` + "\n",
			wantOK:   true,
		},
		{
			name: "heading not first in output - badge still injected into first heading",
			rendered: "<p>Preamble.</p>\n" +
				`<h2 id="later">Later Heading</h2>` + "\n",
			letter: "T",
			wantOut: "<p>Preamble.</p>\n" +
				`<h2 id="later" class="block-title"><span class="ct-badge">T</span>Later Heading</h2>` + "\n",
			wantOK: true,
		},
		{
			name:     "no heading - returns false and string unchanged",
			rendered: "<p>Plain paragraph with no heading.</p>\n",
			letter:   "T",
			wantOut:  "<p>Plain paragraph with no heading.</p>\n",
			wantOK:   false,
		},
		{
			name:     "empty input - returns false",
			rendered: "",
			letter:   "T",
			wantOut:  "",
			wantOK:   false,
		},
		{
			// Text inside a <pre> block that contains an h1-like literal must
			// NOT be matched: the renderer never emits a bare "<h1" inside a
			// <pre>, so this tests that the scanner matches only actual tags.
			// A <pre> block containing literal HTML text does not confuse
			// injectBadgeIntoFirstHeadingHTML because it scans the rendered
			// output string for the first "<h1"/"<h2"/"<h3" byte sequence —
			// a <pre> enclosing literal text would require the author to have
			// written the literal substring "<h1" inside the pre, which the
			// goldmark HTML renderer does not do for code blocks (it
			// HTML-escapes '<' to '&lt;').
			name:     "code-fence false-positive: &lt;h1 in pre is not a tag",
			rendered: "<pre><code>&lt;h1 id=\"x\"&gt;not a heading&lt;/h1&gt;\n</code></pre>\n",
			letter:   "T",
			wantOut:  "<pre><code>&lt;h1 id=\"x\"&gt;not a heading&lt;/h1&gt;\n</code></pre>\n",
			wantOK:   false,
		},
		{
			// existing class= on the open tag gets 'block-title ' prepended
			name:     "tag already has class attribute - block-title prepended",
			rendered: `<h1 id="x" class="existing">Title</h1>` + "\n",
			letter:   "T",
			wantOut:  `<h1 id="x" class="block-title existing"><span class="ct-badge">T</span>Title</h1>` + "\n",
			wantOK:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotOK := injectBadgeIntoFirstHeadingHTML(tc.rendered, tc.letter)
			if gotOK != tc.wantOK {
				t.Fatalf("injectBadgeIntoFirstHeadingHTML() ok = %v, want %v", gotOK, tc.wantOK)
			}
			if gotOut != tc.wantOut {
				t.Fatalf("injectBadgeIntoFirstHeadingHTML() out =\n  %q\nwant:\n  %q", gotOut, tc.wantOut)
			}
		})
	}
}
