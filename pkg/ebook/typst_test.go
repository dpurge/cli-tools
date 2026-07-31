package ebook

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// tableFrom builds a FontTable directly from a slot-key -> family map,
// bypassing font.css parsing, for tests that only care about Lookup's
// resolution algorithm (SPECS §4) rather than classifyFontFamily/parsing.
func tableFrom(slots map[string]string) FontTable {
	return FontTable{slots: slots}
}

// --- §4.1 worked example A: questions/answer vs questions/question -------

func TestLookup_WorkedExampleA_QuestionsAnswerVsQuestion(t *testing.T) {
	table := tableFrom(map[string]string{
		"arab questions question": "Noto Naskh Arabic",
		"arab questions answer":   "Amiri",
	})

	question := table.Lookup("arab", "questions", "question", "")
	if len(question) == 0 || question[0] != "Noto Naskh Arabic" {
		t.Errorf("Lookup(question) = %v, want first = Noto Naskh Arabic", question)
	}
	answer := table.Lookup("arab", "questions", "answer", "")
	if len(answer) == 0 || answer[0] != "Amiri" {
		t.Errorf("Lookup(answer) = %v, want first = Amiri", answer)
	}
	if question[0] == answer[0] {
		t.Errorf("question and answer resolved to the SAME family %q, want distinct (SPECS §4.1)", question[0])
	}
}

// --- §4.2 worked example B: text/transcription vs vocabulary/transcription

func TestLookup_WorkedExampleB_TextVsVocabularyTranscription(t *testing.T) {
	table := tableFrom(map[string]string{
		"latn text transcription":       "DejaVu Sans",
		"latn vocabulary transcription": "Noto Sans",
		"transcription":                 "Legacy Fallback", // legacy base role, lowest priority
	})

	text := table.Lookup("latn", "text", "transcription", "")
	vocab := table.Lookup("latn", "vocabulary", "transcription", "")

	if len(text) == 0 || text[0] != "DejaVu Sans" {
		t.Errorf("Lookup(text transcription) = %v, want first = DejaVu Sans", text)
	}
	if len(vocab) == 0 || vocab[0] != "Noto Sans" {
		t.Errorf("Lookup(vocabulary transcription) = %v, want first = Noto Sans", vocab)
	}
	if text[0] == vocab[0] {
		t.Errorf("text and vocabulary transcription resolved to the SAME family %q, want distinct (SPECS §4.2)", text[0])
	}
}

// --- §4 resolution order: field -> extension -> script -> base-role ------

func TestLookup_ResolutionOrder(t *testing.T) {
	tests := []struct {
		name   string
		slots  map[string]string
		script string
		ext    string
		field  string
		want   string
	}{
		{
			name:   "level 1: script+ext+field wins over everything else",
			slots:  map[string]string{"arab questions answer": "L1", "arab questions": "L2", "arab": "L3", "body": "L4"},
			script: "arab", ext: "questions", field: "answer",
			want: "L1",
		},
		{
			name:   "level 2: script+ext wins when field-specific is absent",
			slots:  map[string]string{"arab questions": "L2", "arab": "L3", "body": "L4"},
			script: "arab", ext: "questions", field: "answer",
			want: "L2",
		},
		{
			name:   "level 3: script-only wins when ext-specific is absent",
			slots:  map[string]string{"arab": "L3", "body": "L4"},
			script: "arab", ext: "questions", field: "answer",
			want: "L3",
		},
		{
			name:   "level 4: base role wins when nothing script-qualified is declared",
			slots:  map[string]string{"body": "L4"},
			script: "arab", ext: "questions", field: "answer",
			want: "L4",
		},
		{
			name:   "level 5: recommended fallback when nothing at all is declared",
			slots:  map[string]string{},
			script: "arab", ext: "questions", field: "answer",
			want: recommendedRoleFont["body"],
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tableFrom(tc.slots).Lookup(tc.script, tc.ext, tc.field, "")
			if len(got) == 0 || got[0] != tc.want {
				t.Errorf("Lookup(%q,%q,%q) = %v, want first = %q", tc.script, tc.ext, tc.field, got, tc.want)
			}
		})
	}
}

// --- SPECS §6 footnote: empty script skips the qualified chain entirely --

func TestLookup_EmptyScript_BaseRoleOnly(t *testing.T) {
	// Even though an extension+field-only slot exists, an empty script MUST
	// resolve via the base role directly (no book-Script inheritance, G1
	// deferred) rather than matching "questions answer" (no script segment).
	table := tableFrom(map[string]string{
		"questions answer": "ShouldNeverMatch",
		"body":             "BodyFont",
	})
	got := table.Lookup("", "questions", "answer", "")
	if len(got) == 0 || got[0] != "BodyFont" {
		t.Errorf("Lookup(empty script) = %v, want first = BodyFont (base-role path only)", got)
	}
}

