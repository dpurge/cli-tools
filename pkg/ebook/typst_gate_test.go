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

// TestBookTyp_ParallelSourceBoldUsesGate is the updated ASR-1/ASR-7 regression
// test for parallel per-column gate isolation (parallel-lang-script SPECS):
// after the column-semantics reversal (ASR-1), the SOURCE column now carries
// the per-column large-script strong/emph gate (keyed on the marker script),
// while the TRANSLATION column has no per-column gate and falls through to
// book()'s book-level gate. The two columns must resolve to different font
// families to prove the gate is actually scoped to source and not leaking
// into translation (ASR-8). Previously this test used the old "main" and
// "secondary" dict keys; updated to "source" and "translation" as part of the
// ASR-1 column-semantics reversal (SPECS §7.2, S5 compile-gate requirement).
// NOTE: `source-dir` is deliberately omitted here (defaults to ltr) even
// though script:"arab" would normally want rtl — this test only checks
// which font FAMILY the strong gate resolves to, which direction does not
// affect. Don't copy this call pattern for a test that cares about layout.
func TestBookTyp_ParallelSourceBoldUsesGate(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#parallel(script: "arab", (
  source: [Plain #strong[BOLD#context [#metadata(text.font) <source-bold>]] text.],
  translation: [Plain #strong[BOLD#context [#metadata(text.font) <trans-bold>]] text.],
),)
`)
	sourceGot := typstQueryFirstFamily(t, typstPath, typPath, "<source-bold>")
	transGot := typstQueryFirstFamily(t, typstPath, typPath, "<trans-bold>")

	if sourceGot != "qualifiedstrongfont" {
		t.Errorf("parallel SOURCE column bold resolved font = %q, want %q (source has its own large-script gate keyed on the marker script, ASR-7)", sourceGot, "qualifiedstrongfont")
	}
	if transGot != "booklevelstrongfont" {
		t.Errorf("parallel TRANSLATION column bold resolved font = %q, want %q (translation has no per-column gate, falls through to book-level, ASR-1)", transGot, "booklevelstrongfont")
	}
	if sourceGot == transGot {
		t.Errorf("parallel source and translation bold resolved to the SAME family %q -- the source gate is not actually isolating (SPECS ASR-7/ASR-8)", sourceGot)
	}
}

// TestBookTyp_ParallelTranscriptionBoldNotCapturedBySourceGate is the ASR-8
// compile gate: bold content in the stacked TRANSCRIPTION cell must NOT be
// captured by the SOURCE column's per-column large-script gate — the
// transcription is rendered outside the source's inner show-rule scope (SPECS
// §7.2, ASR-8). Prove via font: source bold uses the scoped gate
// ("qualifiedstrongfont"), transcription bold falls through to book-level
// ("booklevelstrongfont") — different families confirm gate isolation.
// Also confirms a 3-field parallel row compiles cleanly (SPECS §12.4).
// NOTE: `source-dir` is deliberately omitted (see TestBookTyp_ParallelSourceBoldUsesGate) — this test checks font resolution only.
func TestBookTyp_ParallelTranscriptionBoldNotCapturedBySourceGate(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#parallel(script: "arab", (
  source: [Plain #strong[BOLD#context [#metadata(text.font) <source-bold2>]] text.],
  translation: [Translation text.],
  transcription: [#strong[BOLD#context [#metadata(text.font) <transcription-bold>]] roman.],
),)
`)
	sourceGot := typstQueryFirstFamily(t, typstPath, typPath, "<source-bold2>")
	transcriptionGot := typstQueryFirstFamily(t, typstPath, typPath, "<transcription-bold>")

	if sourceGot != "qualifiedstrongfont" {
		t.Errorf("parallel SOURCE column bold resolved font = %q, want %q (source gate active, marker script=arab)", sourceGot, "qualifiedstrongfont")
	}
	if transcriptionGot != "booklevelstrongfont" {
		t.Errorf("parallel TRANSCRIPTION bold resolved font = %q, want %q (must fall through to book-level gate, NOT captured by source gate, ASR-8)", transcriptionGot, "booklevelstrongfont")
	}
	if sourceGot == transcriptionGot {
		t.Errorf("source and transcription bold resolved to SAME family %q -- source gate is bleeding into the transcription cell (ASR-8 isolation broken)", sourceGot)
	}
}

