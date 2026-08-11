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
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "LTR: anonymous and named turns with multi-paragraph content",
			input: "{start-dialog}\n" +
				"--:\n" +
				"  Hello there.\n" +
				"\n" +
				"  Second **bold** paragraph.\n" +
				"@Bob:\n" +
				"  Hi!\n" +
				"＠李:\n" +
				"  你好!\n" +
				"{end-dialog}\n",
			want: "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
				"<div class=\"dialog\" dir=\"ltr\">\n" +
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
				"</div>\n",
		},
		{
			// RTL integration golden: script=arab → blockDirection("arab")="rtl"
			// → badgeOnlyHTML("D","rtl") emits dir="rtl" on the badge div and
			// the dialog wrapper carries dir="rtl" + s-arab class (NFR-4).
			name:  "script=arab: RTL badge and wrapper",
			input: "{start-dialog script=arab}\n--:\n  مرحبا.\n{end-dialog}\n",
			want: "<div class=\"block-marker\" dir=\"rtl\"><span class=\"ct-badge\">D</span></div>\n" +
				"<div class=\"dialog s-arab\" dir=\"rtl\">\n" +
				"<div class=\"dialog-item\">\n" +
				"<div class=\"dialog-header\">—</div>\n" +
				"<div class=\"dialog-content\"><p>مرحبا.</p>\n" +
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

// ---------------------------------------------------------------------
// SPECS §12.1 — Parser round-trip (verified through HTML rendering)
// ---------------------------------------------------------------------

// TestParseDialogHeaderNoLongerErrors asserts that an un-indented ## Heading
// line inside a dialog block is now recognised as a block-level ItemHeader and
// does NOT return the old indentation error (SPECS §12.1, ASR-7).
func TestParseDialogHeaderNoLongerErrors(t *testing.T) {
	input := "{start-dialog}\n--:\n  Hello.\n## Section\n@Bob:\n  Hi!\n{end-dialog}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">—</div>\n" +
		"<div class=\"dialog-content\"><p>Hello.</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"<h2>Section</h2>\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">Bob:</div>\n" +
		"<div class=\"dialog-content\"><p>Hi!</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error (was error before the fix): %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
}

// TestParseDialogNoteStripped asserts that an un-indented (Formal greeting)
// line in dialog is parsed as ItemNote with the parentheses stripped (D5), and
// rendered as <p class="block-note">Formal greeting</p> (SPECS §12.1, §6).
func TestParseDialogNoteStripped(t *testing.T) {
	input := "{start-dialog}\n--:\n  Hello.\n(Formal greeting)\n{end-dialog}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">—</div>\n" +
		"<div class=\"dialog-content\"><p>Hello.</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"<p class=\"block-note\">Formal greeting</p>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
	}
	// D5: parentheses must not appear in rendered output.
	if strings.Contains(string(got), "(Formal greeting)") {
		t.Fatalf("ToHTML() output contains un-stripped parentheses (D5 violated): %q", string(got))
	}
}

// TestParseDialogIndentedHashIsContent asserts that a 2-space-indented
// "  ## x" line is treated as turn CONTENT, not a block-level header
// (SPECS §12.1, §3.3). The ## x string is the content of the dialog turn
// and recurses through the full HTML renderer (ToHTML), which adds goldmark's
// normal heading id="x" — this is inside dialog-content, NOT a bare
// block-level <h2> emitted by our custom block-header path (ASR-6 does not
// apply here because this path is standard goldmark, not our ItemHeader emitter).
func TestParseDialogIndentedHashIsContent(t *testing.T) {
	input := "{start-dialog}\n--:\n  ## x\n{end-dialog}\n"
	// Goldmark adds id="x" to the heading inside dialog-content — that is the
	// standard heading behavior, distinct from the no-id block-level ItemHeader
	// emitter in renderDialog (ASR-6 applies to ItemHeader only).
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">—</div>\n" +
		"<div class=\"dialog-content\"><h2 id=\"x\">x</h2>\n" +
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
	// Key assertion: the <h2> is INSIDE dialog-content, not a bare block-level
	// element. Verify that the block-level rendering produces NO bare <h2>
	// outside dialog-item (the only <h2> must be inside dialog-content).
	if strings.Contains(string(got), "</div>\n<h2") {
		t.Fatalf("ToHTML() produced a block-level <h2> outside dialog-item for an indented line: %q", string(got))
	}
}

