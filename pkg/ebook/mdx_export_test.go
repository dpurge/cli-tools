package ebook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture writes content to filepath.Join(dir, name) and returns the
// resulting absolute path, failing the test on any write error. Shared by
// every synthetic-project test below (mirrors the writeMD helper already
// used by typst_export_test.go's TestEPUBExporterGlobalChapterNumbering).
func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return path
}

// assertFileExists fails the test if path does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %q to exist: %v", path, err)
	}
}

// --- synthetic single-section project (SPECS §10, PLAN T7/T9) ------------

// TestMdxExporterSingleSection builds a synthetic 1-section/2-chapter
// project entirely inside t.TempDir() and asserts the exporter's directory
// layout, _category_.json shape (fixed key order, no slug), and each
// chapter's frontmatter + <Text>-wrapped body + vocabulary fence (SPECS
// §7.2-§7.5).
func TestMdxExporterSingleSection(t *testing.T) {
	dir := t.TempDir()

	sectionFile := writeFixture(t, dir, "section.md", "# My Section\n\nIntro text.\n")
	ch1File := writeFixture(t, dir, "01.md",
		"# I\n\n{start-vocabulary}\nbağçe = ogród\n{end-vocabulary}\n\n---\n\nSome reading text.\n")
	ch2File := writeFixture(t, dir, "02.md", "# II\n\nMore text.\n")

	project := &EBookProject{
		Identifier:  "urn:test:mdx",
		Filename:    filepath.Join(dir, "book.epub"),
		Title:       "Test Book",
		Language:    "lat",
		Script:      "latn",
		Description: "A test description.",
		Text: [][]string{
			{sectionFile, ch1File, ch2File},
		},
	}

	outDir, err := (mdxExporter{}).Export(project)
	if err != nil {
		t.Fatalf("mdxExporter.Export() error = %v", err)
	}

	wantDir := filepath.Join(dir, "book-mdx")
	if outDir != wantDir {
		t.Fatalf("Export() = %q, want %q", outDir, wantDir)
	}
	if info, err := os.Stat(outDir); err != nil || !info.IsDir() {
		t.Fatalf("output dir %q does not exist or is not a directory: %v", outDir, err)
	}

	// --- _category_.json ---------------------------------------------
	catPath := filepath.Join(outDir, "_category_.json")
	catBytes, err := os.ReadFile(catPath)
	if err != nil {
		t.Fatalf("read %s: %v", catPath, err)
	}

	var cat map[string]any
	if err := json.Unmarshal(catBytes, &cat); err != nil {
		t.Fatalf("_category_.json is not valid JSON: %v\n%s", err, catBytes)
	}
	if cat["label"] != "My Section" {
		t.Errorf("_category_.json label = %v, want %q", cat["label"], "My Section")
	}
	if pos, ok := cat["position"].(float64); !ok || pos != 1 {
		t.Errorf("_category_.json position = %v, want 1", cat["position"])
	}
	if _, ok := cat["slug"]; ok {
		t.Errorf("_category_.json must not contain a slug key: %v", cat)
	}
	link, ok := cat["link"].(map[string]any)
	if !ok {
		t.Fatalf("_category_.json link is not an object: %v", cat)
	}
	if link["type"] != "generated-index" {
		t.Errorf("link.type = %v, want %q", link["type"], "generated-index")
	}
	if link["title"] != "My Section" {
		t.Errorf("link.title = %v, want %q", link["title"], "My Section")
	}
	if link["description"] != "A test description." {
		t.Errorf("link.description = %v, want %q", link["description"], "A test description.")
	}

	// Fixed key order (ASR-3): label, then position, then link - asserted
	// against the raw serialized text since map iteration in Go is
	// unordered and cannot itself prove ordering.
	catStr := string(catBytes)
	labelIdx := strings.Index(catStr, `"label"`)
	positionIdx := strings.Index(catStr, `"position"`)
	linkIdx := strings.Index(catStr, `"link"`)
	if labelIdx < 0 || positionIdx < 0 || linkIdx < 0 || !(labelIdx < positionIdx && positionIdx < linkIdx) {
		t.Errorf("_category_.json key order is not label, position, link:\n%s", catStr)
	}

	// --- 01.mdx ---------------------------------------------------------
	ch1Out := filepath.Join(outDir, "01.mdx")
	ch1Bytes, err := os.ReadFile(ch1Out)
	if err != nil {
		t.Fatalf("read %s: %v", ch1Out, err)
	}
	ch1Str := string(ch1Bytes)

	if !strings.HasPrefix(ch1Str, "---\n") {
		t.Errorf("01.mdx missing opening frontmatter fence:\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, `title: "I"`) {
		t.Errorf("01.mdx frontmatter missing title \"I\":\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, `description: "A test description."`) {
		t.Errorf("01.mdx frontmatter missing description:\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, `<Text lang="lat" script="latn">`) {
		t.Errorf("01.mdx body missing <Text lang=\"lat\" script=\"latn\">:\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, "```vocabulary lang=lat script=latn") {
		t.Errorf("01.mdx body missing vocabulary fence header:\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, "bağçe = ogród") {
		t.Errorf("01.mdx body missing vocabulary content:\n%s", ch1Str)
	}

	// --- 02.mdx (no vocabulary, still gets frontmatter + <Text>) ---------
	ch2Out := filepath.Join(outDir, "02.mdx")
	ch2Bytes, err := os.ReadFile(ch2Out)
	if err != nil {
		t.Fatalf("read %s: %v", ch2Out, err)
	}
	ch2Str := string(ch2Bytes)
	if !strings.Contains(ch2Str, `title: "II"`) {
		t.Errorf("02.mdx frontmatter missing title \"II\":\n%s", ch2Str)
	}
	if !strings.Contains(ch2Str, `<Text lang="lat" script="latn">`) {
		t.Errorf("02.mdx body missing <Text>:\n%s", ch2Str)
	}
}

// TestMdxExporterChapterTitleFallsBackToBasename covers SPECS §7.4/§9: a
// chapter file with no H1 falls back to its basename (without extension)
// for the frontmatter title, rather than erroring or emitting an empty
// title.
func TestMdxExporterChapterTitleFallsBackToBasename(t *testing.T) {
	dir := t.TempDir()

	sectionFile := writeFixture(t, dir, "section.md", "# Section\n\nIntro.\n")
	chapterFile := writeFixture(t, dir, "no-heading.md", "Just a paragraph, no H1.\n")

	project := &EBookProject{
		Filename: filepath.Join(dir, "book.epub"),
		Language: "lat",
		Script:   "latn",
		Text:     [][]string{{sectionFile, chapterFile}},
	}

	outDir, err := (mdxExporter{}).Export(project)
	if err != nil {
		t.Fatalf("mdxExporter.Export() error = %v", err)
	}

	out := filepath.Join(outDir, "no-heading.mdx")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read %s: %v", out, err)
	}
	if !strings.Contains(string(data), `title: "no-heading"`) {
		t.Errorf("no-heading.mdx frontmatter title did not fall back to the basename:\n%s", data)
	}
}

// --- synthetic multi-section project (SPECS §7.2/Decision D7) ------------

// TestMdxExporterMultiSection asserts the per-section-subfolder layout that
// applies once a project has more than one section: each section gets its
// own numbered, slugged subfolder with its own _category_.json and chapter
// .mdx files, guarding the basename collision the flat layout alone would
// risk (SPECS §7.2/D7).
func TestMdxExporterMultiSection(t *testing.T) {
	dir := t.TempDir()

	sec1 := writeFixture(t, dir, "sec1.md", "# Section One\n\nIntro one.\n")
	ch1 := writeFixture(t, dir, "ch1.md", "# Chapter One\n\nBody one.\n")
	sec2 := writeFixture(t, dir, "sec2.md", "# Section Two\n\nIntro two.\n")
	ch2 := writeFixture(t, dir, "ch2.md", "# Chapter Two\n\nBody two.\n")

	project := &EBookProject{
		Filename: filepath.Join(dir, "multi.epub"),
		Language: "lat",
		Script:   "latn",
		Text: [][]string{
			{sec1, ch1},
			{sec2, ch2},
		},
	}

	outDir, err := (mdxExporter{}).Export(project)
	if err != nil {
		t.Fatalf("mdxExporter.Export() error = %v", err)
	}

	// No files directly loose in outDir - it should contain only the two
	// per-section subfolders.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read output dir %q: %v", outDir, err)
	}
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() {
			subdirs = append(subdirs, e.Name())
		} else {
			t.Errorf("unexpected loose file %q directly in the multi-section output dir", e.Name())
		}
	}
	if len(subdirs) != 2 {
		t.Fatalf("expected 2 section subfolders, got %v", subdirs)
	}

	wantSub1 := "01-section-one"
	wantSub2 := "02-section-two"
	sub1Found, sub2Found := false, false
	for _, sd := range subdirs {
		switch sd {
		case wantSub1:
			sub1Found = true
			assertFileExists(t, filepath.Join(outDir, sd, "_category_.json"))
			assertFileExists(t, filepath.Join(outDir, sd, "ch1.mdx"))
		case wantSub2:
			sub2Found = true
			assertFileExists(t, filepath.Join(outDir, sd, "_category_.json"))
			assertFileExists(t, filepath.Join(outDir, sd, "ch2.mdx"))
		}
	}
	if !sub1Found || !sub2Found {
		t.Fatalf("expected subfolders %q and %q, got %v", wantSub1, wantSub2, subdirs)
	}

	// The second section's category position should reflect its own
	// SectionIdx (2), not the fixed "1" from the single-section example -
	// this is the exporter's natural, non-breaking generalization of SPECS
	// §7.3's literal single-section example.
	cat2Path := filepath.Join(outDir, wantSub2, "_category_.json")
	cat2Bytes, err := os.ReadFile(cat2Path)
	if err != nil {
		t.Fatalf("read %s: %v", cat2Path, err)
	}
	var cat2 map[string]any
	if err := json.Unmarshal(cat2Bytes, &cat2); err != nil {
		t.Fatalf("%s is not valid JSON: %v", cat2Path, err)
	}
	if pos, ok := cat2["position"].(float64); !ok || pos != 2 {
		t.Errorf("%s position = %v, want 2", cat2Path, cat2["position"])
	}
	if cat2["label"] != "Section Two" {
		t.Errorf("%s label = %v, want %q", cat2Path, cat2["label"], "Section Two")
	}
}

// --- build-cmd.go --format mdx dispatch (SPECS §8, FR-11) -----------------

func TestExporterForMdx(t *testing.T) {
	exp, err := exporterFor("mdx")
	if err != nil {
		t.Fatalf("exporterFor(\"mdx\") error = %v", err)
	}
	if _, ok := exp.(mdxExporter); !ok {
		t.Errorf("exporterFor(\"mdx\") = %T, want mdxExporter", exp)
	}

	if _, err := exporterFor("bogus"); err == nil {
		t.Error("exporterFor(\"bogus\") expected an error, got nil")
	} else if !strings.Contains(err.Error(), "epub|pdf|mdx") {
		t.Errorf("exporterFor(\"bogus\") error = %v, want it to mention epub|pdf|mdx", err)
	}
}

// --- integration: real tur sample -> MDX (SPECS §10/AC6) ------------------

// TestMdxExporterIntegrationTurSample builds the real tur sample project to
// MDX via mdxExporter, skipping cleanly if the sibling epub-public checkout
// isn't present (mirrors typst_export_test.go's
// TestTypstExporterIntegrationTurSample/findTurProjectFile pattern).
//
// project.Filename is retargeted into a t.TempDir() BEFORE calling Export:
// mdxExporter derives its entire output directory from project.Filename
// alone (derivedMdxDir) and never writes anywhere else, so this guarantees
// every file this test produces lands under the temp dir and the
// epub-public checkout is never written to. project.Text/Cover/Stylesheet
// keep pointing at the real sample files, which mdxExporter only reads
// (never writes) - so loading the project via the normal readProject (which
// validates Cover/Stylesheet/Font/Image paths against the real, intact
// on-disk layout) is safe and avoids having to replicate the sample's
// relative-path asset tree (stylesheets reach three directories up to a
// shared css/ folder) inside a temp copy.
func TestMdxExporterIntegrationTurSample(t *testing.T) {
	projectFile := findTurProjectFile()
	if projectFile == "" {
		t.Skip("tur sample project not found (expected a sibling epub-public checkout at ../../../epub-public); skipping MDX integration test")
	}

	project, err := readProject(projectFile)
	if err != nil {
		t.Fatalf("readProject(%q) error = %v", projectFile, err)
	}

	tmp := t.TempDir()
	project.Filename = filepath.Join(tmp, filepath.Base(project.Filename))

	outDir, err := (mdxExporter{}).Export(project)
	if err != nil {
		t.Fatalf("mdxExporter.Export() error = %v", err)
	}

	wantDir := filepath.Join(tmp, "turkish-notes-private-mdx")
	if outDir != wantDir {
		t.Fatalf("Export() = %q, want %q", outDir, wantDir)
	}

	assertFileExists(t, filepath.Join(outDir, "_category_.json"))
	assertFileExists(t, filepath.Join(outDir, "02.mdx"))

	ch1Path := filepath.Join(outDir, "01.mdx")
	ch1Bytes, err := os.ReadFile(ch1Path)
	if err != nil {
		t.Fatalf("read %s: %v", ch1Path, err)
	}
	ch1Str := string(ch1Bytes)

	if !strings.Contains(ch1Str, `title: "I"`) {
		t.Errorf("01.mdx frontmatter missing title \"I\":\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, "```vocabulary lang=tur script=latn") {
		t.Errorf("01.mdx missing vocabulary fence header:\n%s", ch1Str)
	}
	if !strings.Contains(ch1Str, "bağçe = ogród") {
		t.Errorf("01.mdx missing vocabulary content \"bağçe = ogród\":\n%s", ch1Str)
	}
	textTag := `<Text lang="tur" script="latn">`
	if !strings.Contains(ch1Str, textTag) {
		t.Errorf("01.mdx missing %s:\n%s", textTag, ch1Str)
	}

	// No literal '{' / unescaped '<' leaking into the <Text> prose (SPECS
	// §5.2/ASR-4/AC4) - bounded strictly to the <Text>...</Text> span, since
	// the vocabulary fence (emitted standalone, outside <Text>) is literal
	// by design and may legitimately be adjacent in the file.
	textStart := strings.Index(ch1Str, textTag)
	textEnd := strings.Index(ch1Str, "</Text>")
	if textStart < 0 || textEnd < 0 || textEnd < textStart {
		t.Fatalf("could not locate a <Text>...</Text> span in 01.mdx:\n%s", ch1Str)
	}
	textBody := ch1Str[textStart+len(textTag) : textEnd]
	if strings.ContainsRune(textBody, '{') {
		t.Errorf("<Text> body contains a literal '{' (MDX-unsafe): %q", textBody)
	}
	if strings.ContainsRune(textBody, '<') {
		t.Errorf("<Text> body contains an unescaped '<' (MDX-unsafe): %q", textBody)
	}
}

// --- mdxYamlString escaping (SPECS §5.4 — the frontmatter injection-safety
// function) ---------------------------------------------------------------

func TestMdxYamlString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "Hello", `"Hello"`},
		{"empty", "", `""`},
		{"double quote escaped", `say "hi"`, `"say \"hi\""`},
		{"backslash escaped", `a\b`, `"a\\b"`},
		{"newline collapses to space", "a\nb", `"a b"`},
		{"tab collapses to space", "a\tb", `"a b"`},
		{"trailing block-scalar newline trimmed", "Description.\n", `"Description."`},
		{"leading/trailing space trimmed", "  x  ", `"x"`},
		{"unicode passthrough", "Żaba 你好", `"Żaba 你好"`},
	}
	for _, tc := range cases {
		if got := mdxYamlString(tc.in); got != tc.want {
			t.Errorf("%s: mdxYamlString(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

// --- sectionSlug / slugify (SPECS §7.2/D7 multi-section dir naming) --------

func TestSectionSlug(t *testing.T) {
	cases := []struct {
		name        string
		idx         int
		title       string
		sectionFile string
		want        string
	}{
		{"title slugified with numeric prefix", 1, "Section One", "/x/sec.md", "01-section-one"},
		{"non-ascii title degrades but numeric prefix keeps it unique", 2, "Żaba", "/x/z.md", "02-aba"},
		{"empty title falls back to file basename", 3, "", "/x/intro.md", "03-intro"},
		{"empty title and basename fall back to 'section'", 4, "", "", "04-section"},
	}
	for _, tc := range cases {
		if got := sectionSlug(tc.idx, tc.title, tc.sectionFile); got != tc.want {
			t.Errorf("%s: sectionSlug(%d,%q,%q) = %q, want %q", tc.name, tc.idx, tc.title, tc.sectionFile, got, tc.want)
		}
	}
}
