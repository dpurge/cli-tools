package markdown_test

import (
	"strings"
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
			want: "<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\" dir=\"ltr\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">hello</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase only",
			input: "{start-vocabulary}\n你好\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + translation",
			input: "{start-vocabulary}\n谢谢 = thank you\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">thank you</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "phrase + grammar",
			input: "{start-vocabulary}\n早上好 {greeting}\n{end-vocabulary}\n",
			want: "<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">早上好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">greeting</span>\n" +
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
			want: "<div class=\"vocabulary\" dir=\"ltr\">\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">你好</span>\n" +
				"<span class=\"vocabulary-grammar\" dir=\"ltr\">noun</span>\n" +
				"<span class=\"vocabulary-transcription\" dir=\"ltr\">nǐ hǎo</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">hello</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">再见</span>\n" +
				"</div>\n" +
				"<div class=\"vocabulary-item\">\n" +
				"<span class=\"vocabulary-phrase\">谢谢</span>\n" +
				"<span class=\"vocabulary-translation\" dir=\"ltr\">thank you</span>\n" +
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

// TestVocabulary_As_Rejected covers SPECS §5: vocabulary's field languages
// are fixed (phrase=foreign, translation=base), so as= is rejected
// entirely — even an in-grammar value like "source" — with a clearer
// message than the pre-unification blanket rejection.
func TestVocabulary_As_Rejected(t *testing.T) {
	input := "{start-vocabulary as=source}\nphrase = translation\n{end-vocabulary}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an error for as= on {start-vocabulary}, got nil")
	}
	wantErr := "as= not applicable to {start-vocabulary}: its field languages are fixed"
	if !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), wantErr)
	}
}
