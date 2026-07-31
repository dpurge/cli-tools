package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestBug5_RTLContainerLTRMetadataFields covers bug #5: in RTL-script blocks
// (script=arab/hebr/syrc), the container div must carry dir="rtl" (D9), but
// vocabulary grammar/transcription/translation spans and models
// transcription/translation spans must carry dir="ltr" (bug #5 fix) so that
// Latin metadata fields are not forced into the RTL visual order.
func TestBug5_RTLContainerLTRMetadataFields(t *testing.T) {
	t.Run("vocab arab-script: container rtl, metadata spans ltr (HTML)", func(t *testing.T) {
		input := "{start-vocabulary lang=arb script=arab}\nكتاب {noun} [kitaab] = book\n{end-vocabulary}\n"
		got, err := markdown.ToHTML([]byte(input))
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}
		out := string(got)

		// Container must be RTL (D9); also carries the new s-arab class
		// token (SPECS §7.1, INC3), sourced from the block's own script.
		if !strings.Contains(out, `<div class="vocabulary s-arab" dir="rtl">`) {
			t.Errorf("expected RTL container div, got:\n%s", out)
		}
		// Grammar/transcription/translation spans must remain LTR (bug #5).
		if !strings.Contains(out, `<span class="vocabulary-grammar" dir="ltr">`) {
			t.Errorf("expected LTR grammar span, got:\n%s", out)
		}
		if !strings.Contains(out, `<span class="vocabulary-transcription" dir="ltr">`) {
			t.Errorf("expected LTR transcription span, got:\n%s", out)
		}
		if !strings.Contains(out, `<span class="vocabulary-translation" dir="ltr">`) {
			t.Errorf("expected LTR translation span, got:\n%s", out)
		}
	})

	t.Run("models arab-script: container rtl, transcription/translation spans ltr (HTML)", func(t *testing.T) {
		input := "{start-models lang=arb script=arab}\nكتب [kataba] = he wrote\n{end-models}\n"
		got, err := markdown.ToHTML([]byte(input))
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}
		out := string(got)

		// Container must be RTL (D9); also carries the new s-arab class
		// token (SPECS §7.1, INC3), sourced from the block's own script.
		if !strings.Contains(out, `<div class="models s-arab" dir="rtl">`) {
			t.Errorf("expected RTL models container div, got:\n%s", out)
		}
		// Transcription and translation spans must remain LTR (bug #5).
		if !strings.Contains(out, `<span class="models-transcription" dir="ltr">`) {
			t.Errorf("expected LTR transcription span, got:\n%s", out)
		}
		if !strings.Contains(out, `<span class="models-translation" dir="ltr">`) {
			t.Errorf("expected LTR translation span, got:\n%s", out)
		}
	})

	t.Run("vocab arab-script: Typst emits dir: rtl", func(t *testing.T) {
		input := "{start-vocabulary lang=arb script=arab}\nكتاب {noun} [kitaab] = book\n{end-vocabulary}\n"
		got, err := markdown.ToTypst([]byte(input))
		if err != nil {
			t.Fatalf("ToTypst() unexpected error: %v", err)
		}
		if !strings.Contains(string(got), "#vocabulary(dir: rtl,") {
			t.Errorf("expected Typst vocabulary call with dir: rtl, got:\n%s", string(got))
		}
	})

	t.Run("models arab-script: Typst emits dir: rtl", func(t *testing.T) {
		input := "{start-models lang=arb script=arab}\nكتب [kataba] = he wrote\n{end-models}\n"
		got, err := markdown.ToTypst([]byte(input))
		if err != nil {
			t.Fatalf("ToTypst() unexpected error: %v", err)
		}
		if !strings.Contains(string(got), "#models(dir: rtl,") {
			t.Errorf("expected Typst models call with dir: rtl, got:\n%s", string(got))
		}
	})

	t.Run("ltr-script block still gets dir: ltr (regression guard)", func(t *testing.T) {
		input := "{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n"
		got, err := markdown.ToHTML([]byte(input))
		if err != nil {
			t.Fatalf("ToHTML() unexpected error: %v", err)
		}
		if !strings.Contains(string(got), `<div class="vocabulary" dir="ltr">`) {
			t.Errorf("expected LTR container for default (no-script) block, got:\n%s", string(got))
		}
	})
}
