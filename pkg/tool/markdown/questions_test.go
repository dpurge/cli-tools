package markdown_test

import (
	"strings"
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
			want: "<div class=\"questions\" dir=\"ltr\">\n" +
				"<div class=\"questions-item\">\n" +
				"<span class=\"questions-question\">What is your name?</span>\n" +
				"</div>\n" +
				"</div>\n",
		},
		{
			name:  "question + answer",
			input: "{start-questions}\nWhere are you from? = Poland\n{end-questions}\n",
			want: "<div class=\"questions\" dir=\"ltr\">\n" +
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
			want: "<div class=\"questions\" dir=\"ltr\">\n" +
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
			want: "<div class=\"questions\" dir=\"ltr\">\n" +
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

// TestQuestions_As_Unification covers SPECS §5's as= unification for
// {start-questions}: source (default, implicit) and translation are
// accepted; any other in-grammar value (transcription/grammar, valid on
// {start-text} but not here) is a build error naming questions' accepted
// set. This also fixes the previously-broken `lat` book content pattern
// (`lat/lewicki-1924/a05.md:85`-style as=translation usage on a paired
// block) for questions.
func TestQuestions_As_Unification(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // substring; empty means no error expected
	}{
		{
			name:  "as=source accepted (explicit)",
			input: "{start-questions as=source}\nQ1 = A1\n{end-questions}\n",
		},
		{
			name:  "as=translation accepted",
			input: "{start-questions as=translation}\nQ1 = A1\n{end-questions}\n",
		},
		{
			name:  "as omitted defaults to source (no error)",
			input: "{start-questions}\nQ1 = A1\n{end-questions}\n",
		},
		{
			name:    "as=transcription rejected with questions' accepted set named",
			input:   "{start-questions as=transcription}\nQ1 = A1\n{end-questions}\n",
			wantErr: `as="transcription" is not valid on {start-questions}: must be source|translation`,
		},
		{
			name:    "as=grammar rejected with questions' accepted set named",
			input:   "{start-questions as=grammar}\nQ1 = A1\n{end-questions}\n",
			wantErr: `as="grammar" is not valid on {start-questions}: must be source|translation`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := markdown.ToHTML([]byte(tc.input))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ToHTML() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ToHTML() expected an error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ToHTML() error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestQuestions_AsTranslation_EPUBClassToken covers the FIX-2/ASR-4
// code-review finding: renderer.go emitted `s-<script>` but no `as=` hook, so
// PDF's as=translation Body->Translation swap (SPECS §4.1) silently failed
// to mirror into EPUB (questions' wrapper class is always "questions"
// regardless of as=). The wrapper now also carries an "as-<value>" token
// (SPECS §7.1) so component CSS can apply the same swap; as=source (default
// or explicit) emits no as-* token at all.
func TestQuestions_AsTranslation_EPUBClassToken(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSubstr string
		wantAbsent string
	}{
		{
			name:       "as=translation emits as-translation token",
			input:      "{start-questions as=translation}\nQ1 = A1\n{end-questions}\n",
			wantSubstr: `<div class="questions as-translation" dir="ltr">`,
		},
		{
			name:       "as omitted (default source) emits no as-* token",
			input:      "{start-questions}\nQ1 = A1\n{end-questions}\n",
			wantAbsent: "as-",
		},
		{
			name:       "as=source (explicit) emits no as-* token",
			input:      "{start-questions as=source}\nQ1 = A1\n{end-questions}\n",
			wantAbsent: "as-",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := markdown.ToHTML([]byte(tc.input))
			if err != nil {
				t.Fatalf("ToHTML() unexpected error: %v", err)
			}
			if tc.wantSubstr != "" && !strings.Contains(string(got), tc.wantSubstr) {
				t.Fatalf("ToHTML() = %q, want substring %q", string(got), tc.wantSubstr)
			}
			if tc.wantAbsent != "" && strings.Contains(string(got), tc.wantAbsent) {
				t.Fatalf("ToHTML() = %q, want NO substring %q", string(got), tc.wantAbsent)
			}
		})
	}
}
