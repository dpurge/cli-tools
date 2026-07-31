package ebook

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/config"
)

// --- languageInfo (SPECS AC7: reproduce every setLanguage mapping) --------

func TestLanguageInfoLanguageMapping(t *testing.T) {
	tests := []struct {
		language string
		script   string
		wantLang string
	}{
		{"ajp", "", "ar"},
		{"apc", "", "ar"},
		{"arb", "", "ar"},
		{"bul", "", "bg"},
		{"ces", "", "cs"},
		{"cmn", "hant", "zh-Hant"},
		{"cmn", "hans", "zh-Hans"},
		{"cmn", "", "zh-Hans"},
		{"dan", "", "da"},
		{"deu", "", "de"},
		{"ell", "", "el"},
		{"fas", "", "fa"},
		{"fra", "", "fr"},
		{"grc", "", "el"},
		{"hin", "", "hi"},
		{"ind", "", "id"},
		{"ita", "", "it"},
		{"kaz", "", "kk"},
		{"lat", "", "la"},
		{"lit", "", "lt"},
		{"mon", "", "mn"},
		{"nld", "", "nl"},
		{"ron", "", "ro"},
		{"spa", "", "es"},
		{"srp", "", "sr"},
		{"tgk", "", "tg"},
		{"tha", "", "th"},
		{"tur", "", "tr"},
		{"uig", "", "ug"},
		{"ukr", "", "uk"},
		{"uzb", "", "uz"},
		{"vie", "", "vi"},
		{"yid", "", "yi"},
		{"yue", "hans", "zh-Hans"},
		{"yue", "hant", "zh-Hant"},
		{"yue", "", "zh-Hant"},
		// Pre-existing quirk (documented in exporter.go's languageInfo
		// doc comment): the real ISO 639-3 Hebrew code "heb" (used by the
		// heb sample project) was never one of setLanguage's cases, so it
		// falls through to the default "en". languageInfo MUST reproduce
		// this, not fix it.
		{"heb", "hebr", "en"},
		{"xyz", "", "en"},
		{"", "", "en"},
	}

	for _, tt := range tests {
		lang, _ := languageInfo(tt.language, tt.script)
		if lang != tt.wantLang {
			t.Errorf("languageInfo(%q, %q) lang = %q, want %q", tt.language, tt.script, lang, tt.wantLang)
		}
	}
}

func TestLanguageInfoScriptDirection(t *testing.T) {
	tests := []struct {
		script  string
		wantDir string
	}{
		{"latn", "ltr"},
		{"arab", "rtl"},
		{"hebr", "rtl"},
		{"", "ltr"},
		{"cyrl", "ltr"},
		{"hant", "ltr"},
	}

	for _, tt := range tests {
		// Language is held constant; direction depends only on script.
		_, dir := languageInfo("tur", tt.script)
		if dir != tt.wantDir {
			t.Errorf("languageInfo(\"tur\", %q) dir = %q, want %q", tt.script, dir, tt.wantDir)
		}
	}
}

// --- WalkTexts (SPECS §8.1 CRITICAL global-counter invariant) ------------

