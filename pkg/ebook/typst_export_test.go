package ebook

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
