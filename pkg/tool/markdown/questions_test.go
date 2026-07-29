package markdown_test

import (
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Questions_Golden asserts the EXACT (byte-identical) wrapper
// output for the {start-questions}...{end-questions} block: a
// question-only line renders as a plain paragraph-style item (never a
// grid cell, per AC-3.1); a question+answer line renders as a paired
// two-column item; and consecutive question+answer items are grouped into
// a single "questions-group" wrapper (agreed migration decision), flushed
// whenever a question-only item or the end of the block is reached.
func TestToHTML_Questions_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "question only, no answer",
			input: "{start-questions}\nWhat is your name?\n{end-questions}\n",
			want: "<div class=\"questions\">\n" +
				"<div class=\"questions-item\">\n" +
				"<span class=\"questions-question\">What is your name?</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "question + answer",
			input: "{start-questions}\nWhere are you from? = Poland\n{end-questions}\n",
			want: "<div class=\"questions\">\n" +
				"<div class=\"questions-group\">\n" +
				"<div class=\"questions-item paired\">\n" +
				"<div class=\"questions-col1\">\n" +
				"<span class=\"questions-question\">Where are you from?</span>\n" +
				"</div>\n" +
				"<div class=\"questions-col2\">\n" +
				"<span class=\"questions-answer\">Poland</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "mixed block: Q-only, then a run of two Q+A, then Q-only again",
			input: "{start-questions}\n" +
				"Q1\n" +
				"Q2 = A2\n" +
				"Q3 = A3\n" +
				"Q4\n" +
				"{end-questions}\n",
			want: "<div class=\"questions\">\n" +
				"<div class=\"questions-item\">\n" +
				"<span class=\"questions-question\">Q1</span>\n" +
				"</div>\n" +
				"<div class=\"questions-group\">\n" +
				"<div class=\"questions-item paired\">\n" +
				"<div class=\"questions-col1\">\n" +
				"<span class=\"questions-question\">Q2</span>\n" +
				"</div>\n" +
				"<div class=\"questions-col2\">\n" +
				"<span class=\"questions-answer\">A2</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"questions-item paired\">\n" +
				"<div class=\"questions-col1\">\n" +
				"<span class=\"questions-question\">Q3</span>\n" +
				"</div>\n" +
				"<div class=\"questions-col2\">\n" +
				"<span class=\"questions-answer\">A3</span>\n" +
				"</div>\n" +
				"</div>\n" +
				"</div>\n" +
				"<div class=\"questions-item\">\n" +
				"<span class=\"questions-question\">Q4</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "answer containing '=' proves the FIRST ' = ' split, not the last",
			input: "{start-questions}\nQ = A = B\n{end-questions}\n",
			want: "<div class=\"questions\">\n" +
				"<div class=\"questions-group\">\n" +
				"<div class=\"questions-item paired\">\n" +
				"<div class=\"questions-col1\">\n" +
				"<span class=\"questions-question\">Q</span>\n" +
				"</div>\n" +
				"<div class=\"questions-col2\">\n" +
				"<span class=\"questions-answer\">A = B</span>\n" +
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
