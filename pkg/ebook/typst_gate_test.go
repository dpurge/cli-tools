package ebook

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// This file holds the FIX-3(a) regression tests (font-config fix pass,
// code review 2026-07-31) for SPECS §8.4's per-block/per-column
// Strong/Emphasis gate: a bold run must resolve the SAME script/family as
// the REGULAR content around it, never a script "borrowed" from a
// different field/column in the same block, and never the enclosing
// book-level large-script substitute when this field's OWN resolved
// script isn't large.
//
// These compile a small, self-contained .typ document built from the REAL
// embedded `bookTemplate` (templates/book.typ, byte-identical to what
// ships) plus a minimal test-only harness that seeds the same states
// book() would have populated (bypassing its full page/document setup,
// which is irrelevant to font resolution) and calls the public block
// functions (textblock/parallel) directly.
//
// Verifying the actually-applied font requires more than "compiles clean":
// a `#metadata(text.font) <label>` marker is embedded INSIDE each bold
// run's own content, so it is a descendant of the show-rule's
// `text(font: resolvedFont, it.body)` call -- a `context text.font` read
// there reflects the font ACTUALLY applied to that specific run, extracted
// via `typst query <label> --field value`. (This was verified against
// Typst 0.15.1 in isolation before relying on it here: a bare `it`
// passthrough in a show rule's "else" branch is STILL visible to an
// enclosing show rule of the same selector, which is exactly the second,
// deeper defect these tests also guard against -- gating on the right
// script alone is not sufficient if the non-large branch doesn't finalize.)

// bookTypGateFixtureHeader is appended directly after the real bookTemplate
// (never modifies it). It installs a literal, distinctive stand-in for
// book()'s own book-level Strong gate (mimicking an actual large-script
// book, e.g. project.Script=arab -> large-script:true) and seeds
// _fontSlots/_roleFonts with distinctive synthetic family names so each
// resolution path is unambiguously identifiable in the query output.
const bookTypGateFixtureHeader = `
// --- test-only harness below (not part of the shipped book.typ) ---
#let _testBookStrongFont = ("BOOKLEVELSTRONGFONT",)
#show strong: it => context text(font: _testBookStrongFont, weight: "regular", it.body)

#_fontSlots.update((strong: "QUALIFIEDSTRONGFONT", translation: "TRANSFONT", body: "BODYFONT"))
#_roleFonts.update((
  body: ("BODYFONT",), header: ("HEADERFONT",), transcription: ("TRANSFONT",),
  translation: ("TRANSFONT",), strong: ("ROLESTRONGFONT",), emph: ("ROLEEMPHFONT",),
))
`

// compileBookTypGateFixture writes bookTemplate + the harness + body to a
// temp .typ file and compiles it, failing the test on any compile ERROR
// (warnings -- e.g. "unknown font family" for these deliberately-fake
// synthetic names -- are expected and ignored). Skips cleanly when no
// typst binary is available, mirroring every other real-typst test in this
// package. Returns the resolved typst binary path and the fixture's .typ
// path, ready for `typst query`.
func compileBookTypGateFixture(t *testing.T, body string) (typstPath, typPath string) {
	t.Helper()
	typstPath, err := locateTypst()
	if err != nil {
		t.Skipf("typst binary not available: %v", err)
	}

	dir := t.TempDir()
	typPath = filepath.Join(dir, "fixture.typ")
	pdfPath := filepath.Join(dir, "fixture.pdf")
	src := bookTemplate + bookTypGateFixtureHeader + "\n" + body + "\n"
	if err := os.WriteFile(typPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if out, err := runTypst(typstPath, "compile", typPath, pdfPath); err != nil {
		t.Fatalf("typst compile failed: %s", out)
	}
	return typstPath, typPath
}

// typstQueryFirstFamily runs `typst query <typPath> <selector> --field
// value` and returns the FIRST family name from the captured `text.font`
// metadata (which may itself be a single string or a font-stack array,
// depending on how the resolved font was shaped) -- the winning, most
// specific family either way. Fails the test if the selector matched
// nothing (the marker never rendered) or the value has an unexpected shape.
func typstQueryFirstFamily(t *testing.T, typstPath, typPath, selector string) string {
	t.Helper()
	out, err := exec.Command(typstPath, "query", typPath, selector, "--field", "value").Output()
	if err != nil {
		t.Fatalf("typst query %s %s: %v", typPath, selector, err)
	}
	var results []json.RawMessage
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("typst query %s: parse %q: %v", selector, out, err)
	}
	if len(results) == 0 {
		t.Fatalf("typst query %s: no results -- metadata marker never rendered", selector)
	}
	var single string
	if err := json.Unmarshal(results[0], &single); err == nil {
		return single
	}
	var stack []string
	if err := json.Unmarshal(results[0], &stack); err == nil && len(stack) > 0 {
		return stack[0]
	}
	t.Fatalf("typst query %s: unexpected value shape %s", selector, results[0])
	return ""
}

