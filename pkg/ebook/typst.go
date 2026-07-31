package ebook

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	if project.ContentsTitle != "" {
		doc.WriteString("  contents-title: " + typstStringLiteral(project.ContentsTitle) + ",\n")
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
	// SPECS §8.3: the 6 legacy base-role args stay exactly as before (ASR-1);
	// the qualified table (below) is purely additive.
	table := parseFontRoles(project.Stylesheet.Common)
	for _, r := range []struct{ arg, key string }{
		{"font-body", "body"},
		{"font-header", "header"},
		{"font-transcription", "transcription"},
		{"font-translation", "translation"},
		{"font-strong", "strong"},
		{"font-emph", "emphasis"},
	} {
		if stack := roleFontPrefix(table.BaseRole(r.key), r.key); len(stack) > 0 {
			doc.WriteString("  " + r.arg + ": " + typstFontArray(stack) + ",\n")
		}
	}

	// SPECS §8.2/§8.3: the full qualified slot table, for book.typ's
	// #_resolveFont to look up script/extension/field/style-qualified
	// families at compile time (per-block resolution, since only the
	// generated .typ block calls -- not this Go-side assembly -- know each
	// block's own script/field). book-script is passed through for
	// forward-compat/debugging ONLY (SPECS A2: resolution keys exclusively
	// on each block's OWN resolved script, never the book's, so book-script
	// deliberately does not participate in _resolveFont).
	if slots := table.Slots(); len(slots) > 0 {
		doc.WriteString("  font-slots: " + typstFontSlotsDict(slots) + ",\n")
	}
	if project.Script != "" {
		doc.WriteString("  book-script: " + typstStringLiteral(strings.ToLower(project.Script)) + ",\n")
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

// typstFontSlotsDict renders a canonical-key -> family map as a Typst
// dictionary literal, sorted by key for deterministic (golden-test-stable)
// output. Values are plain strings (not arrays): #_resolveFont in book.typ
// appends the recommended fallback itself, mirroring how the base-role args
// above are shaped by roleFontPrefix rather than pre-baked here.
func typstFontSlotsDict(slots map[string]string) string {
	keys := make([]string, 0, len(slots))
	for k := range slots {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, typstStringLiteral(k)+": "+typstStringLiteral(slots[k]))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// recommendedRoleFont is the installed Latin fallback prepended per role when
// font.css is absent or omits it; non-Latin glyphs resolve via book.typ's stack.
// strong/emphasis fall back to the body font, so an undeclared role drops the
// synthetic bold/italic distinction rather than inventing an unrelated one.
// Keys are the 6 legacy base-role names (SPECS §3's "Base role" vocabulary,
// lowercased) — also used as FontTable's base-role fallback keys (§4 level 4).
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

// SPECS §3's four disjoint segment vocabularies, checked in this priority
// order (extension/field/style as exact sets first) so a segment that is
// ALSO shaped like a 4-letter ISO-15924 script code — e.g. the "Text"
// extension — classifies as the extension, not a script. A segment
// matching none of these AND not shaped like a script code is silently
// ignored (G9, deferred lint — mirrors the pre-existing whole-role
// typo-ignoring behavior this replaces).
var fontExtensions = map[string]bool{
	"text": true, "dialog": true, "questions": true,
	"vocabulary": true, "models": true, "parallel": true,
}

var fontFields = map[string]bool{
	"source": true, "transcription": true, "translation": true, "grammar": true,
	"phrase": true, "question": true, "answer": true, "content": true,
	"main": true, "secondary": true, "tag": true, "header": true,
}

var fontStyles = map[string]bool{"strong": true, "emphasis": true}

// fontBaseRoleWords are the 6 legacy base-role names: SPECS §3 reserves
// these as "zero-qualifier only" — a role name is the legacy base role
// ONLY when it has exactly one segment matching this set (checked before
// the general per-segment classification below), e.g. "Font Body".
var fontBaseRoleWords = map[string]bool{
	"body": true, "header": true, "transcription": true,
	"translation": true, "strong": true, "emphasis": true,
}

// scriptSegmentRe matches a bare ISO-15924 code shape (4 alphabetic
// characters): SPECS §3 treats Script as the open/unbounded vocabulary, so
// anything not classified as an extension/field/style is presumed a script
// only if it has this shape; anything else is an unrecognized segment.
var scriptSegmentRe = regexp.MustCompile(`^[A-Za-z]{4}$`)

// slotKey joins the non-empty parts (in the order given) with a single
// space, producing the canonical FontTable key regardless of how many
// axes are populated. Both classifyFontFamily (parse time) and
// FontTable.resolve (query time) use this so a declared family's key and a
// query's candidate key always agree for the same (script,ext,field,style).
func slotKey(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " ")
}

// classifyFontFamily parses a "Font <segments...>" family name (case-
// insensitive) per SPECS §3's grammar and returns its canonical slot key —
// segments are re-ordered Script,Extension,Field,Style regardless of the
// author's own order ("order-tolerant" parsing; canonical order is a
// readability convention only). ok is false when name isn't a "Font ..."
// role name, or every segment is unrecognized (nothing to classify).
func classifyFontFamily(name string) (key string, ok bool) {
	fields := strings.Fields(name)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "font") {
		return "", false
	}
	segments := fields[1:]
	if len(segments) == 0 {
		return "", false
	}
	// A single segment matching a legacy base-role name IS that base role
	// (SPECS §3: reserved, zero-qualifier only) — checked before the
	// general per-segment loop so e.g. "Font Body" (4 letters, would also
	// match scriptSegmentRe) is never misread as a script-only role.
	if len(segments) == 1 && fontBaseRoleWords[strings.ToLower(segments[0])] {
		return strings.ToLower(segments[0]), true
	}

	var script, ext, field, style string
	for _, seg := range segments {
		s := strings.ToLower(seg)
		switch {
		case fontExtensions[s]:
			ext = s
		case fontFields[s]:
			field = s
		case fontStyles[s]:
			style = s
		case scriptSegmentRe.MatchString(seg):
			script = s
		default:
			// Unrecognized segment: ignored (G9), matches legacy behavior.
		}
	}
	key = slotKey(script, ext, field, style)
	if key == "" {
		return "", false
	}
	return key, true
}

// FontTable holds every parsed font.css @font-face family, keyed by its
// classified SPECS §3 slot, including the 6 legacy base roles (a strict,
// backward-compatible subset stored under their bare lowercased name —
// ASR-1). Zero value is a valid, empty table (every Lookup falls through to
// recommendedRoleFont).
type FontTable struct {
	slots map[string]string
}

// BaseRole returns the legacy base role's declared family (e.g. "body",
// "header"), or "" if font.css didn't declare it — used by
// assembleTypstDocument for the unchanged font-body/font-header/... args
// (SPECS §8.3: "base roles as today, plus the qualified families").
func (t FontTable) BaseRole(name string) string {
	if t.slots == nil {
		return ""
	}
	return t.slots[strings.ToLower(name)]
}

// Slots returns the full parsed slot map (canonical key -> family), for
// serializing into the generated .typ document (assembleTypstDocument) and
// for test inspection. The returned map must not be mutated by the caller.
func (t FontTable) Slots() map[string]string {
	return t.slots
}

// baseRoleForField implements SPECS §4's "BaseRole(F) map": Source/Content/
// Main/Question/Answer/Phrase -> Body; Transcription -> Transcription;
// Translation/Secondary/Grammar -> Translation; Tag/Header -> Header. When
// asTranslation is true (a block's as=translation, mirroring
// {start-text as=translation}), the primary-text fields resolve Translation
// instead of Body.
func baseRoleForField(field string, asTranslation bool) string {
	switch strings.ToLower(field) {
	case "source", "content", "main", "question", "answer", "phrase":
		if asTranslation {
			return "translation"
		}
		return "body"
	case "transcription":
		return "transcription"
	case "translation", "secondary", "grammar":
		return "translation"
	case "tag", "header":
		return "header"
	default:
		return "body"
	}
}

// Lookup implements SPECS §4's regular (non-styled) resolution chain --
// field -> extension -> script -> base-role -- returning the Typst font
// stack (the first declared, most-specific family, then the recommended
// fallback for the resolved base role; mirrors the pre-existing
// roleFontPrefix shape). style selects the Strong/Emphasis sub-axis
// ("strong"/"emphasis"; "" = regular). An empty script SKIPS the
// script-qualified levels entirely (SPECS §6 footnote: no automatic
// book-Script inheritance for EPUB/PDF, G1 deferred) and resolves via the
// base role directly -- this is also how Major-2's fixed-script fields
// (transcription/translation/grammar) are realized: the CALLER passes
// their own fixed script ("latn" / ""), never the block's foreign script.
func (t FontTable) Lookup(script, ext, field, style string) []string {
	return t.resolve(script, ext, field, style, false)
}

// LookupTranslation is Lookup's as=translation variant (SPECS §4 footnote):
// the qualified levels (1-3) still use field's own name unchanged, but the
// base-role fallback (level 4) resolves primary-text fields (source/
// content/main/question/answer/phrase) to Translation instead of Body,
// mirroring {start-text as=translation} (book.typ's textblock role branch).
func (t FontTable) LookupTranslation(script, ext, field, style string) []string {
	return t.resolve(script, ext, field, style, true)
}

func (t FontTable) resolve(script, ext, field, style string, asTranslation bool) []string {
	script = strings.ToLower(strings.TrimSpace(script))
	ext = strings.ToLower(strings.TrimSpace(ext))
	field = strings.ToLower(strings.TrimSpace(field))
	style = strings.ToLower(strings.TrimSpace(style))
	baseRole := baseRoleForField(field, asTranslation)

	if style != "" {
		// Style sub-axis (SPECS §4): try the regular levels with style
		// appended, then the pure base style role ("Font Strong"/"Font
		// Emphasis"); fall through to the regular chain below if none is
		// declared (book.typ's per-block/book-level gate then decides
		// whether the regular font is an acceptable substitute, §8.4).
		var candidates []string
		if script != "" {
			candidates = append(candidates,
				slotKey(script, ext, field, style),
				slotKey(script, ext, style),
				slotKey(script, style),
			)
		}
		candidates = append(candidates, style)
		for _, c := range candidates {
			if fam := t.slots[c]; fam != "" {
				return []string{fam, recommendedRoleFont[style]}
			}
		}
	}

	var candidates []string
	if script != "" {
		candidates = append(candidates,
			slotKey(script, ext, field),
			slotKey(script, ext),
			slotKey(script),
		)
	}
	for _, c := range candidates {
		if fam := t.slots[c]; fam != "" {
			return []string{fam, recommendedRoleFont[baseRole]}
		}
	}
	if fam := t.slots[baseRole]; fam != "" {
		return []string{fam, recommendedRoleFont[baseRole]}
	}
	if fb := recommendedRoleFont[baseRole]; fb != "" {
		return []string{fb}
	}
	return nil
}

// parseFontRoles reads the project's font.css and returns the FontTable
// every declared "Font ..." @font-face classifies into (SPECS §8.2). The
// name is kept from the pre-generalization API (only its return type
// changed, fontRoles -> FontTable) per SPECS §8.2's literal contract.
func parseFontRoles(stylesheetPaths []string) FontTable {
	table := FontTable{slots: map[string]string{}}
	var cssPath string
	for _, p := range stylesheetPaths {
		if strings.EqualFold(filepath.Base(p), "font.css") {
			cssPath = p
			break
		}
	}
	if cssPath == "" {
		return table
	}
	data, err := os.ReadFile(cssPath)
	if err != nil {
		return table
	}
	for _, block := range fontFaceRe.FindAllStringSubmatch(string(data), -1) {
		fam := fontFamilyRe.FindStringSubmatch(block[1])
		loc := fontLocalRe.FindStringSubmatch(block[1])
		if fam == nil || loc == nil {
			continue
		}
		key, ok := classifyFontFamily(strings.TrimSpace(fam[1]))
		if !ok {
			continue
		}
		table.slots[key] = strings.TrimSpace(loc[1])
	}
	return table
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
