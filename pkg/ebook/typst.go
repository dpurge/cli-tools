package ebook

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// bookTemplate is the self-contained Typst preamble (pkg/ebook/templates/
// book.typ, FROZEN by this batch — not modified here): it defines the
// `book`, `vocabulary`, `dialog`, and `parallel` top-level functions that
// the generated `.typ` document below relies on.
//
//go:embed templates/book.typ
var bookTemplate string

// typstExporter implements Exporter, producing the project's PDF file by
// assembling one self-contained Typst document (the embedded book.typ
// preamble + a `#show: book.with(...)` call + every text file converted
// with markdown.ToTypst, joined by weak pagebreaks) and subprocess-
// compiling it with the `typst` binary.
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

	document := assembleTypstDocument(project, lang, dir, cover, bodies)

	// The generated .typ is written unconditionally before compiling, and
	// is deliberately left in place (not a Go os.CreateTemp file) so it
	// remains available for debugging a compile failure (SPECS §9).
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
	// Surface Typst's own warnings (e.g. missing-font substitutions) even on
	// a successful compile — CombinedOutput captured them and they would
	// otherwise be silently discarded, hiding font/glyph issues from the user.
	if strings.TrimSpace(output) != "" {
		fmt.Fprint(os.Stderr, output)
	}

	return pdfPath, nil
}

// assembleTypstDocument builds the full `.typ` source: the embedded
// book.typ preamble, a `#show: book.with(...)` call mapping the
// EBookProject fields per SPECS §8.3 (cover passed through as a path
// string, matching book.typ's `if type(cover) == str` branch), and every
// chapter/section body joined with a weak pagebreak (Decision D4). cover is
// already a Typst-root-relative path (typstAssetPath) or "" for no cover.
func assembleTypstDocument(project *EBookProject, lang, dir, cover string, bodies []string) string {
	var doc strings.Builder

	doc.WriteString(bookTemplate)
	doc.WriteString("\n#show: book.with(\n")
	doc.WriteString("  title: " + typstStringLiteral(project.Title) + ",\n")
	// author is ALWAYS passed (never omitted, unlike description/cover
	// below): book.typ's own default is `author: none`, and its body does
	// `set document(title: title, author: author)` unconditionally.
	// Typst's built-in document() requires `author` to be a string or an
	// array — never `none` (empirically confirmed: `set document(author:
	// none)` is a compile error against Typst 0.15.1) — so leaving this
	// argument out for an author-less project would surface that error
	// from inside the frozen template on every such build. Passing "" is
	// the one value that satisfies BOTH that call and the template's other
	// use of the same parameter (`if author != none {...}` on the title
	// page, which also requires non-array content) without editing
	// book.typ.
	doc.WriteString("  author: " + typstStringLiteral(project.Author) + ",\n")
	if project.Description != "" {
		doc.WriteString("  description: " + typstStringLiteral(project.Description) + ",\n")
	}
	doc.WriteString("  lang: " + typstStringLiteral(lang) + ",\n")
	// dir is one of the two literal identifiers "ltr"/"rtl" (languageInfo,
	// exporter.go) and MUST be emitted unquoted: book.typ's `dir` parameter
	// is Typst's built-in `direction` type (the bare keywords ltr/rtl), not
	// a string.
	doc.WriteString("  dir: " + dir + ",\n")
	if cover != "" {
		doc.WriteString("  cover: " + typstStringLiteral(cover) + ",\n")
	}
	doc.WriteString(")\n\n")

	doc.WriteString(strings.Join(bodies, "\n#pagebreak(weak: true)\n\n"))
	doc.WriteString("\n")

	return doc.String()
}

// typstAssetPath converts project.Cover's absolute filesystem path (as
// resolved by readProject, project.go) into a Typst project-root-relative
// path: a leading "/" in a Typst path string means "relative to --root",
// NOT the OS filesystem root. Emitting the raw OS-absolute path verbatim
// (the SPECS §8.3 literal reading of "absolute path string") makes Typst
// concatenate root+absolute-path and fail with "file not found" —
// empirically discovered compiling the real tur sample, whose cover.svg
// sits next to its ebook.yml. rootDir is always the directory this
// exporter writes the .pdf/.typ into (derivedTypstPaths), which is also
// where readProject resolves every other project asset, so a path
// expressed relative to rootDir resolves correctly. Returns "" unchanged
// for an unset cover.
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

// typstStringLiteral quotes s as a Typst string literal. It mirrors the
// escaping rules of the markdown package's (unexported) escapeTypstString
// (SPECS §5.2: inside "..." only "\\" and '"' are metacharacters) rather
// than importing it, since these values (title/author/description/cover
// path) originate from the project YAML, not from parsed markdown content.
func typstStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// derivedTypstPaths derives the output PDF path and the generated .typ path
// from the project's EPUB filename (mirrors buildVocabulary's derivation of
// its .csv path from the .epub filename, vocabulary.go), placing both next
// to the EPUB (SPECS §8.4/D7): --root for the typst compile is that
// directory, so relative/absolute project assets (cover, fonts, markdown
// sources) already resolved under it by readProject continue to resolve.
func derivedTypstPaths(epubFilename string) (pdfPath, typPath string) {
	base := epubFilename
	if strings.HasSuffix(base, ".epub") {
		base = strings.TrimSuffix(base, ".epub")
	} else {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return base + ".pdf", base + ".typ"
}

// fontPathDirs returns the de-duplicated, order-preserving set of
// directories containing project.Font's font files, one `--font-path`
// argument per directory (SPECS A4: one directory per project.Font
// directory suffices; per-family selection is deferred, Decision D5).
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

// locateTypst finds the typst binary: first via config (Typst.typst,
// cfg/linux.yml), then falling back to the PATH (SPECS §8.5/Decision D8).
// config.GetToolPath os.Stats its configured value, so a bare "typst" (as
// shipped in cfg/linux.yml) fails that check and falls through to
// exec.LookPath — which is required for Docker/PATH-only installs, and is
// also why an entirely absent config file does not, by itself, make PDF
// export impossible (only config.ReadConfig's own pre-existing log.Fatal on
// a missing config file, config/main.go, does — SPECS AC6/Decision D9).
func locateTypst() (string, error) {
	if path, err := config.GetToolPath("Typst", "typst"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("typst"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("typst not found: set Typst.typst in config or install on PATH (https://typst.app)")
}

// runTypst runs the located typst binary with args, mirroring
// pkg/tool/command.go's RunCmd (exec.Command + CombinedOutput) but taking
// an already-resolved path instead of a config lookup, since locateTypst
// above already covers both the config and PATH cases RunCmd's
// config-only lookup does not.
func runTypst(path string, args ...string) (string, error) {
	cmd := exec.Command(path, args...)
	buf, err := cmd.CombinedOutput()
	return string(buf), err
}
