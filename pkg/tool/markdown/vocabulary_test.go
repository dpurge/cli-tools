package markdown_test

import (
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Vocabulary_Golden asserts the EXACT (byte-identical) wrapper
// and span output for the {start-vocabulary}...{end-vocabulary} block per
// SPECS §4.1. Vocabulary content is raw and unescaped, so these are golden,
// full-string comparisons rather than substring checks.
func TestToHTML_Vocabulary_Golden(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full line: phrase, grammar, transcription, translation",
			input: "{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\">hello</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase only",
			input: "{start-vocabulary}\n你好\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + translation",
			input: "{start-vocabulary}\n谢谢 = thank you\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\">thank you</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + grammar",
			input: "{start-vocabulary}\n早上好 {greeting}\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">早上好</span>\n" +
				"<span class=\"vocabulary-grammar\">greeting</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name: "multiple items",
			input: "{start-vocabulary}\n" +
				"你好 {noun} [nǐ hǎo] = hello\n" +
				"再见\n" +
				"谢谢 = thank you\n" +
				"{end-vocabulary}\n",
			want: "<div class=\"vocabulary\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\">hello</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">再见</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\">thank you</span>\n" +
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