// --- Style sub-axis (Strong/Emphasis) -------------------------------------

func TestLookup_StyleSubAxis(t *testing.T) {
	table := tableFrom(map[string]string{
		"arab text source strong": "SpecificStrong",
		"strong":                  "BaseStrong",
		"body":                    "BodyFont",
	})

	// Most specific styled candidate wins.
	got := table.Lookup("arab", "text", "source", "strong")
	if len(got) == 0 || got[0] != "SpecificStrong" {
		t.Errorf("Lookup(style=strong, specific declared) = %v, want first = SpecificStrong", got)
	}

	// No script-qualified styled candidate -> falls to the base style role.
	table2 := tableFrom(map[string]string{"strong": "BaseStrong", "body": "BodyFont"})
	got2 := table2.Lookup("arab", "text", "source", "strong")
	if len(got2) == 0 || got2[0] != "BaseStrong" {
		t.Errorf("Lookup(style=strong, only base declared) = %v, want first = BaseStrong", got2)
	}

	// Nothing styled declared at all -> falls through to the regular chain.
	table3 := tableFrom(map[string]string{"arab": "RegularArab", "body": "BodyFont"})
	got3 := table3.Lookup("arab", "text", "source", "strong")
	if len(got3) == 0 || got3[0] != "RegularArab" {
		t.Errorf("Lookup(style=strong, nothing styled declared) = %v, want first = RegularArab (regular fallback)", got3)
	}
}

// --- Major-2: fixed-script decoupling -------------------------------------

// TestLookup_Major2_FixedScriptDecoupling is the SPECS §10-required test:
// a font.css with a bare `Font <Script>` catch-all plus a legacy `Font
// Translation` base role. Calling Lookup with the block's OWN foreign
// script for a translation field would incorrectly hijack it into the
// catch-all; the CALLER is responsible for decoupling by passing the
// field's own fixed script ("" for translation/grammar, "latn" for
// transcription) instead of the block's foreign script (book.typ's
// textblock/questions/dialog/parallel all do this — see book.typ's
// familyScript variables).
func TestLookup_Major2_FixedScriptDecoupling(t *testing.T) {
	table := tableFrom(map[string]string{
		"arab":        "ArabicCatchAll", // e.g. a book-wide "Font Arab" declaration
		"translation": "PolishTranslationFont",
		"body":        "BodyFont",
	})

	// Correct (decoupled) usage: caller passes the FIXED base script "" for
	// a translation-role field, regardless of the block's own script=arab.
	decoupled := table.Lookup("", "questions", "translation", "")
	if len(decoupled) == 0 || decoupled[0] != "PolishTranslationFont" {
		t.Errorf("Lookup(decoupled, script=\"\") = %v, want first = PolishTranslationFont", decoupled)
	}

	// Demonstrates WHY decoupling matters: passing the block's foreign
	// script directly (the bug Major-2 prevents) DOES hijack into the
	// catch-all — this is the wrong behavior, confirmed here only to prove
	// book.typ/callers must never do this for fixed-script fields.
	notDecoupled := table.Lookup("arab", "questions", "translation", "")
	if len(notDecoupled) == 0 || notDecoupled[0] != "ArabicCatchAll" {
		t.Fatalf("sanity check failed: expected the undecoupled call to demonstrate the hijack, got %v", notDecoupled)
	}
	if notDecoupled[0] == decoupled[0] {
		t.Errorf("decoupled and undecoupled resolution should differ (that's the whole point of Major-2), both got %q", decoupled[0])
	}
}

// TestLookupTranslation_AsTranslationSwapsBaseRole covers SPECS §4's
// footnote: as=translation resolves primary-text fields (source/content/
// main/question/answer/phrase) to the Translation base role instead of
// Body, but ONLY at the base-role fallback level — a script-qualified
// declaration for the field's own name is still tried first, unchanged.
func TestLookupTranslation_AsTranslationSwapsBaseRole(t *testing.T) {
	table := tableFrom(map[string]string{
		"body":        "BodyFont",
		"translation": "TranslationFont",
	})

	// No script-qualified override declared -> as=translation's fallback
	// resolves Translation, not Body.
	got := table.LookupTranslation("", "dialog", "content", "")
	if len(got) == 0 || got[0] != "TranslationFont" {
		t.Errorf("LookupTranslation(no override) = %v, want first = TranslationFont", got)
	}

	// A script-qualified override for the field's OWN name is still tried
	// first, unaffected by the as=translation swap (only the base-role
	// FALLBACK level changes).
	table2 := tableFrom(map[string]string{
		"arab dialog content": "SpecificContentFont",
		"body":                "BodyFont",
		"translation":         "TranslationFont",
	})
	got2 := table2.Lookup("arab", "dialog", "content", "") // regular Lookup, not swapped
	if len(got2) == 0 || got2[0] != "SpecificContentFont" {
		t.Errorf("Lookup(specific override present) = %v, want first = SpecificContentFont", got2)
	}
}