// TestParseDialogMalformedStillErrors asserts that an un-indented line that is
// NEITHER a recognised block header, a note, NOR a dialog turn marker still
// returns the indentation error (SPECS §12.1, ASR-7 — the error removal is
// deliberate only for header/note lines).
func TestParseDialogMalformedStillErrors(t *testing.T) {
	input := "{start-dialog}\n@Bob:\nBadly indented line\n{end-dialog}\n"
	_, err := markdown.ToHTML([]byte(input))
	if err == nil {
		t.Fatalf("ToHTML() expected an indentation error for a malformed dialog line, got nil")
	}
	wantErr := "Wrong line indentation for dialog item: Badly indented line"
	if err.Error() != wantErr {
		t.Fatalf("ToHTML() error = %q, want %q", err.Error(), wantErr)
	}
}

// TestParseDialogHeaderInterleavedBetweenTurns is the F5/F6 test case.
// It verifies correct positional interleaving of a block header between two
// speaker turns:
//
//	(a) speaker1's turn flushes correctly as a DialogItem before the header,
//	(b) ItemHeader lands at the correct position between the two turns,
//	(c) speaker2's turn parses cleanly afterward with the correct speaker
//	    (no stale-header bleed — verifies the header="" reset, SPECS §5 F6).
func TestParseDialogHeaderInterleavedBetweenTurns(t *testing.T) {
	input := "{start-dialog}\n@Alice:\n  Good morning!\n## Greetings\n@Bob:\n  Hello!\n{end-dialog}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		// (a) Alice's turn appears first, correctly attributed.
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">Alice:</div>\n" +
		"<div class=\"dialog-content\"><p>Good morning!</p>\n" +
		"</div>\n" +
		"</div>\n" +
		// (b) Block header lands between the two turns.
		"<h2>Greetings</h2>\n" +
		// (c) Bob's turn uses "Bob:" not "Alice:" — no stale-header bleed.
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">Bob:</div>\n" +
		"<div class=\"dialog-content\"><p>Hello!</p>\n" +
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

// TestParseDialogHeaderResetsStaleSpeaker exercises the exact failure mode
// the header="" reset (parser.go, SPECS §5 F6) exists to prevent: a
// "@Speaker:" line sets the pending header BEFORE any turn content is
// buffered, a block header interrupts immediately (flush() is a no-op since
// buf is empty), and indented content follows with no new "@speaker:" line
// before it. Without the reset, that orphaned content would be silently
// misattributed to the stale "Alice:" speaker at the trailing flush().
//
// Unlike TestParseDialogHeaderInterleavedBetweenTurns (whose input already
// has content buffered for Alice before the header, so flush() fires and
// harmlessly re-sets header via the following "@Bob:" line regardless of the
// header="" reset), this input reaches the reset with buf still empty — the
// only path that can actually distinguish "reset" from "not reset".
func TestParseDialogHeaderResetsStaleSpeaker(t *testing.T) {
	input := "{start-dialog}\n@Alice:\n## Interlude\n  Orphaned content, no new speaker line.\n{end-dialog}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		"<h2>Interlude</h2>\n" +
		// If header were NOT reset, this would incorrectly read "Alice:".
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\"></div>\n" +
		"<div class=\"dialog-content\"><p>Orphaned content, no new speaker line.</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch (header=\"\" reset regression)\n got: %q\nwant: %q", string(got), want)
	}
}

// ---------------------------------------------------------------------
// SPECS §12.2 — HTML render
// ---------------------------------------------------------------------

// TestRenderDialogNoteHTML asserts that a dialog note item renders as
// <p class="block-note">…</p> with the inner text written raw, no parens
// (SPECS §12.2, §6).
func TestRenderDialogNoteHTML(t *testing.T) {
	input := "{start-dialog}\n--:\n  Hello.\n(A greeting)\n{end-dialog}\n"
	want := "<div class=\"block-marker\"><span class=\"ct-badge\">D</span></div>\n" +
		"<div class=\"dialog\" dir=\"ltr\">\n" +
		"<div class=\"dialog-item\">\n" +
		"<div class=\"dialog-header\">—</div>\n" +
		"<div class=\"dialog-content\"><p>Hello.</p>\n" +
		"</div>\n" +
		"</div>\n" +
		"<p class=\"block-note\">A greeting</p>\n" +
		"</div>\n"
	got, err := markdown.ToHTML([]byte(input))
	if err != nil {
		t.Fatalf("ToHTML() unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("ToHTML() mismatch\n got: %q\nwant: %q", string(got), want)
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
