package ebook

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// bookTemplate is the embedded Typst preamble (templates/book.typ) defining the
// book/vocabulary/dialog/parallel functions and the page/typography defaults.
//
//go:embed templates/book.typ
var bookTemplate string

// typstExporter implements Exporter by assembling one self-contained Typst
// document and compiling it with the `typst` binary.
type typstExporter struct{}

func (typstExporter) Export(project *EBookProject) (string, error) {
	lang, dir := languageInfo(project.Language, project.Script)

	items := WalkTexts(project.Text)
	bodies := make([]string, 0, len(items))
	for _, item := range items {
		content, err := markdown.FileToTypst(item.File)
		if err != nil {
			return "", err
		}
		bodies = append(bodies, content)
	}

	pdfPath, typPath := derivedTypstPaths(project.Filename)
	rootDir := filepath.Dir(pdfPath)

	cover, err := typstAssetPath(rootDir, project.Cover)
	if err != nil {
		return "", err
	}

	document, err := assembleTypstDocument(project, lang, dir, cover, bodies, config.GetPdfConfig())
	if err != nil {
		return "", err
	}

	// Left in place (not a temp file) so it survives for debugging a compile
	// failure (SPECS §9).
	if err := os.WriteFile(typPath, []byte(document), 0o644); err != nil {
		return "", err
	}

	typstPath, err := locateTypst()
	if err != nil {
		return "", err
	}

	args := []string{"compile", typPath, pdfPath, "--root", rootDir}
	for _, fontDir := range fontPathDirs(project.Font) {
		args = append(args, "--font-path", fontDir)
	}

	output, err := runTypst(typstPath, args...)
	if err != nil {
		return "", fmt.Errorf("typst compile failed: %s", output)
	}
	// Surface Typst's warnings (e.g. missing-font substitutions) on a successful
	// compile; otherwise they are silently discarded.
	if strings.TrimSpace(output) != "" {
		fmt.Fprint(os.Stderr, output)
	}

	return pdfPath, nil
}

// assembleTypstDocument builds the full `.typ` source: the embedded book.typ
// preamble, a `#show: book.with(...)` call over the EBookProject fields and
// optional Pdf overrides, and the bodies joined by weak pagebreaks. Empty
// override fields are omitted so book.typ's defaults stand. Returns an error
// for a malformed configured length so a typo fails the build clearly.
func assembleTypstDocument(project *EBookProject, lang, dir, cover string, bodies []string, cfg config.PdfConfig) (string, error) {
	var doc strings.Builder

	doc.WriteString(bookTemplate)
	doc.WriteString("\n#show: book.with(\n")
	doc.WriteString("  title: " + typstStringLiteral(project.Title) + ",\n")
	// author is ALWAYS passed, even as "": book.typ does `set document(author:
	// author)`, and Typst's document() rejects `none` (compile error), so an
	// omitted author would break every author-less build.
	doc.WriteString("  author: " + typstStringLiteral(project.Author) + ",\n")
	if project.Description != "" {
		doc.WriteString("  description: " + typstStringLiteral(project.Description) + ",\n")
	}
	// lang is emitted as the bare primary subtag (typstLang): `set text(lang:)`
	// wants the ISO 639 subtag, not the full BCP-47 tag languageInfo returns.
	doc.WriteString("  lang: " + typstStringLiteral(typstLang(lang)) + ",\n")
	// dir MUST be emitted unquoted: book.typ's `dir` is Typst's `direction`
	// type (bare ltr/rtl keywords), not a string.
	doc.WriteString("  dir: " + dir + ",\n")
	// Signals book.typ to use its enlarged body size for scripts that read
	// poorly at base size (CJK, Arabic, Hebrew). Emitted unquoted — it's a boolean.
	if largeScript(project.Script) {
		doc.WriteString("  large-script: true,\n")
	}
	if cover != "" {
		doc.WriteString("  cover: " + typstStringLiteral(cover) + ",\n")
	}

	// Optional Pdf overrides. paper/font are quoted (injection-safe); lengths
	// are emitted unquoted so each is validated first (typstLength).
	if cfg.Paper != "" {
		doc.WriteString("  paper: " + typstStringLiteral(cfg.Paper) + ",\n")
	}
	writeLen := func(configKey, value, arg string) error {
		if value == "" {
			return nil
		}
		v, err := typstLength(configKey, value)
		if err != nil {
			return err
		}
		doc.WriteString("  " + arg + ": " + v + ",\n")
		return nil
	}
	if err := writeLen("Pdf.size", cfg.Size, "size"); err != nil {
		return "", err
	}
	if err := writeLen("Pdf.sizeLarge", cfg.SizeLarge, "size-large"); err != nil {
		return "", err
	}
	margin, err := typstMarginDict(cfg.Margin)
	if err != nil {
		return "", err
	}
	if margin != "" {
		doc.WriteString("  margin: " + margin + ",\n")
	}
	if len(cfg.Font) > 0 {
		doc.WriteString("  font: " + typstFontArray(cfg.Font) + ",\n")
	}

	// Per-role fonts from font.css, prepended in book.typ so the PDF mirrors the
	// EPUB CSS roles; a recommended font fills in when a role is undeclared.
	roles := parseFontRoles(project.Stylesheet.Common)
	for _, r := range []struct{ arg, parsed, key string }{
		{"font-body", roles.body, "body"},
		{"font-header", roles.header, "header"},
		{"font-transcription", roles.transcription, "transcription"},
		{"font-translation", roles.translation, "translation"},
		{"font-strong", roles.strong, "strong"},
		{"font-emph", roles.emph, "emphasis"},
	} {
		if stack := roleFontPrefix(r.parsed, r.key); len(stack) > 0 {
			doc.WriteString("  " + r.arg + ": " + typstFontArray(stack) + ",\n")
		}
	}

	doc.WriteString(")\n\n")

	doc.WriteString(strings.Join(bodies, "\n#pagebreak(weak: true)\n\n"))
	doc.WriteString("\n")

	return doc.String(), nil
}