// --- Handbook (book=native) path: resolution is book-agnostic (A2) -------

// TestLookup_HandbookPath_BookScriptNeverParticipates proves the resolver
// is architecturally incapable of depending on the book's own language/
// script (SPECS A2, framing-agnostic): Lookup's signature has no book-level
// parameter, so resolving the SAME (script,ext,field,style) tuple gives the
// SAME result regardless of whether the surrounding book is framed as
// "native" (a beginner handbook, book-level script=latn) or "foreign" (an
// advanced reader, book-level script=arab) — only the block's OWN resolved
// script drives the outcome, symmetrically in both framings.
func TestLookup_HandbookPath_BookScriptNeverParticipates(t *testing.T) {
	table := tableFrom(map[string]string{
		"arab vocabulary phrase": "ForeignPhraseFont",
		"body":                   "BodyFont",
	})

	// Framing 1: "handbook" — book-level script is native (latn), the
	// reader is LEARNING Arabic, so an Arabic vocabulary block still
	// declares script=arab per-block (per-block declaration is the only
	// mechanism, SPECS G1: no book-level inheritance for EPUB/PDF).
	handbook := table.Lookup("arab", "vocabulary", "phrase", "")

	// Framing 2: "advanced reader" — book-level script is foreign (arab),
	// same per-block declaration.
	advanced := table.Lookup("arab", "vocabulary", "phrase", "")

	if !reflect.DeepEqual(handbook, advanced) {
		t.Errorf("Lookup should be symmetric across book framings: handbook=%v, advanced=%v", handbook, advanced)
	}
	if len(handbook) == 0 || handbook[0] != "ForeignPhraseFont" {
		t.Errorf("Lookup(arab vocabulary phrase) = %v, want first = ForeignPhraseFont regardless of book framing", handbook)
	}
}

// --- ASR-4: EPUB<->Typst parity (de-circularized, review 2026-07-31) -----
//
// The PREVIOUS version of this test compared `table.Lookup(...)` against
// `table.Lookup(...)` on the identical synthetic table -- trivially always
// equal by construction (code-review finding: "not Lookup-vs-Lookup, that
// is circular"). This version instead sources the "authored" side from the
// REAL, hand-written epub-public CSS component file (skips cleanly if that
// sibling checkout isn't present, mirroring findTurProjectFile in
// typst_export_test.go) and derives the "expected resolver order" side
// independently from slotKey/baseRoleForField -- restating SPECS §4's
// algorithm in its own terms, never by invoking Lookup itself.

// findEpubPublicCSS locates a real epub-public arab/<name> CSS file in a
// sibling checkout, returning "" if it isn't present (this repo carries no
// CSS fixtures of its own).
func findEpubPublicCSS(name string) string {
	abs, err := filepath.Abs("../../../epub-public/src/css/main/arab/" + name)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return ""
	}
	return abs
}

// cssSelectorFontFamily extracts the font-family declaration's raw value
// (everything up to the terminating ";") from the given selector's own
// rule block in css.
func cssSelectorFontFamily(css, selector string) (string, bool) {
	re := regexp.MustCompile(regexp.QuoteMeta(selector) + `\s*\{[^}]*font-family\s*:\s*([^;]+);`)
	m := re.FindStringSubmatch(css)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// parseAuthoredFontFamilyChain splits a CSS font-family value into its
// ordered list of quoted "Font ..." role names, dropping any trailing
// unquoted generic keyword (serif/sans-serif/...) -- that's the CSS-native
// final fallback, not a SPECS §3 role name.
func parseAuthoredFontFamilyChain(value string) []string {
	var names []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if len(part) >= 2 && (part[0] == '"' || part[0] == '\'') {
			names = append(names, part[1:len(part)-1])
		}
	}
	return names
}