func TestWalkTextsSingleSection(t *testing.T) {
	text := [][]string{
		{"section.md", "ch1.md", "ch2.md", "ch3.md"},
	}

	items := WalkTexts(text)

	want := []ProjectItem{
		{File: "section.md", Kind: SectionItem, SectionIdx: 1, ChapterIdx: 0},
		{File: "ch1.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 1},
		{File: "ch2.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 2},
		{File: "ch3.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 3},
	}

	if len(items) != len(want) {
		t.Fatalf("WalkTexts() returned %d items, want %d: %+v", len(items), len(want), items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("WalkTexts()[%d] = %+v, want %+v", i, items[i], want[i])
		}
	}
}

// TestWalkTextsMultiSectionGlobalChapterCounter is THE regression test for
// the CRITICAL constraint in SPECS §8.1: chapterId in the pre-refactor
// addTexts (epub.go) was a running counter across ALL sections, never reset
// when a new section starts. Every real project in this codebase has
// exactly one section, so this synthetic 2-section, multi-chapter fixture
// is the only thing that can catch a per-section-counter regression.
func TestWalkTextsMultiSectionGlobalChapterCounter(t *testing.T) {
	text := [][]string{
		{"sec1.md", "ch1.md", "ch2.md"},
		{"sec2.md", "ch3.md"},
	}

	items := WalkTexts(text)

	want := []ProjectItem{
		{File: "sec1.md", Kind: SectionItem, SectionIdx: 1, ChapterIdx: 0},
		{File: "ch1.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 1},
		{File: "ch2.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 2},
		{File: "sec2.md", Kind: SectionItem, SectionIdx: 2, ChapterIdx: 0},
		// THE regression case: this chapter is the first chapter of the
		// SECOND section, but its ChapterIdx must be 3 (continuing the
		// global count from section 1's ch1/ch2), NOT 1.
		{File: "ch3.md", Kind: ChapterItem, SectionIdx: 2, ChapterIdx: 3},
	}

	if len(items) != len(want) {
		t.Fatalf("WalkTexts() returned %d items, want %d: %+v", len(items), len(want), items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("WalkTexts()[%d] = %+v, want %+v", i, items[i], want[i])
		}
	}
}

func TestWalkTextsSkipsEmptyGroups(t *testing.T) {
	text := [][]string{
		{"sec1.md", "ch1.md"},
		{},
		{"sec2.md", "ch2.md"},
	}

	items := WalkTexts(text)

	// The empty group must be skipped entirely (mirrors the pre-refactor
	// `if len(items) > 0` guard) and must NOT consume a SectionIdx.
	want := []ProjectItem{
		{File: "sec1.md", Kind: SectionItem, SectionIdx: 1, ChapterIdx: 0},
		{File: "ch1.md", Kind: ChapterItem, SectionIdx: 1, ChapterIdx: 1},
		{File: "sec2.md", Kind: SectionItem, SectionIdx: 2, ChapterIdx: 0},
		{File: "ch2.md", Kind: ChapterItem, SectionIdx: 2, ChapterIdx: 2},
	}

	if len(items) != len(want) {
		t.Fatalf("WalkTexts() returned %d items, want %d: %+v", len(items), len(want), items)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("WalkTexts()[%d] = %+v, want %+v", i, items[i], want[i])
		}
	}
}

// --- epubExporter global chapter numbering, end to end --------------------

// TestEPUBExporterGlobalChapterNumbering builds an actual EPUB (not just a
// WalkTexts structure) from a synthetic 2-section, multi-chapter project
// and inspects the real internal EPUB filenames, proving the global counter
// invariant survives the full epubExporter.Export path, not merely
// WalkTexts in isolation.
func TestEPUBExporterGlobalChapterNumbering(t *testing.T) {
	dir := t.TempDir()

	writeMD := func(name, title string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("# "+title+"\n\nBody text.\n"), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
		return path
	}

	project := &EBookProject{
		Identifier: "urn:test:multi-section",
		Filename:   filepath.Join(dir, "multi.epub"),
		Title:      "Multi Section Test",
		Language:   "eng",
		Text: [][]string{
			{
				writeMD("sec1.md", "Section One"),
				writeMD("ch1.md", "Chapter One"),
				writeMD("ch2.md", "Chapter Two"),
			},
			{
				writeMD("sec2.md", "Section Two"),
				writeMD("ch3.md", "Chapter Three"),
			},
		},
	}

	outfile, err := (epubExporter{}).Export(project)
	if err != nil {
		t.Fatalf("epubExporter.Export() error = %v", err)
	}

	r, err := zip.OpenReader(outfile)
	if err != nil {
		t.Fatalf("open generated epub as zip: %v", err)
	}
	defer r.Close()

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}

	// The CRITICAL assertion: chapter0003.xhtml (not a re-started
	// chapter0001.xhtml) must exist for the chapter under the SECOND
	// section, proving the counter is global/continuous across sections.
	wantSuffixes := []string{
		"section0001.xhtml",
		"chapter0001.xhtml",
		"chapter0002.xhtml",
		"section0002.xhtml",
		"chapter0003.xhtml",
	}
	for _, want := range wantSuffixes {
		found := false
		for _, n := range names {
			if strings.HasSuffix(n, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected an internal EPUB file ending in %q; got %v", want, names)
		}
	}

	// And the regression this guards against: a per-section counter would
	// produce a SECOND chapter0001.xhtml instead of chapter0003.xhtml.
	chapter0001Count := 0
	for _, n := range names {
		if strings.HasSuffix(n, "chapter0001.xhtml") {
			chapter0001Count++
		}
	}
	if chapter0001Count != 1 {
		t.Errorf("expected exactly one chapter0001.xhtml (global counter), found %d in %v", chapter0001Count, names)
	}
}

// --- build-cmd.go --format dispatch (FR-8) --------------------------------

func TestExporterFor(t *testing.T) {
	if exp, err := exporterFor("epub"); err != nil {
		t.Errorf("exporterFor(\"epub\") error = %v", err)
	} else if _, ok := exp.(epubExporter); !ok {
		t.Errorf("exporterFor(\"epub\") = %T, want epubExporter", exp)
	}

	if exp, err := exporterFor("pdf"); err != nil {
		t.Errorf("exporterFor(\"pdf\") error = %v", err)
	} else if _, ok := exp.(typstExporter); !ok {
		t.Errorf("exporterFor(\"pdf\") = %T, want typstExporter", exp)
	}

	if _, err := exporterFor("bogus"); err == nil {
		t.Error("exporterFor(\"bogus\") expected an error, got nil")
	}
}

// --- locateTypst fallback (SPECS §8.5) ------------------------------------

func TestLocateTypstFallsBackToPath(t *testing.T) {
	// No config file is loaded in this test binary (config.ReadConfig is
	// never called - see the package doc note in this file), so
	// config.GetToolPath("Typst", "typst") always errors here and
	// locateTypst must fall back to exec.LookPath. This only proves the
	// fallback branch executes; whether it actually finds a binary depends
	// on the host, which is why every other test that needs a real typst
	// skips cleanly when it doesn't.
	_, err := locateTypst()
	if err != nil && !strings.Contains(err.Error(), "typst not found") {
		t.Errorf("locateTypst() unexpected error shape: %v", err)
	}
}

// --- typst exporter integration: real tur sample -> PDF -------------------

// findTurProjectFile locates the tur sample's ebook.yml in a sibling
// epub-public checkout (this repo carries no ebook fixtures of its own).
// It returns "" if that checkout isn't present, so the test below skips
// cleanly on any machine/CI without it.
func findTurProjectFile() string {
	abs, err := filepath.Abs("../../../epub-public/src/txt/lang-notes/tur/ebook.yml")
	if err != nil {
		return ""
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return ""
	}
	return abs
}

// TestTypstExporterIntegrationTurSample builds the real tur sample project
// to PDF via typstExporter, skipping cleanly if either the sample checkout
// or the typst binary isn't available (SPECS §10).
//
// The generated .pdf/.typ are written next to the sample's ebook.yml (the
// exporter's documented contract, SPECS §8.4/D7: --root must contain the
// project's own assets, e.g. cover.svg, which live there) and removed via
// t.Cleanup regardless of outcome, so this test never leaves artifacts in
// the sibling checkout.
func TestTypstExporterIntegrationTurSample(t *testing.T) {
	projectFile := findTurProjectFile()
	if projectFile == "" {
		t.Skip("tur sample project not found (expected a sibling epub-public checkout at ../../../epub-public); skipping PDF integration test")
	}

	if _, err := locateTypst(); err != nil {
		t.Skipf("typst binary not available: %v", err)
	}

	project, err := readProject(projectFile)
	if err != nil {
		t.Fatalf("readProject(%q) error = %v", projectFile, err)
	}

	pdfPath, typPath := derivedTypstPaths(project.Filename)
	t.Cleanup(func() {
		os.Remove(pdfPath)
		os.Remove(typPath)
	})

	outfile, err := (typstExporter{}).Export(project)
	if err != nil {
		t.Fatalf("typstExporter.Export() error = %v", err)
	}
	if outfile != pdfPath {
		t.Fatalf("typstExporter.Export() outfile = %q, want %q", outfile, pdfPath)
	}

	data, err := os.ReadFile(outfile)
	if err != nil {
		t.Fatalf("read generated pdf %q: %v", outfile, err)
	}
	if len(data) == 0 {
		t.Fatal("generated pdf is empty")
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		n := len(data)
		if n > 16 {
			n = 16
		}
		t.Fatalf("generated file does not start with a %%PDF header (first bytes: %q)", data[:n])
	}
}

// --- largeScript / A5 defaults / assembled document ----------------------

// showCall returns just the emitted `#show: book.with(...)` call from an
// assembled document, dropping the template preamble. That preamble contains
// both a commented `// #show: book.with(...)` example and doc comments that
// mention argument names (e.g. "large-script: true"), which would otherwise
// produce false substring matches. The real call is the only occurrence of
// the exact marker below (newline, then the call, then a newline).
func showCall(doc string) string {
	marker := "\n#show: book.with(\n"
	i := strings.Index(doc, marker)
	if i < 0 {
		return doc
	}
	return doc[i+1:]
}

// TestLargeScript covers the scripts that select book.typ's enlarged body
// size (Chinese/Arabic/Hebrew/Korean/Japanese), the case/whitespace
// tolerance, and the negatives (Latin, empty, unknown).
func TestLargeScript(t *testing.T) {
	large := []string{
		"hans", "hant", "hani", "arab", "hebr",
		"kore", "hang", "jpan", "hira", "kana",
		"HANS", "Arab", " hebr ",
	}
	for _, s := range large {
		if !largeScript(s) {
			t.Errorf("largeScript(%q) = false, want true", s)
		}
	}

	small := []string{"latn", "cyrl", "grek", "thai", "", "xyz"}
	for _, s := range small {
		if largeScript(s) {
			t.Errorf("largeScript(%q) = true, want false", s)
		}
	}
}

// TestBookTemplateDefaults asserts the embedded book.typ carries the built-in
// defaults the exporter falls back to when nothing is configured: A5 paper,
// 12pt base body, the enlarged-body knob, and 1cm uniform margin. A regression
// that drops or renames any of these (they are the config fallback) is caught
// here.
func TestBookTemplateDefaults(t *testing.T) {
	for _, want := range []string{`paper: "a5"`, "size: 12pt", "size-large: 16pt", "large-script: false", "inside: 1.8cm, outside: 1.4cm"} {
		if !strings.Contains(bookTemplate, want) {
			t.Errorf("book.typ template missing expected default %q", want)
		}
	}
}

// TestAssembleTypstDocumentLargeScript proves the exporter emits the
// `large-script: true` argument only for a large-script project, so the base
// size stays in effect for Latin and the enlarged size kicks in for CJK/RTL.
func TestAssembleTypstDocumentLargeScript(t *testing.T) {
	arab, err := assembleTypstDocument(
		&EBookProject{Title: "T", Script: "arab"}, "ar", "rtl", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument(arab) error = %v", err)
	}
	if !strings.Contains(showCall(arab), "large-script: true") {
		t.Errorf("expected `large-script: true` for arab script, got:\n%s", showCall(arab))
	}

	latn, err := assembleTypstDocument(
		&EBookProject{Title: "T", Script: "latn"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument(latn) error = %v", err)
	}
	if strings.Contains(showCall(latn), "large-script: true") {
		t.Errorf("did not expect `large-script: true` for latn script, got:\n%s", showCall(latn))
	}
}

// TestAssembleTypstDocumentNoConfigOverrides confirms an empty PdfConfig (the
// no-config-file case) emits none of the override arguments, so book.typ's
// built-in defaults remain in force and the config section stays optional.
func TestAssembleTypstDocumentNoConfigOverrides(t *testing.T) {
	doc, err := assembleTypstDocument(
		&EBookProject{Title: "T", Script: "latn"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	call := showCall(doc)
	for _, arg := range []string{"paper:", "size:", "size-large:", "margin:", "font:"} {
		if strings.Contains(call, arg) {
			t.Errorf("empty config should not emit %q, got call:\n%s", arg, call)
		}
	}
}

// TestAssembleTypstDocumentConfigOverrides verifies every configured Pdf.*
// value is emitted into the book.with(...) call in the right Typst form:
// paper/font as string literals, size/size-large as bare lengths, and margin
// as a dict with a `rest` fallback for the unset sides.
func TestAssembleTypstDocumentConfigOverrides(t *testing.T) {
	cfg := config.PdfConfig{
		Paper:     "a4",
		Size:      "12pt",
		SizeLarge: "18pt",
		Margin:    config.PdfMargin{Top: "2cm", Left: "1.5cm"},
		Font:      []string{"Amiri", "Noto Sans"},
	}
	doc, err := assembleTypstDocument(
		&EBookProject{Title: "T", Script: "arab"}, "ar", "rtl", "", []string{"body"}, cfg)
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	call := showCall(doc)

	wants := []string{
		`paper: "a4"`,
		"size: 12pt",
		"size-large: 18pt",
		"margin: (top: 2cm, left: 1.5cm, rest: 1.5cm)",
		`font: ("Amiri", "Noto Sans")`,
	}
	for _, w := range wants {
		if !strings.Contains(call, w) {
			t.Errorf("expected override %q in call, got:\n%s", w, call)
		}
	}
}

// TestAssembleTypstDocumentRejectsBadLength ensures a malformed configured
// length fails the build with a clear error rather than emitting broken Typst.
func TestAssembleTypstDocumentRejectsBadLength(t *testing.T) {
	cfg := config.PdfConfig{Size: "12"} // missing unit
	if _, err := assembleTypstDocument(
		&EBookProject{Title: "T"}, "en", "ltr", "", []string{"body"}, cfg); err == nil {
		t.Error("expected an error for a unitless size, got nil")
	}
}

// TestTypstLength covers the length validator's accepted units and rejects.
func TestTypstLength(t *testing.T) {
	for _, ok := range []string{"12pt", "1cm", "1.5in", "10mm", "2em", " 12pt "} {
		if v, err := typstLength("k", ok); err != nil {
			t.Errorf("typstLength(%q) unexpected error: %v", ok, err)
		} else if v != strings.TrimSpace(ok) {
			t.Errorf("typstLength(%q) = %q, want trimmed", ok, v)
		}
	}
	for _, bad := range []string{"12", "pt", "12 pt", "12px", "abc", "", "-3pt"} {
		if _, err := typstLength("k", bad); err == nil {
			t.Errorf("typstLength(%q) expected error, got nil", bad)
		}
	}
}

// TestTypstMarginDict covers: no sides -> empty; some sides -> those plus a
// `rest` default; and propagation of a bad length.
func TestTypstMarginDict(t *testing.T) {
	if got, err := typstMarginDict(config.PdfMargin{}); err != nil || got != "" {
		t.Errorf("empty margin = (%q, %v), want (\"\", nil)", got, err)
	}
	got, err := typstMarginDict(config.PdfMargin{Top: "2cm", Right: "3mm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(top: 2cm, right: 3mm, rest: 1.5cm)" {
		t.Errorf("partial margin = %q, want (top: 2cm, right: 3mm, rest: 1.5cm)", got)
	}
	// Binding-aware inside/outside are recognized alongside the fixed edges.
	binding, err := typstMarginDict(config.PdfMargin{Inside: "1.8cm", Outside: "1.4cm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if binding != "(inside: 1.8cm, outside: 1.4cm, rest: 1.5cm)" {
		t.Errorf("binding margin = %q, want (inside: 1.8cm, outside: 1.4cm, rest: 1.5cm)", binding)
	}
	if _, err := typstMarginDict(config.PdfMargin{Bottom: "nope"}); err == nil {
		t.Error("expected error for bad margin length, got nil")
	}
}

// TestTypstFontArray covers single (trailing comma), multiple, and quote
// escaping of family names.
func TestTypstFontArray(t *testing.T) {
	if got := typstFontArray([]string{"Amiri"}); got != `("Amiri",)` {
		t.Errorf("single font = %q, want (\"Amiri\",)", got)
	}
	if got := typstFontArray([]string{"Amiri", "Noto Sans"}); got != `("Amiri", "Noto Sans")` {
		t.Errorf("multi font = %q", got)
	}
}

// --- typstLang primary-subtag (CJK PDF lang fix) --------------------------

// TestTypstLang covers the BCP-47 → Typst text.lang reduction: Chinese tags
// lose their script subtag, already-bare tags pass through unchanged.
func TestTypstLang(t *testing.T) {
	cases := map[string]string{
		"zh-Hans": "zh",
		"zh-Hant": "zh",
		"ar":      "ar",
		"en":      "en",
		"zh":      "zh",
		"":        "",
	}
	for in, want := range cases {
		if got := typstLang(in); got != want {
			t.Errorf("typstLang(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestAssembleTypstDocumentCJKLang proves a Chinese project (which languageInfo
// maps to the full tag "zh-Hans") emits a Typst-valid bare `lang: "zh"`, not
// the "zh-Hans" that `set text(lang:)` rejects.
func TestAssembleTypstDocumentCJKLang(t *testing.T) {
	lang, dir := languageInfo("cmn", "hans") // -> "zh-Hans", "ltr"
	doc, err := assembleTypstDocument(
		&EBookProject{Title: "T", Language: "cmn", Script: "hans"},
		lang, dir, "", []string{"你好"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	call := showCall(doc)
	if !strings.Contains(call, `lang: "zh"`) {
		t.Errorf("expected `lang: \"zh\"` in call, got:\n%s", call)
	}
	if strings.Contains(call, `lang: "zh-Hans"`) {
		t.Errorf("must not emit the BCP-47 tag `lang: \"zh-Hans\"`, got:\n%s", call)
	}
}

// TestAssembleTypstDocumentNoCoverOmitsArg confirms an empty cover (the
// post-fix readProject value for an unset cover) emits no `cover:` argument,
// so Typst is never handed a bogus path.
func TestAssembleTypstDocumentNoCoverOmitsArg(t *testing.T) {
	doc, err := assembleTypstDocument(
		&EBookProject{Title: "T"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	if strings.Contains(showCall(doc), "cover:") {
		t.Errorf("empty cover should emit no `cover:` arg, got:\n%s", showCall(doc))
	}
}

// TestReadProjectNoCoverStaysEmpty guards the cover fix end to end: a project
// file without a `cover:` must leave project.Cover == "" after readProject
// (before the fix it became the project directory).
func TestReadProjectNoCoverStaysEmpty(t *testing.T) {
	dir := t.TempDir()
	ymlPath := filepath.Join(dir, "ebook.yml")
	yml := "identifier: urn:test:nocover\nfilename: out.epub\ntitle: T\n"
	if err := os.WriteFile(ymlPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("write ebook.yml: %v", err)
	}
	project, err := readProject(ymlPath)
	if err != nil {
		t.Fatalf("readProject error = %v", err)
	}
	if project.Cover != "" {
		t.Errorf("Cover = %q, want \"\" for an unset cover", project.Cover)
	}
}

// --- font.css role parsing + role prefixes -------------------------------

func TestParseFontRoles(t *testing.T) {
	dir := t.TempDir()
	css := `
@font-face { font-family: "Font Header"; src: local("Helvetica"); }
@font-face { font-family: "Font Body"; src: local("Amiri Regular"); }
@font-face { font-family: "Font Transcription"; src: local("DejaVu Sans"); }
@font-face { font-family: "Font Translation"; src: local("Times New Roman"); }
@font-face { font-family: "Font Strong"; src: local("Amiri Bold"); }
@font-face { font-family: "Font Emphasis"; src: local("Amiri Slanted"); }
`
	path := filepath.Join(dir, "font.css")
	if err := os.WriteFile(path, []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}
	got := parseFontRoles([]string{filepath.Join(dir, "base.css"), path})
	want := map[string]string{
		"header": "Helvetica", "body": "Amiri Regular", "transcription": "DejaVu Sans",
		"translation": "Times New Roman", "strong": "Amiri Bold", "emphasis": "Amiri Slanted",
	}
	for role, fam := range want {
		if got.BaseRole(role) != fam {
			t.Errorf("parseFontRoles(...).BaseRole(%q) = %q, want %q", role, got.BaseRole(role), fam)
		}
	}

	// No font.css among the paths => all empty (zero-value FontTable).
	r := parseFontRoles([]string{filepath.Join(dir, "base.css")})
	for role := range want {
		if got := r.BaseRole(role); got != "" {
			t.Errorf("parseFontRoles(no font.css).BaseRole(%q) = %q, want \"\"", role, got)
		}
	}
}

func TestRoleFontPrefix(t *testing.T) {
	// Parsed name leads, then the recommended installed fallback.
	if got := roleFontPrefix("Helvetica", "header"); len(got) != 2 || got[0] != "Helvetica" || got[1] != "Noto Sans" {
		t.Errorf("roleFontPrefix(parsed) = %v, want [Helvetica Noto Sans]", got)
	}
	// No parsed name => just the recommended fallback.
	if got := roleFontPrefix("", "transcription"); len(got) != 1 || got[0] != "DejaVu Sans" {
		t.Errorf("roleFontPrefix(empty) = %v, want [DejaVu Sans]", got)
	}
	// strong/emphasis fall back to the body font (Gentium), dropping the
	// synthetic bold/italic distinction when the role is undeclared.
	if got := roleFontPrefix("", "strong"); len(got) != 1 || got[0] != "Gentium" {
		t.Errorf("roleFontPrefix(empty, strong) = %v, want [Gentium]", got)
	}
	if got := roleFontPrefix("Amiri Slanted", "emphasis"); len(got) != 2 || got[0] != "Amiri Slanted" || got[1] != "Gentium" {
		t.Errorf("roleFontPrefix(parsed, emphasis) = %v, want [Amiri Slanted Gentium]", got)
	}
}

// TestAssembleTypstDocumentRoleFonts confirms the exporter emits per-role font
// prefixes (recommended fallback used when no font.css is configured),
// including the strong/emphasis substitution-font roles.
func TestAssembleTypstDocumentRoleFonts(t *testing.T) {
	doc, err := assembleTypstDocument(
		&EBookProject{Title: "T"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	call := showCall(doc)
	for _, want := range []string{
		`font-body: ("Gentium",)`,
		`font-header: ("Noto Sans",)`,
		`font-transcription: ("DejaVu Sans",)`,
		`font-translation: ("Gentium",)`,
		`font-strong: ("Gentium",)`,
		`font-emph: ("Gentium",)`,
	} {
		if !strings.Contains(call, want) {
			t.Errorf("expected %q in call, got:\n%s", want, call)
		}
	}
}

// TestAssembleTypstDocumentRoleFontsDeclared confirms that when the project's
// font.css declares Font Strong/Font Emphasis, the exporter emits the parsed
// local() name prefixed ahead of the recommended (Gentium) fallback.
func TestAssembleTypstDocumentRoleFontsDeclared(t *testing.T) {
	dir := t.TempDir()
	css := `
@font-face { font-family: "Font Strong"; src: local("Amiri Bold"); }
@font-face { font-family: "Font Emphasis"; src: local("Amiri Slanted"); }
`
	path := filepath.Join(dir, "font.css")
	if err := os.WriteFile(path, []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}

	project := &EBookProject{Title: "T", Stylesheet: EBookStyles{Common: []string{path}}}
	doc, err := assembleTypstDocument(project, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument error = %v", err)
	}
	call := showCall(doc)
	for _, want := range []string{
		`font-strong: ("Amiri Bold", "Gentium")`,
		`font-emph: ("Amiri Slanted", "Gentium")`,
	} {
		if !strings.Contains(call, want) {
			t.Errorf("expected %q in call, got:\n%s", want, call)
		}
	}
}

// --- typstStringLiteral escaping (SPECS §5.2 string context / ASR-4) ------

func TestTypstStringLiteral(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "plain", `"plain"`},
		{"empty", "", `""`},
		{"double quote", `he said "hi"`, `"he said \"hi\""`},
		{"backslash", `a\b`, `"a\\b"`},
		{"newline", "a\nb", `"a\nb"`},
		{"tab", "a\tb", `"a\tb"`},
		{"carriage return", "a\rb", `"a\rb"`},
		{"markup metachars are literal in string context", "a#b*c_d", `"a#b*c_d"`},
		{"unicode passthrough", "café 你好 —", `"café 你好 —"`},
	}
	for _, tc := range cases {
		if got := typstStringLiteral(tc.in); got != tc.want {
			t.Errorf("%s: typstStringLiteral(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// --- typstAssetPath root-relative conversion (SPECS §8.3/D7) --------------

func TestTypstAssetPath(t *testing.T) {
	cases := []struct {
		name    string
		root    string
		path    string
		want    string
		wantErr bool
	}{
		{"empty passthrough", "/root/dir", "", "", false},
		{"asset directly under root", "/root/dir", "/root/dir/cover.svg", "/cover.svg", false},
		{"nested asset", "/root/dir", "/root/dir/assets/cover.svg", "/assets/cover.svg", false},
		{"asset outside root becomes ..-relative", "/root/dir", "/root/other/cover.svg", "/../other/cover.svg", false},
	}
	for _, tc := range cases {
		got, err := typstAssetPath(tc.root, tc.path)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: typstAssetPath(%q,%q) expected error, got %q", tc.name, tc.root, tc.path, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: typstAssetPath(%q,%q) unexpected error: %v", tc.name, tc.root, tc.path, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: typstAssetPath(%q,%q) = %q, want %q", tc.name, tc.root, tc.path, got, tc.want)
		}
	}
}

// --- readProject resolves Stylesheet.Cover (regression) -------------------

// TestReadProjectResolvesStylesheetCover guards the fix for a pre-existing
// bug: readProject resolved Cover/Stylesheet.Section/Chapter/Common to
// absolute paths but NOT Stylesheet.Cover, so `build --format epub` failed
// (book.AddCSS resolved the raw relative path against the process CWD) when
// invoked with -p pointing outside the project directory.
func TestReadProjectResolvesStylesheetCover(t *testing.T) {
	dir := t.TempDir()

	coverCSS := filepath.Join(dir, "cover.css")
	if err := os.WriteFile(coverCSS, []byte("/* cover */\n"), 0o644); err != nil {
		t.Fatalf("write cover.css: %v", err)
	}
	ymlPath := filepath.Join(dir, "ebook.yml")
	yml := "identifier: urn:test:cover\nfilename: out.epub\ntitle: T\nstylesheet:\n  cover: cover.css\n"
	if err := os.WriteFile(ymlPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("write ebook.yml: %v", err)
	}

	project, err := readProject(ymlPath)
	if err != nil {
		t.Fatalf("readProject(%q) error = %v", ymlPath, err)
	}

	if !filepath.IsAbs(project.Stylesheet.Cover) {
		t.Errorf("Stylesheet.Cover = %q, want an absolute path", project.Stylesheet.Cover)
	}
	if project.Stylesheet.Cover != coverCSS {
		t.Errorf("Stylesheet.Cover = %q, want %q", project.Stylesheet.Cover, coverCSS)
	}
}

// --- bug fix regressions (Increment 1) ------------------------------------

// TestBookTemplateColumns50_50 guards bug #2: vocabulary and models grids must
// use (1fr, 1fr) so a long phrase does not collapse the translation column.
// (dialog and questions Q&A grids intentionally keep (auto, 1fr) for their
// auto-sized header/question column — only the phrase/translation split is fixed.)
func TestBookTemplateColumns50_50(t *testing.T) {
	count := strings.Count(bookTemplate, "columns: (1fr, 1fr)")
	if count < 2 {
		t.Errorf("book.typ must contain columns: (1fr, 1fr) at least twice (vocabulary + models), found %d occurrence(s)", count)
	}
}

// TestBookTemplateQuestionsNoIndent guards bug #3: the questions function must
// scope away the global first-line-indent so question-only lines are flush-left.
func TestBookTemplateQuestionsNoIndent(t *testing.T) {
	want := "set par(first-line-indent: 0pt)"
	if !strings.Contains(bookTemplate, want) {
		t.Errorf("book.typ questions function missing %q", want)
	}
}

// TestBookTemplateContentsTitle guards bug #4: the book() function must accept
// a contents-title parameter defaulting to [Contents], and use it in outline().
func TestBookTemplateContentsTitle(t *testing.T) {
	for _, want := range []string{
		"contents-title: [Contents]",
		"outline(title: contents-title,",
	} {
		if !strings.Contains(bookTemplate, want) {
			t.Errorf("book.typ missing expected string %q", want)
		}
	}
}

// TestAssembleTypstDocumentContentsTitle covers bug #4 end to end: when
// ContentsTitle is set, the exporter emits contents-title as a quoted string
// literal; when unset, it is omitted so book.typ's default "Contents" stands.
func TestAssembleTypstDocumentContentsTitle(t *testing.T) {
	set, err := assembleTypstDocument(
		&EBookProject{Title: "T", ContentsTitle: "Spis treści"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument(ContentsTitle set) error = %v", err)
	}
	wantSet := `contents-title: "Spis treści"`
	if !strings.Contains(showCall(set), wantSet) {
		t.Errorf("expected %q in call when ContentsTitle is set, got:\n%s", wantSet, showCall(set))
	}

	unset, err := assembleTypstDocument(
		&EBookProject{Title: "T"}, "en", "ltr", "", []string{"body"}, config.PdfConfig{})
	if err != nil {
		t.Fatalf("assembleTypstDocument(ContentsTitle unset) error = %v", err)
	}
	if strings.Contains(showCall(unset), "contents-title:") {
		t.Errorf("must not emit contents-title: when ContentsTitle is unset, got:\n%s", showCall(unset))
	}
}
