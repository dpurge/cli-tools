package markdown_test

import (
	"strings"
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

// TestDialog_As_Unification covers SPECS §5's as= unification for
// {start-dialog}: source (default, implicit) and translation are accepted;
// any other in-grammar value (transcription/grammar, valid on {start-text}
// but not here) is a build error naming dialog's accepted set.
func TestDialog_As_Unification(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // substring; empty means no error expected
	}{
		{
			name:  "as=source accepted (explicit)",
			input: "{start-dialog as=source}\n--:\n  Hi.\n{end-dialog}\n",
		},
		{
			name:  "as=translation accepted",
			input: "{start-dialog as=translation}\n--:\n  Hi.\n{end-dialog}\n",
		},
		{
			name:  "as omitted defaults to source (no error)",
			input: "{start-dialog}\n--:\n  Hi.\n{end-dialog}\n",
		},
		{
			name:    "as=transcription rejected with dialog's accepted set named",
			input:   "{start-dialog as=transcription}\n--:\n  Hi.\n{end-dialog}\n",
			wantErr: `as="transcription" is not valid on {start-dialog}: must be source|translation`,
		},
		{
			name:    "as=grammar rejected with dialog's accepted set named",
			input:   "{start-dialog as=grammar}\n--:\n  Hi.\n{end-dialog}\n",
			wantErr: `as="grammar" is not valid on {start-dialog}: must be source|translation`,
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

// TestDialog_AsTranslation_EPUBClassToken covers the FIX-2/ASR-4 code-review
// finding: renderer.go emitted `s-<script>` but no `as=` hook, so PDF's
// as=translation Body->Translation swap (SPECS §4.1) silently failed to
// mirror into EPUB (dialog's wrapper class is always "dialog" regardless of
// as=, unlike text's class-per-role). The wrapper now also carries an
// "as-<value>" token (SPECS §7.1) so component CSS can apply the same swap;
// as=source (default or explicit) emits no as-* token at all.
func TestDialog_AsTranslation_EPUBClassToken(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSubstr string
		wantAbsent string
	}{
		{
			name:       "as=translation emits as-translation token",
			input:      "{start-dialog as=translation}\n--:\n  Hi.\n{end-dialog}\n",
			wantSubstr: `<div class="dialog as-translation" dir="ltr">`,
		},
		{
			name:       "as omitted (default source) emits no as-* token",
			input:      "{start-dialog}\n--:\n  Hi.\n{end-dialog}\n",
			wantAbsent: "as-",
		},
		{
			name:       "as=source (explicit) emits no as-* token",
			input:      "{start-dialog as=source}\n--:\n  Hi.\n{end-dialog}\n",
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