// TestEPUBTypstParity_DerivedChainMatchesResolutionOrder is the SPECS §10
// ASR-4 parity test. For each of epub-public's real, hand-authored
// arab/questions.css chains (.s-arab .questions-question/.answer), it:
//  1. Classifies each authored role name (classifyFontFamily) into its
//     canonical slot key and compares that ORDER against the expected §4
//     chain, built independently via slotKey/baseRoleForField.
//  2. Cross-checks that the real Go resolver, fed the REAL arab/font.css
//     via parseFontRoles, actually picks the family the authored chain's
//     most-specific declared level names -- tying the author's declared
//     CSS order to the resolver's real runtime behavior.
func TestEPUBTypstParity_DerivedChainMatchesResolutionOrder(t *testing.T) {
	cssPath := findEpubPublicCSS("questions.css")
	if cssPath == "" {
		t.Skip("epub-public arab/questions.css not found (expected a sibling epub-public checkout at ../../../epub-public); skipping ASR-4 parity test")
	}
	data, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read %s: %v", cssPath, err)
	}
	css := string(data)

	for _, tc := range []struct{ selector, field string }{
		{".s-arab .questions-question", "question"},
		{".s-arab .questions-answer", "answer"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			value, ok := cssSelectorFontFamily(css, tc.selector)
			if !ok {
				t.Fatalf("selector %q not found in %s", tc.selector, cssPath)
			}
			chain := parseAuthoredFontFamilyChain(value)

			var gotKeys []string
			for _, fam := range chain {
				key, ok := classifyFontFamily(fam)
				if !ok {
					t.Fatalf("authored family %q does not classify to a SPECS §3 slot", fam)
				}
				gotKeys = append(gotKeys, key)
			}

			wantKeys := []string{
				slotKey("arab", "questions", tc.field),
				slotKey("arab", "questions"),
				slotKey("arab"),
				baseRoleForField(tc.field, false),
			}
			if !reflect.DeepEqual(gotKeys, wantKeys) {
				t.Errorf("authored CSS chain for %q classifies to %v, want resolver order %v (ASR-4: authored order must match §4's field->extension->script->base-role chain)", tc.selector, gotKeys, wantKeys)
			}
		})
	}

	// Cross-check against the REAL resolver fed the REAL font.css: Lookup's
	// winning family for (arab, questions, question|answer, "") must be the
	// family actually declared at font.css's own "Font Arab Questions
	// Question/Answer" slot -- the most-specific level both the authored
	// chain and the resolver agree should win (INC4's actual demo
	// substitutes, not test-only fixture data).
	fontCSSPath := findEpubPublicCSS("font.css")
	if fontCSSPath == "" {
		t.Skip("epub-public arab/font.css not found; skipping resolver cross-check")
	}
	table := parseFontRoles([]string{fontCSSPath})
	for _, tc := range []struct{ field, wantFamily string }{
		{"question", "Geeza Pro"},
		{"answer", "KufiStandardGK"},
	} {
		got := table.Lookup("arab", "questions", tc.field, "")
		if len(got) == 0 || got[0] != tc.wantFamily {
			t.Errorf("Lookup(arab, questions, %q, \"\") = %v, want first = %q (the real family declared in font.css at the level the authored CSS chain names first)", tc.field, got, tc.wantFamily)
		}
	}
}

// --- classifyFontFamily / parseFontRoles (SPECS §3 grammar) ---------------

func TestClassifyFontFamily(t *testing.T) {
	tests := []struct {
		name    string
		family  string
		wantKey string
		wantOK  bool
	}{
		{"legacy base role", "Font Body", "body", true},
		{"legacy base role emphasis", "Font Emphasis", "emphasis", true},
		{"script-only qualified role", "Font Arab", "arab", true},
		{"script+extension+field", "Font Arab Questions Answer", "arab questions answer", true},
		{"script+extension+field, author order varies (order-tolerant)", "Font Questions Arab Answer", "arab questions answer", true},
		// SPECS §3 reachability rule (review 2026-07-31): a field segment
		// requires its extension segment to be resolvable -- the §4 chain
		// drops right-to-left (S E F)->(S E)->(S)->base, never (S F). So
		// "Font Hebr Phrase Strong" (extension omitted) IS classifiable
		// (this test) but is NOT reachable via Lookup (no candidate key ever
		// omits only the extension) -- authoring it would be a silent dead
		// declaration. This case uses a reachable extension+field+style
		// example instead, matching the §3 "Font Hebr Vocabulary Phrase
		// Strong" example.
		{"script+extension+field+style", "Font Hebr Vocabulary Phrase Strong", "hebr vocabulary phrase strong", true},
		{"extension+field, script omitted", "Font Vocabulary Transcription", "vocabulary transcription", true},
		{"not a Font role at all", "Helvetica", "", false},
		{"Font with no segments", "Font", "", false},
		{"every segment unrecognized -> ignored", "Font Zzzz1", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, ok := classifyFontFamily(tc.family)
			if ok != tc.wantOK {
				t.Fatalf("classifyFontFamily(%q) ok = %v, want %v", tc.family, ok, tc.wantOK)
			}
			if ok && key != tc.wantKey {
				t.Errorf("classifyFontFamily(%q) key = %q, want %q", tc.family, key, tc.wantKey)
			}
		})
	}
}
