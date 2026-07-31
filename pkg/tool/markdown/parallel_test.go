package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Parallel_Golden asserts the EXACT wrapper output for the
// {start-parallel}...{end-parallel} block per SPECS §4.3, including the D2
// fix: a row whose MAIN cell contains its own "---" thematic break must
// still split the secondary cell off at the LAST "---" line, and the first
// character of the main cell must not be dropped (the original gomarkdown
// port had a slice-length bug that dropped it).
func TestToHTML_Parallel_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "main cell contains thematic break, plus secondary after LAST ---",
			input: "{start-parallel}\n" +
				"First para.\n" +
				"\n" +
				"---\n" +
				"\n" +
				"Second para in main.\n" +
				"---\n" +
				"Secondary cell.\n" +
				"{end-parallel}\n",
			want: "<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<p>First para.</p>\n" +
				"<hr />\n" +
				"<p>Second para in main.</p>\n" +
				"\n</div>\n" +
				"<div class=\"parallel-cell secondary\" dir=\"ltr\">\n" +
				"<p>Secondary cell.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "row with no secondary cell",
			input: "{start-parallel}\n" +
				"Just main cell text.\n" +
				"{end-parallel}\n",
			want: "<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<p>Just main cell text.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "two rows",
			input: "{start-parallel}\n" +
				"Row1 main.\n" +
				"---\n" +
				"Row1 secondary.\n" +
				"===\n" +
				"Row2 main only.\n" +
				"{end-parallel}\n",
			want: "<div class=\"parallel\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<p>Row1 main.</p>\n" +
				"\n</div>\n" +
				"<div class=\"parallel-cell secondary\" dir=\"ltr\">\n" +
				"<p>Row1 secondary.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<p>Row2 main only.</p>\n" +
				"\n</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			// The secondary column follows the marker script: script=arab
			// pins its dir="rtl". The container and main cell carry NO dir
			// (they inherit the book direction from body{direction} in CSS).
			name: "script=arab pins secondary cell dir=rtl; main inherits book",
			input: "{start-parallel script=arab}\n" +
				"Main text.\n" +
				"---\n" +
				"Secondary text.\n" +
				"{end-parallel}\n",
			want: "<div class=\"parallel s-arab\">\n" +
				"<div class=\"parallel-row\">\n" +
				"<div class=\"parallel-cell main\">\n" +
				"<p>Main text.</p>\n" +
				"\n</div>\n" +
				"<div class=\"parallel-cell secondary\" dir=\"rtl\">\n" +
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
			// D2 regression is already covered by the exact-string `want`
			// comparison above (main cells render "First para."/"Row1 main."
			// with their leading character intact) and by the focused
			// TestToHTML_Parallel_MainCellFirstCharNotDropped below.
		})
	}
}

// TestToHTML_Parallel_MainCellFirstCharNotDropped is a focused D2 regression
// test: the main cell text starts immediately after the marker line, and
// its first character must appear verbatim in the output.
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
