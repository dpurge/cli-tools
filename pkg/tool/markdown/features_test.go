package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_FeatureSmoke asserts (via substring, per the semantic-
// equivalence bar in SPECS §6 for standard-markdown features) that each
// goldmark extension wired up in converter.go actually fires: tables,
// strikethrough, autolink, explicit links with target="_blank", definition
// lists, self-closed thematic breaks/hard breaks/images (html.WithXHTML()),
// and heading IDs (parser.WithAutoHeadingID()).
func TestToHTML_FeatureSmoke(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "table",
			input: "| a | b |\n|---|---|\n| 1 | 2 |\n",
			want:  "<table>",
		},
		{
			name:  "strikethrough",
			input: "~~x~~\n",
			want:  "<del>",
		},
		{
			name:  "bare autolink gets target=_blank",
			input: "https://x.com\n",
			want:  `<a href="https://x.com" target="_blank">`,
		},
		{
			name:  "explicit link gets target=_blank",
			input: "[t](u)\n",
			want:  `target="_blank"`,
		},
		{
			name:  "definition list",
			input: "Term\n: Definition\n",
			want:  "<dl>",
		},
		{
			name:  "thematic break is self-closed",
			input: "a\n\n---\n\nb\n",
			want:  "<hr />",
		},
		{
			name:  "hard break is self-closed",
			input: "line1  \nline2\n",
			want:  "<br />",
		},
		{
			name:  "image is self-closed",
			input: "![alt](src.png)\n",
			want:  "<img",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToHTML() unexpected error: %v", err)
			}
			if !strings.Contains(string(got), tc.want) {
				t.Fatalf("ToHTML(%q) = %q, want substring %q", tc.input, string(got), tc.want)
			}
		})
	}
}

// TestToHTML_Image_SelfClosed additionally confirms the image tag's self-
// closing slash specifically (not just the opening "<img"), since
// html.WithXHTML() self-closes <img> as a new, accepted divergence (SPECS §6).
func TestToHTML_Image_SelfClosed(t *testing.T) {
	got, err := markdown.ToHTML([]byte("![alt](src.png)\n"))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "/>") {
		t.Fatalf("expected self-closed <img ... />, got: %q", string(got))
	}
}

// TestToHTML_Heading_GetsID confirms parser.WithAutoHeadingID() assigns an
// id= attribute to headings.
func TestToHTML_Heading_GetsID(t *testing.T) {
	got, err := markdown.ToHTML([]byte("# Hello World\n"))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `id="`) {
		t.Fatalf("expected heading to carry an id= attribute, got: %q", string(got))
	}
}