// TestBookTyp_TranslationRoleBoldStaysInFamilyScript is the FIX-3(a)
// regression test for the code-review defect (SPECS §8.4): a
// {start-text as=translation} block's bold content must resolve the SAME
// (Translation/base) family as its own regular content, even inside a
// large-script (here: Arabic) book -- NOT the book-level large-script
// Strong substitute. This exercises the REAL, unmodified textblock()
// function from templates/book.typ; script:"arab" on the call represents a
// Polish/Latin translation column embedded in an Arabic reader (Major-2).
func TestBookTyp_TranslationRoleBoldStaysInFamilyScript(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#textblock(role: "translation", script: "arab", dir: ltr, [
  Plain #strong[BOLD#context [#metadata(text.font) <text-bold>]] text.
])
`)
	got := typstQueryFirstFamily(t, typstPath, typPath, "<text-bold>")
	if got != "transfont" {
		t.Errorf("translation-role bold resolved font = %q, want %q (must match the field's OWN Translation/base family, not the book-level large-script Strong substitute)", got, "transfont")
	}
}

// TestBookTyp_ParallelMainBoldStaysBookLevel is the FIX-3(a) regression test
// for the parallel per-column gate defect (SPECS §8.4): a bold word in the
// MAIN column must NOT pick up the SECONDARY column's (foreign-script)
// Strong substitute -- main has no per-column override at all (by design,
// "parallel main -> book") and falls through to book()'s own book-level
// gate, while secondary (genuinely large script here) DOES substitute via
// its own qualified slot, and the two must differ.
func TestBookTyp_ParallelMainBoldStaysBookLevel(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#parallel(script: "arab", (
  main: [Plain #strong[BOLD#context [#metadata(text.font) <main-bold>]] text.],
  secondary: [Plain #strong[BOLD#context [#metadata(text.font) <sec-bold>]] text.],
),)
`)
	mainGot := typstQueryFirstFamily(t, typstPath, typPath, "<main-bold>")
	secGot := typstQueryFirstFamily(t, typstPath, typPath, "<sec-bold>")

	if mainGot != "booklevelstrongfont" {
		t.Errorf("parallel MAIN column bold resolved font = %q, want %q (book-level gate, unaffected by the secondary column's own show rule)", mainGot, "booklevelstrongfont")
	}
	if secGot != "qualifiedstrongfont" {
		t.Errorf("parallel SECONDARY column bold resolved font = %q, want %q (its own large-script Strong slot)", secGot, "qualifiedstrongfont")
	}
	if mainGot == secGot {
		t.Errorf("parallel main and secondary bold resolved to the SAME family %q -- the per-column gate scoping (SPECS §8.4) isn't actually isolating the two columns", mainGot)
	}
}

// TestBookTyp_VocabularyPhraseBoldFinalizes is the regression test for the
// residual limitation flagged at the end of the fix-pass impl log (SPECS
// §8.4): vocabulary's (and models', same pattern) phrase field always bolds
// itself programmatically -- there is no `show strong` rule to gate, just an
// inline `if _isLargeScript(script) { ... } else { strong(...) } }` choice.
// A NON-large-script phrase (here: script="latn") nested in a book that
// mimics a large-script book-level Strong gate must keep the AMBIENT font
// (i.e. render as plain bold, unmodified) -- a bare `strong(...)` call is
// NOT finalized and remains visible to the enclosing book-level `show
// strong` rule, which would otherwise re-substitute the book's Strong font
// onto a phrase whose own script isn't large at all (same defect class as
// FIX 1, reached via a bare strong() call instead of an unfinalized show-
// rule "else" branch).
func TestBookTyp_VocabularyPhraseBoldFinalizes(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(font: ("AMBIENTFONT",))
#vocabulary(script: "latn",
  (phrase: [BOLD#context [#metadata(text.font) <vocab-phrase-bold>]], grammar: "", transcription: "", translation: ""),
)
`)
	got := typstQueryFirstFamily(t, typstPath, typPath, "<vocab-phrase-bold>")
	if got != "ambientfont" {
		t.Errorf("vocabulary phrase (non-large script) bold resolved font = %q, want %q (must stay the ambient font -- a bare strong() call is not finalized and would otherwise be re-substituted by the enclosing book-level large-script Strong gate)", got, "ambientfont")
	}
}

// TestBookTyp_ModelsPhraseBoldFinalizes mirrors
// TestBookTyp_VocabularyPhraseBoldFinalizes above for models(), which has
// the identical inline (non-show-rule) phrase-bolding pattern.
func TestBookTyp_ModelsPhraseBoldFinalizes(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(font: ("AMBIENTFONT",))
#models(script: "latn",
  (phrase: [BOLD#context [#metadata(text.font) <models-phrase-bold>]], transcription: "", translation: ""),
)
`)
	got := typstQueryFirstFamily(t, typstPath, typPath, "<models-phrase-bold>")
	if got != "ambientfont" {
		t.Errorf("models phrase (non-large script) bold resolved font = %q, want %q (must stay the ambient font -- a bare strong() call is not finalized and would otherwise be re-substituted by the enclosing book-level large-script Strong gate)", got, "ambientfont")
	}
}
