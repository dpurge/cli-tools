package markdown_test

import (
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Dialog_Golden asserts the EXACT wrapper output for the
// {start-dialog}...{end-dialog} block per SPECS §4.2:
//   - "--:" header renders as "—"
//   - "@Bob:" strips the leading "@" but KEEPS the trailing colon -> "Bob:"
//   - "＠李:" strips the leading full-width "＠" but keeps the colon -> "李:"
//   - 2-space-indented content lines are dedented and rendered recursively
//     as standard markdown (so **bold** and multiple paragraphs work); the
//     inner HTML below was captured by running the converter itself, since
//     standard-markdown HTML content is accepted at the semantic-equivalence
//     bar (only the custom wrapper markup is byte-identical).
func TestToHTML_Dialog_Golden(t *testing.T) {
	input := "{start-dialog}\n" +
		"--:\n" +
		"  Hello there.\n" +
		"\n" +
		"  Second **bold** paragraph.\n" +
		"@Bob:\n" +
		"  Hi!\n" +
		"＠李:\n" +
		"  你好!\n" +
		"{end-dialog}\n"

	want := "<div class=\"dialog\" dir=\"ltr\">\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">—</div>\n" +
		"<div class=\"dialog-content\"><p>Hello there.</p>\n" +
		"<p>Second <strong>bold</strong> paragraph.</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">Bob:</div>\n" +
		"<div class=\"dialog-content\"><p>Hi!</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">李:</div>\n" +
		"<div class=\"dialog-content\"><p>你好!</p>\n" +
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
}