// typstLengthRe matches accepted Typst length literals (number + unit). It
// rejects typos early and keeps arbitrary config text from being emitted
// unquoted into Typst source.
var typstLengthRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(pt|mm|cm|in|em)$`)

// typstLength validates a configured Typst length (e.g. "12pt") and returns it
// trimmed, ready to emit unquoted. configKey names the field in the error.
func typstLength(configKey, value string) (string, error) {
	v := strings.TrimSpace(value)
	if !typstLengthRe.MatchString(v) {
		return "", fmt.Errorf("invalid Typst length for %s: %q (want e.g. 12pt, 1cm, 1.5in)", configKey, value)
	}
	return v, nil
}

// defaultMargin is the neutral `rest:` value for sides left unset when the
// config specifies at least one margin (with no margin at all, book.typ's own
// asymmetric binding-aware default applies instead).
const defaultMargin = "1.5cm"

// typstMarginDict builds the Typst `margin` dictionary from the per-side config.
// Returns "" when no side is set (caller omits the argument, keeping book.typ's
// default); otherwise a `rest:` entry covers the unconfigured sides.
func typstMarginDict(m config.PdfMargin) (string, error) {
	sides := []struct{ name, value string }{
		{"top", m.Top}, {"bottom", m.Bottom},
		{"left", m.Left}, {"right", m.Right},
		{"inside", m.Inside}, {"outside", m.Outside},
	}
	var parts []string
	for _, s := range sides {
		if strings.TrimSpace(s.value) == "" {
			continue
		}
		v, err := typstLength("Pdf.margin."+s.name, s.value)
		if err != nil {
			return "", err
		}
		parts = append(parts, s.name+": "+v)
	}
	if len(parts) == 0 {
		return "", nil
	}
	parts = append(parts, "rest: "+defaultMargin)
	return "(" + strings.Join(parts, ", ") + ")", nil
}

// typstFontArray renders a font family list as a Typst array. A single family
// still emits array syntax (trailing comma) so book.typ always receives an
// array, never a parenthesised bare string.
func typstFontArray(fonts []string) string {
	lits := make([]string, len(fonts))
	for i, f := range fonts {
		lits[i] = typstStringLiteral(f)
	}
	joined := strings.Join(lits, ", ")
	if len(lits) == 1 {
		joined += ","
	}
	return "(" + joined + ")"
}

// fontRoles holds the font each @font-face role in a project's font.css
// resolves to (via its local(...) name); empty means the role is undeclared.
type fontRoles struct {
	header, body, transcription, translation, strong, emph string
}

// recommendedRoleFont is the installed Latin fallback prepended per role when
// font.css is absent or omits it; non-Latin glyphs resolve via book.typ's stack.
// strong/emphasis fall back to the body font, so an undeclared role drops the
// synthetic bold/italic distinction rather than inventing an unrelated one.
var recommendedRoleFont = map[string]string{
	"header":        "Noto Sans",
	"body":          "Gentium",
	"transcription": "DejaVu Sans",
	"translation":   "Gentium",
	"strong":        "Gentium",
	"emphasis":      "Gentium",
}

var (
	fontFaceRe   = regexp.MustCompile(`(?s)@font-face\s*\{(.*?)\}`)
	fontFamilyRe = regexp.MustCompile(`font-family\s*:\s*["']([^"']+)["']`)
	fontLocalRe  = regexp.MustCompile(`local\(\s*["']?([^"')]+?)["']?\s*\)`)
)

// parseFontRoles reads the project's font.css and returns the local() font each
// Font Header/Body/Transcription/Translation @font-face resolves to.
func parseFontRoles(stylesheetPaths []string) fontRoles {
	var roles fontRoles
	var cssPath string
	for _, p := range stylesheetPaths {
		if strings.EqualFold(filepath.Base(p), "font.css") {
			cssPath = p
			break
		}
	}
	if cssPath == "" {
		return roles
	}
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return roles
	}
	for _, block := range fontFaceRe.FindAllStringSubmatch(string(data), -1) {
		fam := fontFamilyRe.FindStringSubmatch(block[1])
		loc := fontLocalRe.FindStringSubmatch(block[1])
		if fam == nil || loc == nil {
			continue
		}
		name := strings.TrimSpace(loc[1])
		switch strings.ToLower(strings.TrimSpace(fam[1])) {
		case "font header":
			roles.header = name
		case "font body":
			roles.body = name
		case "font transcription":
			roles.transcription = name
		case "font translation":
			roles.translation = name
		case "font strong":
			roles.strong = name
		case "font emphasis":
			roles.emph = name
		}
	}
	return roles
}

// roleFontPrefix builds the per-role prefix book.typ prepends to its base stack:
// the font.css name (if any) then the recommended fallback.
func roleFontPrefix(parsed, role string) []string {
	var stack []string
	if parsed != "" {
		stack = append(stack, parsed)
	}
	if fb := recommendedRoleFont[role]; fb != "" {
		stack = append(stack, fb)
	}
	return stack
}

// typstLang reduces a BCP-47 tag to the bare primary subtag `set text(lang:)`
// accepts (an ISO 639 code): languageInfo returns full tags like "zh-Hans",
// whose script subtag Typst can't express and is carried by font selection.
func typstLang(tag string) string {
	if i := strings.IndexByte(tag, '-'); i >= 0 {
		return tag[:i]
	}
	return tag
}

// typstFontFamilies returns the font family names Typst can see (`typst fonts`),
// lowercased — the same view a real build resolves against, so `doctor` checks
// against exactly that.
func typstFontFamilies(typstPath string) (map[string]bool, error) {
	out, err := runTypst(typstPath, "fonts")
	if err != nil {
		return nil, fmt.Errorf("listing typst fonts: %s", strings.TrimSpace(out))
	}
	families := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			families[strings.ToLower(name)] = true
		}
	}
	return families, nil
}

// largeScriptCodes is the set of ISO 15924 script codes (lowercase) that select
// book.typ's enlarged body size, including syllabary aliases.
var largeScriptCodes = map[string]bool{
	"hans": true, // Chinese, simplified Han
	"hant": true, // Chinese, traditional Han
	"hani": true, // Han (script unspecified)
	"arab": true, // Arabic
	"hebr": true, // Hebrew
	"kore": true, // Korean (Hangul + Hanja)
	"hang": true, // Hangul
	"jpan": true, // Japanese (Han + Kana)
	"hira": true, // Hiragana
	"kana": true, // Katakana
}

// largeScript reports whether an ISO 15924 script code selects book.typ's
// enlarged body size. Matching is case-insensitive and whitespace-tolerant.
func largeScript(script string) bool {
	return largeScriptCodes[strings.ToLower(strings.TrimSpace(script))]
}

// typstAssetPath converts an absolute filesystem path into a Typst root-relative
// path: a leading "/" in a Typst path means "relative to --root", not the OS
// root, so emitting the raw OS-absolute path fails with "file not found".
// Returns "" unchanged for an unset cover.
func typstAssetPath(rootDir, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return "", err
	}
	return "/" + filepath.ToSlash(rel), nil
}

// typstStringLiteral quotes s as a Typst string literal, delegating body
// escaping to the shared tool.EscapeQuoted (also used by scanbook's JS writer,
// which differs only in quote character).
func typstStringLiteral(s string) string {
	return `"` + tool.EscapeQuoted(s, '"') + `"`
}