// TestBookTyp_ParallelNoTranscriptionKeyCompiles verifies that a 1-field and
// 2-field parallel row (no "transcription" key) compile clean and emit no
// stray linebreak — the "transcription" in r dict-key check correctly skips
// the block when the key is absent (SPECS §7.2, §12.4).
func TestBookTyp_ParallelNoTranscriptionKeyCompiles(t *testing.T) {
	compileBookTypGateFixture(t, `
#parallel(script: "latn", (source: [One-field row],),)
#parallel(script: "latn", (source: [Source.], translation: [Translation.]),)
`)
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

// --- script-size-markers feature gate tests (SPECS F-SIZE / F-MARK) --------

// typstQueryBool reads a boolean `#metadata(...)` value (typst query --field
// value) at the given label, mirroring typstQueryFirstFamily's shape handling
// for a single JSON bool. Used to assert an APPLIED text.size relation from
// inside a specific run without depending on the pt serialization format.
func typstQueryBool(t *testing.T, typstPath, typPath, selector string) bool {
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
	var b bool
	if err := json.Unmarshal(results[0], &b); err != nil {
		t.Fatalf("typst query %s: value %s not a bool: %v", selector, results[0], err)
	}
	return b
}

// TestBookTyp_BadgeCompiles proves the content-type badge (SPECS FR-5) and the
// exact Typst shapes the renderer now emits (SPECS FR-7) compile clean against
// the REAL bookTemplate: the `_ctbadge` box (context + header-font pin), a
// badge-only block (ltr and rtl align(right)), and a badge injected into a
// heading. Compilation itself is the assertion (compileBookTypGateFixture
// t.Fatalf's on any error); a warning for the fake seeded font name is ignored.
func TestBookTyp_BadgeCompiles(t *testing.T) {
	compileBookTypGateFixture(t, `
#set text(size: 10pt)
#_ctbadge("T")
#block(above: 1.2em, below: 0.5em)[#_ctbadge("V")]
#block(above: 1.2em, below: 0.5em, align(right)[#_ctbadge("D")])
= #_ctbadge("T") Heading title
`)
}

// TestBookTyp_ForeignScriptEnlargedWhenBookNotLarge is the FR-2/FR-3 regression
// test: with _sizeFactor 1.5 (a non-large-script book, e.g. project.Script=latn
// with hans blocks), a large-script (hans) vocabulary phrase enlarges above the
// 10pt base while a non-large (latn) phrase stays at base. A 12pt threshold
// discriminates the two (15pt vs 10pt) robustly.
func TestBookTyp_ForeignScriptEnlargedWhenBookNotLarge(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(size: 10pt)
#_sizeFactor.update(1.5)
#vocabulary(script: "hans",
  (phrase: [P#context [#metadata(text.size > 12pt) <hans-size>]], grammar: "", transcription: "", translation: ""),
)
#vocabulary(script: "latn",
  (phrase: [P#context [#metadata(text.size < 12pt) <latn-size>]], grammar: "", transcription: "", translation: ""),
)
`)
	if !typstQueryBool(t, typstPath, typPath, "<hans-size>") {
		t.Errorf("hans (large-script) vocabulary phrase was NOT enlarged (want applied text.size > 12pt at _sizeFactor 1.5 over a 10pt base)")
	}
	if !typstQueryBool(t, typstPath, typPath, "<latn-size>") {
		t.Errorf("latn (non-large) vocabulary phrase must stay at base (want applied text.size < 12pt)")
	}
}

// TestBookTyp_ForeignScriptNotDoubleEnlargedWhenBookLarge is the FR-2 no-
// regression guard: a whole-book large-script book sets _sizeFactor 1.0 (its
// base is ALREADY size-large), so per-block enlargement is a no-op — a hans
// phrase must stay at base, never double-enlarge.
func TestBookTyp_ForeignScriptNotDoubleEnlargedWhenBookLarge(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(size: 10pt)
#_sizeFactor.update(1.0)
#vocabulary(script: "hans",
  (phrase: [P#context [#metadata(text.size < 12pt) <hans-size-1x>]], grammar: "", transcription: "", translation: ""),
)
`)
	if !typstQueryBool(t, typstPath, typPath, "<hans-size-1x>") {
		t.Errorf("hans vocabulary phrase double-enlarged when the book is already large-script (want applied text.size < 12pt at _sizeFactor 1.0 over a 10pt base)")
	}
}

// TestBookTyp_AsTranslationNotEnlarged verifies the FR-3 / FR-4 `as=translation`
// gate on the Typst side (the mirror of CSS's `.s-hans:not(.as-translation)`):
// an as=translation block resolves familyScript="" so `_foreignSize` returns 1em
// even when the block's OWN script is large (hans) — the translated content
// stays at base. A companion as=source dialog in the SAME large script DOES
// enlarge, proving the gate discriminates on `as=`, not on the raw script.
func TestBookTyp_AsTranslationNotEnlarged(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(size: 10pt)
#_sizeFactor.update(1.5)
#dialog(script: "hans", role: "translation",
  (header: "H", content: [C#context [#metadata(text.size < 12pt) <trans-content>]]),
)
#dialog(script: "hans", role: "source",
  (header: "H", content: [C#context [#metadata(text.size > 12pt) <source-content>]]),
)
`)
	if !typstQueryBool(t, typstPath, typPath, "<trans-content>") {
		t.Errorf("as=translation dialog content was enlarged despite familyScript=\"\" (want base text.size < 12pt, mirroring CSS :not(.as-translation))")
	}
	if !typstQueryBool(t, typstPath, typPath, "<source-content>") {
		t.Errorf("as=source dialog content was NOT enlarged in a large (hans) script (want text.size > 12pt)")
	}
}

// --- structured-block-headers gate tests (SPECS §12.3 / PLAN S9) ----------

// sbFixtureHeader is the harness for structured-block-headers compile tests.
// Identical in shape to bookTypGateFixtureHeader above but extends _roleFonts
// with the notes role so _blocknote (which reads _roleFonts.get().notes) does
// not error at runtime with an undefined dictionary key.
const sbFixtureHeader = `
// --- structured-block-headers test harness ---
#_roleFonts.update((
  body: ("BODYFONT",), header: ("HEADERFONT",), transcription: ("TRANSFONT",),
  translation: ("TRANSFONT",), strong: ("ROLESTRONGFONT",), emph: ("ROLEEMPHFONT",),
  notes: ("NOTESFONT",),
))
`

// compileStructuredBlockFixture writes bookTemplate + sbFixtureHeader + body
// to a temp .typ file and compiles it with the real typst binary.
// Mirrors compileBookTypGateFixture but uses sbFixtureHeader (with the notes
// role) instead of bookTypGateFixtureHeader. Skips cleanly when no typst
// binary is available, matching every other real-typst test in this package.
// Returns the resolved typst binary path and the fixture .typ path, ready
// for `typst query`.
func compileStructuredBlockFixture(t *testing.T, body string) (typstPath, typPath string) {
	t.Helper()
	typstPath, err := locateTypst()
	if err != nil {
		t.Skipf("typst binary not available: %v", err)
	}

	dir := t.TempDir()
	typPath = filepath.Join(dir, "fixture.typ")
	pdfPath := filepath.Join(dir, "fixture.pdf")
	src := bookTemplate + sbFixtureHeader + "\n" + body + "\n"
	if err := os.WriteFile(typPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if out, err := runTypst(typstPath, "compile", typPath, pdfPath); err != nil {
		t.Fatalf("typst compile failed: %s", out)
	}
	return typstPath, typPath
}

// TestBookTyp_BlockHeaderNotInOutline is the §12.3 mandatory compile gate for
// structured-block-headers: a vocabulary block with a header item compiles
// clean AND the block heading element has outlined:false (D4 / ASR-6) — it
// must NOT appear in the document outline/TOC.
//
// A real document-level heading (= …) provides a positive control: it must
// be outlined:true so the test detects both wrong-count and wrong-boolean
// outcomes.
func TestBookTyp_BlockHeaderNotInOutline(t *testing.T) {
	typstPath, typPath := compileStructuredBlockFixture(t, `
= Document Heading
#vocabulary(script: "latn",
  (kind: "header", level: 2, text: "Block Section Header"),
  (phrase: "hello", grammar: "", transcription: "", translation: "world"),
)
`)
	// Query the outlined field for every heading element in document order.
	// Expect: [true, false] — document heading outlined, block heading not.
	out, err := exec.Command(typstPath, "query", typPath, "heading", "--field", "outlined").Output()
	if err != nil {
		t.Fatalf("typst query heading outlined: %v", err)
	}
	var outlined []bool
	if err := json.Unmarshal(out, &outlined); err != nil {
		t.Fatalf("parse typst query output %q: %v", out, err)
	}
	if len(outlined) != 2 {
		t.Fatalf("expected 2 headings (1 document + 1 block), got %d: %v", len(outlined), outlined)
	}
	if !outlined[0] {
		t.Errorf("document heading[0].outlined = false, want true (positive control)")
	}
	if outlined[1] {
		t.Errorf("block header[1].outlined = true, want false (ASR-6 / D4: block headers must NOT enter the document outline)")
	}
}

// TestBookTyp_BlockNoteUsesNotesFont is the §12.3 mandatory compile gate for
// the notes font role: _blocknote applies _roleFonts.get().notes to its
// content. A #metadata(text.font) marker embedded inside the note body is
// probed via typst query to confirm the ACTUALLY applied font matches the
// seeded notes role ("NOTESFONT") rather than any other role (body/emph/…).
func TestBookTyp_BlockNoteUsesNotesFont(t *testing.T) {
	typstPath, typPath := compileStructuredBlockFixture(t, `
#_blocknote[Test note #context [#metadata(text.font) <note-font>]]
`)
	got := typstQueryFirstFamily(t, typstPath, typPath, "<note-font>")
	// Typst lowercases font names in metadata output (mirrors existing gate tests).
	const want = "notesfont"
	if got != want {
		t.Errorf("block note applied font = %q, want %q (notes role seeded as NOTESFONT; ASR-5: emph fallback when font.css omits Font Notes)", got, want)
	}
}

// --- FR-6 / FR-5 outline-exclusion gate tests ---

// TestBookTyp_TextblockHeadingsNotInOutline is the FR-6 AC-1 compile gate:
// a heading rendered inside textblock() must have outlined:false — it must
// NOT appear in the document outline / TOC. A bare document-level heading
// provides a positive control (outlined:true) so the test detects both
// wrong-count and wrong-boolean outcomes.
//
// Implementation: textblock() now contains `show heading: set heading(outlined:
// false)` (T4, PLAN.md), scoped to the textblock body so it does NOT affect
// headings outside the block.
func TestBookTyp_TextblockHeadingsNotInOutline(t *testing.T) {
	typstPath, typPath := compileStructuredBlockFixture(t, `
= Document Heading
#textblock(role: "source", dir: ltr, script: "latn", [
== Inner Textblock Heading
Body text.
])
`)
	// Expect two headings: document heading (outlined:true), textblock heading (outlined:false).
	out, err := exec.Command(typstPath, "query", typPath, "heading", "--field", "outlined").Output()
	if err != nil {
		t.Fatalf("typst query heading outlined: %v", err)
	}
	var outlined []bool
	if err := json.Unmarshal(out, &outlined); err != nil {
		t.Fatalf("parse typst query output %q: %v", out, err)
	}
	if len(outlined) != 2 {
		t.Fatalf("expected 2 headings (1 document + 1 textblock), got %d: %v", len(outlined), outlined)
	}
	if !outlined[0] {
		t.Errorf("document heading[0].outlined = false, want true (positive control)")
	}
	if outlined[1] {
		t.Errorf("textblock heading[1].outlined = true, want false (FR-6 AC-1: textblock headings must NOT enter the outline)")
	}
}

// TestBookTyp_DialogHeaderStillNotInOutlineAfterFR5 is the FR-6 AC-2 regression
// gate: after the FR-5 change that wrapped _blockheading's output in align(center,
// heading(...)), dialog headers must still carry outlined:false. The align() wrapper
// must not inadvertently change the heading's outlined field.
func TestBookTyp_DialogHeaderStillNotInOutlineAfterFR5(t *testing.T) {
	typstPath, typPath := compileStructuredBlockFixture(t, `
= Document Heading
#dialog(dir: ltr, script: "latn", role: "source",
  (kind: "header", level: 2, text: "Dialog Section Header"),
  (header: "—", content: [Content.]),
)
`)
	// Expect two headings: document heading (outlined:true), dialog header (outlined:false).
	out, err := exec.Command(typstPath, "query", typPath, "heading", "--field", "outlined").Output()
	if err != nil {
		t.Fatalf("typst query heading outlined: %v", err)
	}
	var outlined []bool
	if err := json.Unmarshal(out, &outlined); err != nil {
		t.Fatalf("parse typst query output %q: %v", out, err)
	}
	if len(outlined) != 2 {
		t.Fatalf("expected 2 headings (1 document + 1 dialog header), got %d: %v", len(outlined), outlined)
	}
	if !outlined[0] {
		t.Errorf("document heading[0].outlined = false, want true (positive control)")
	}
	if outlined[1] {
		t.Errorf("dialog header[1].outlined = true, want false (FR-6 AC-2: align() wrap in FR-5 must not change outlined:false)")
	}
}

// --- FR-1 heading-size counter gate tests ---

// TestBookTyp_TranscriptionRoleHeadingNotShrunkenByCounter is the FR-1 AC-4
// compile gate: the heading-size counter
//
//	show heading: it => context text(size: 1em / _foreignSizeFactor(script), it)
//
// is scoped INSIDE textblock's source-role else branch only. A heading inside a
// transcription-role block must NOT be affected — those roles do not wrap their
// body in _foreignSize, so dividing by _foreignSizeFactor there would incorrectly
// shrink the heading.
//
// Setup: base 10pt, _sizeFactor=1.5. Correct transcription H2 = 1.4×10 = 14pt.
// Mis-scoped: (1.4/1.5)×10 ≈ 9.3pt. Threshold 12pt cleanly separates them.
func TestBookTyp_TranscriptionRoleHeadingNotShrunkenByCounter(t *testing.T) {
	// Mirror the book-level heading-level size rules: compileBookTypGateFixture
	// does not call book(), so these must be seeded explicitly. They are required
	// to match the thresholds in the comment above (H2 at 1.4em → 14pt correct,
	// 9.3pt mis-scoped). Without them headings are 1em = ambient, making both
	// paths resolve to 10pt (indistinguishable and both below 12pt — wrong).
	typstPath, typPath := compileBookTypGateFixture(t, `
#show heading.where(level: 1): set text(size: 1.7em)
#show heading.where(level: 2): set text(size: 1.4em)
#show heading.where(level: 3): set text(size: 1.2em)
#set text(size: 10pt)
#_sizeFactor.update(1.5)
#textblock(role: "transcription", dir: ltr, script: "arab", [
== H2 heading #context [#metadata(text.size > 12pt) <tc-h2-size>]
Body text.
])
`)
	if !typstQueryBool(t, typstPath, typPath, "<tc-h2-size>") {
		t.Errorf("FR-1 AC-4: transcription-role arab H2 was shrunken by the counter " +
			"(counter must only fire in source-role else branch; " +
			"mis-scoped shrinks from ~14pt to ~9.3pt — want >12pt)")
	}
}

// TestBookTyp_SourceRoleHeadingLevelsRemainDistinct is the FR-1 AC-5 compile
// gate: inside a source-role start-text block with an arab script, the
// heading-size counter normalises the ambient back to the base (undoing the
// _foreignSize enlargement) so that the book-level per-level size rules
// (H1=1.7em, H2=1.4em, H3=1.2em) still produce three distinct, correctly
// ordered sizes.
//
// Setup: base 10pt, _sizeFactor=1.5. Expected (counter correct):
//
//	H1 = 1.7×10 = 17pt  (> 16pt threshold)
//	H2 = 1.4×10 = 14pt  (> 13pt threshold)
//	H3 = 1.2×10 = 12pt  (< 13pt threshold)
func TestBookTyp_SourceRoleHeadingLevelsRemainDistinct(t *testing.T) {
	// Mirror the book-level heading-level size rules. The FR-1 counter
	// (show heading: it => context text(size: 1em/_foreignSizeFactor(script), it))
	// normalises the ambient back to the base text size. The level-specific
	// rules then apply their em multiplier on that normalised base:
	//   _foreignSize ambient (15pt) → counter → 10pt → level rule → 17/14/12pt.
	// Without these rules all headings resolve to 10pt (counter absorbs _foreignSize);
	// H1/H2/H3 become indistinguishable and all AC-5 assertions would fail.
	typstPath, typPath := compileBookTypGateFixture(t, `
#show heading.where(level: 1): set text(size: 1.7em)
#show heading.where(level: 2): set text(size: 1.4em)
#show heading.where(level: 3): set text(size: 1.2em)
#set text(size: 10pt)
#_sizeFactor.update(1.5)
#textblock(role: "source", dir: ltr, script: "arab", [
= H1 heading #context [#metadata(text.size > 16pt) <src-h1-large>]
== H2 heading #context [#metadata(text.size > 13pt) <src-h2-medium>]
=== H3 heading #context [#metadata(text.size < 13pt) <src-h3-small>]
Body text.
])
`)
	if !typstQueryBool(t, typstPath, typPath, "<src-h1-large>") {
		t.Errorf("FR-1 AC-5: H1 inside source-role arab textblock not large enough (want >16pt; counter may have homogenised heading levels)")
	}
	if !typstQueryBool(t, typstPath, typPath, "<src-h2-medium>") {
		t.Errorf("FR-1 AC-5: H2 inside source-role arab textblock not medium-sized (want >13pt; H2 must be larger than H3)")
	}
	if !typstQueryBool(t, typstPath, typPath, "<src-h3-small>") {
		t.Errorf("FR-1 AC-5: H3 inside source-role arab textblock not small enough (want <13pt; H3 must be smaller than H2)")
	}
}

// --- FR-1 AC-1/AC-2/AC-3 parity and scope gate tests ---

// TestBookTyp_FR1_TextblockDialogHeadingParity covers the three original FR-1
// acceptance criteria:
//
// AC-1 — parity: an Arab-script H2 inside a source-role textblock and an Arab-
// script (kind:"header") H2 inside dialog both render at the same size. Without
// the counter the textblock H2 sits at the _foreignSize-enlarged ambient (15pt)
// and gets Typst's built-in H2 default (1.2em) applied on top: 1.2×15pt = 18pt.
// With the counter the ambient is normalised back to 10pt before the default
// fires: 1.2×10pt = 12pt — identical to the dialog H2 which is NOT inside
// _foreignSize: 1.2×10pt = 12pt.
//
// AC-2 — scope: body text inside that same textblock IS still enlarged (15pt);
// the counter is a show-heading rule and must not affect body text.
//
// AC-3 — no-op for non-large scripts: with script="latn",
// _foreignSizeFactor("latn") = 1.0, so the counter divides by 1.0 = no change.
// Heading stays at 1.2×10pt = 12pt. If the implementation mistakenly divides by
// the raw _sizeFactor state value (1.5) instead of _foreignSizeFactor(script),
// the ambient shrinks from 10pt to 6.67pt and the heading becomes 1.2×6.67 ≈ 8pt.
//
// Setup: base 10pt, _sizeFactor=1.5. No book-level heading-size rules are added;
// Typst's built-in per-level default (1.2em for H2) drives the final size.
// Thresholds chosen to lie between the correct value and the buggy value:
//   AC-1/parity: correct=12pt, bug=18pt  → threshold < 15pt
//   AC-2/body:   correct=15pt            → threshold > 12pt
//   AC-3/latn:   correct=12pt, bug=~8pt  → threshold > 9pt
func TestBookTyp_FR1_TextblockDialogHeadingParity(t *testing.T) {
	typstPath, typPath := compileBookTypGateFixture(t, `
#set text(size: 10pt)
#_sizeFactor.update(1.5)
// AC-1 probe (textblock): heading must be counter-normalised to ~12pt, not the
// un-countered 18pt (1.2em default × 15pt _foreignSize ambient).
#textblock(role: "source", dir: rtl, script: "arab", [
== Arab H2 #context [#metadata(text.size < 15pt) <tb-arab-h2-narrow>]
Body #context [#metadata(text.size > 12pt) <tb-arab-body-large>]
])
// AC-1 probe (dialog): _blockheading is NOT inside _foreignSize so its H2 is
// always 1.2em × 10pt = 12pt regardless of counter. Parity anchor.
#dialog(dir: rtl, script: "arab", role: "source",
  (kind: "header", level: 2, text: [Arab Hdr #context [#metadata(text.size < 15pt) <dlg-arab-h2-narrow>]]),
  (header: "—", content: [Content.]),
)
// AC-3 probe: latn — _foreignSizeFactor("latn")=1.0, counter is 1em/1.0 = no-op.
// Heading stays at ~12pt (>9pt). Bug (divides by _sizeFactor=1.5): ~8pt (<9pt).
#textblock(role: "source", dir: ltr, script: "latn", [
== Latn H2 #context [#metadata(text.size > 9pt) <tb-latn-h2-ok>]
Body text.
])
`)
	// AC-1: textblock arab H2 counter-normalised — NOT at the enlarged ambient
	if !typstQueryBool(t, typstPath, typPath, "<tb-arab-h2-narrow>") {
		t.Errorf("FR-1 AC-1: textblock source arab H2 at enlarged size (want <15pt; " +
			"missing counter leaves Typst default 1.2em applied on 15pt ambient = 18pt; textblock H2 ≠ dialog H2)")
	}
	// AC-1: dialog arab H2 always at base size (parity anchor, unaffected by bug)
	if !typstQueryBool(t, typstPath, typPath, "<dlg-arab-h2-narrow>") {
		t.Errorf("FR-1 AC-1: dialog arab H2 header at enlarged size (want <15pt; parity anchor should always be 12pt)")
	}
	// AC-2: body text inside textblock IS enlarged by _foreignSize (counter is heading-only)
	if !typstQueryBool(t, typstPath, typPath, "<tb-arab-body-large>") {
		t.Errorf("FR-1 AC-2: body text inside source arab textblock not enlarged " +
			"(want >12pt; counter must be show-heading scoped — body must stay at _foreignSize = 15pt)")
	}
	// AC-3: latn heading not shrunken — counter divides by 1.0, not by 1.5
	if !typstQueryBool(t, typstPath, typPath, "<tb-latn-h2-ok>") {
		t.Errorf("FR-1 AC-3: latn H2 inside source textblock shrunken " +
			"(want >9pt; counter must use _foreignSizeFactor(script)=1.0 for latn, not raw _sizeFactor=1.5 giving ~8pt)")
	}
}

// --- FR-5 AC-1 centering gate tests ---

// TestBookTyp_FR5_BlockHeaderCentered is the FR-5 AC-1 compile gate:
// (kind:"header") items inside dialog and vocabulary must produce
// center-aligned headings. _blockheading() implements this as
// align(center, heading(...)) — the centering must survive through the
// heading render pipeline into the final layout.
//
// Centering is verified via a #metadata(here().position().x > 100pt) probe
// placed at the VERY START of the heading body (before any visible text).
// In Typst's default page (A4, 2.5cm margins = ≈70.87pt each side):
//   left-aligned start x ≈ 70.87pt → probe returns false (<100pt)
//   center-aligned start x ≈ (page_width − margin×2 − heading_width)/2 + margin
//                          ≈ 270pt for a short heading → probe returns true (>>100pt)
// The 100pt threshold cleanly separates these without requiring exact page metrics.
func TestBookTyp_FR5_BlockHeaderCentered(t *testing.T) {
	typstPath, typPath := compileStructuredBlockFixture(t, `
#dialog(dir: ltr, script: "latn", role: "source",
  (kind: "header", level: 2, text: [#context [#metadata(here().position().x > 100pt) <dlg-hdr-centered>]Dlg Hdr]),
  (header: "—", content: [Content.]),
)
#vocabulary(script: "latn",
  (kind: "header", level: 2, text: [#context [#metadata(here().position().x > 100pt) <voc-hdr-centered>]Voc Hdr]),
  (phrase: "hello", grammar: "", transcription: "", translation: "world"),
)
`)
	if !typstQueryBool(t, typstPath, typPath, "<dlg-hdr-centered>") {
		t.Errorf("FR-5 AC-1: dialog block header not centered " +
			"(want here().position().x > 100pt; _blockheading must wrap in align(center,...))")
	}
	if !typstQueryBool(t, typstPath, typPath, "<voc-hdr-centered>") {
		t.Errorf("FR-5 AC-1: vocabulary block header not centered " +
			"(want here().position().x > 100pt; _blockheading must wrap in align(center,...))")
	}
}