// derivedTypstPaths derives the output PDF and generated .typ paths from the
// EPUB filename, placing both next to the EPUB (SPECS §8.4/D7).
func derivedTypstPaths(epubFilename string) (pdfPath, typPath string) {
	base := epubFilename
	if strings.HasSuffix(base, ".epub") {
		base = strings.TrimSuffix(base, ".epub")
	} else {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return base + ".pdf", base + ".typ"
}

// fontPathDirs returns the de-duplicated, order-preserving directories
// containing project.Font's files, one per `--font-path` argument.
func fontPathDirs(fontFiles []string) []string {
	var dirs []string
	seen := make(map[string]bool, len(fontFiles))
	for _, f := range fontFiles {
		d := filepath.Dir(f)
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// locateTypst finds the typst binary via config (Typst.typst) then PATH. The
// explicit exec.LookPath covers an entirely absent config file, where
// GetToolPath returns an error rather than resolving via PATH.
func locateTypst() (string, error) {
	if path, err := config.GetToolPath("Typst", "typst"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("typst"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("typst not found: set Typst.typst in config or install on PATH (https://typst.app)")
}

// runTypst runs the resolved typst binary with args, mirroring tool.RunCmd but
// taking an already-resolved path instead of doing a config lookup.
func runTypst(path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	buf, err := cmd.CombinedOutput()
	return string(buf), err
}
