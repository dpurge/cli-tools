package ebook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// mdxExporter implements Exporter, producing a DIRECTORY of
// phraseforge-web-consumable source (SPECS Increment 3, §7): one
// "<chapter-basename>.mdx" per ChapterItem (frontmatter + markdown.ToMDX
// body) and one "_category_.json" per SectionItem (Docusaurus
// generated-index), reusing the same WalkTexts seam epubExporter/
// typstExporter already consume (exporter.go).
//
// Unlike those two exporters, mdxExporter does NOT call languageInfo: MDX/
// phraseforge use the project's raw ISO-639-3 Language + lowercase
// ISO-15924 Script verbatim (SPECS §7.1) — languageInfo's BCP-47 mapping
// ("tur"->"tr") is EPUB/Typst-only and would be wrong here.
type mdxExporter struct{}

func (mdxExporter) Export(project *EBookProject) (string, error) {
	dir := derivedMdxDir(project.Filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	items := WalkTexts(project.Text)
	multiSection := countSections(items) > 1

	// currentDir tracks the most recently written SectionItem's output
	// directory, mirroring epub.go's addTexts currentSection tracking: every
	// ChapterItem that follows a SectionItem in WalkTexts' document order
	// belongs under that section's directory (dir itself in the flat/
	// single-section layout, or a per-section subfolder when multiSection,
	// SPECS §7.2/Decision D7).
	currentDir := dir
	for _, item := range items {
		switch item.Kind {
		case SectionItem:
			sectionDir, err := writeCategoryJSON(dir, item, project, multiSection)
			if err != nil {
				return "", err
			}
			currentDir = sectionDir
		case ChapterItem:
			if err := writeChapterMDX(currentDir, item.File, project); err != nil {
				return "", err
			}
		}
	}

	return dir, nil
}

// countSections returns the number of SectionItem entries in items, used to
// decide whether the flat (single-section, the norm) or per-section
// subfolder (multi-section, Decision D7) layout applies.
func countSections(items []ProjectItem) int {
	n := 0
	for _, item := range items {
		if item.Kind == SectionItem {
			n++
		}
	}
	return n
}

// derivedMdxDir derives the "-mdx" output directory from the project's EPUB
// filename (mirrors derivedTypstPaths, typst.go): strip ".epub" (else
// filepath.Ext) and append "-mdx", e.g.
// ".../turkish-notes-private.epub" -> ".../turkish-notes-private-mdx".
func derivedMdxDir(epubFilename string) string {
	base := epubFilename
	if strings.HasSuffix(base, ".epub") {
		base = strings.TrimSuffix(base, ".epub")
	} else {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return base + "-mdx"
}

// mdxCategoryLink is the "link" object of a Docusaurus "_category_.json"
// generated-index (SPECS §7.3). Field order is fixed (type, title,
// description) by declaration order — encoding/json.Marshal emits struct
// fields in that order, giving the byte-stable output ASR-3 requires
// without resorting to a map (whose key order is unspecified).
type mdxCategoryLink struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// mdxCategory is the full "_category_.json" shape (SPECS §7.3): label,
// position, link — in that fixed order, and deliberately WITHOUT a "slug"
// key (Decision D6: the exporter does not know the phraseforge docs mount
// path, so Docusaurus is left to default the route from the folder
// location).
type mdxCategory struct {
	Label    string          `json:"label"`
	Position int             `json:"position"`
	Link     mdxCategoryLink `json:"link"`
}

// writeCategoryJSON reads item's section file, extracts its H1 via
// markdown.Title for both "label" and "link.title" (Decision D8: no
// separate short-label source exists), and writes "_category_.json" into
// item's target directory: rootDir itself for the single-section norm, or
// a per-section subfolder under rootDir when multiSection (Decision D7,
// guarding against a basename collision across sections — the subfolder
// name is prefixed with the zero-padded, globally unique SectionIdx). It
// returns that target directory so the caller can place the section's
// ChapterItems inside it.
func writeCategoryJSON(rootDir string, item ProjectItem, project *EBookProject, multiSection bool) (string, error) {
	src, err := os.ReadFile(item.File)
	if err != nil {
		return "", err
	}

	title, err := markdown.Title(src)
	if err != nil {
		return "", err
	}

	sectionDir := rootDir
	if multiSection {
		sectionDir = filepath.Join(rootDir, sectionSlug(item.SectionIdx, title, item.File))
	}
	if err := os.MkdirAll(sectionDir, 0o755); err != nil {
		return "", err
	}

	category := mdxCategory{
		Label:    title,
		Position: item.SectionIdx,
		Link: mdxCategoryLink{
			Type:  "generated-index",
			Title: title,
			// TrimSpace so a `description: |` YAML block scalar's trailing
			// newline doesn't leak into the JSON string (keeps this
			// consistent with the chapter frontmatter, which trims via
			// mdxYamlString).
			Description: strings.TrimSpace(project.Description),
		},
	}

	data, err := json.MarshalIndent(category, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')

	if err := os.WriteFile(filepath.Join(sectionDir, "_category_.json"), data, 0o644); err != nil {
		return "", err
	}

	return sectionDir, nil
}

// writeChapterMDX reads chapterFile once, derives its title (first H1 via
// markdown.Title, falling back to the file's basename when absent, SPECS
// §7.4/§9), converts its body with markdown.ToMDX using the project's RAW
// Language/Script (SPECS §7.1 - NOT languageInfo), and writes
// "<basename>.mdx" (frontmatter + body) into dir.
func writeChapterMDX(dir, chapterFile string, project *EBookProject) error {
	src, err := os.ReadFile(chapterFile)
	if err != nil {
		return err
	}

	title, err := markdown.Title(src)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(chapterFile), filepath.Ext(chapterFile))
	if title == "" {
		title = base
	}

	body, err := markdown.ToMDX(src, project.Language, project.Script)
	if err != nil {
		return err
	}

	var doc strings.Builder
	doc.WriteString("---\n")
	doc.WriteString("title: " + mdxYamlString(title) + "\n")
	doc.WriteString("description: " + mdxYamlString(project.Description) + "\n")
	doc.WriteString("---\n\n")
	doc.Write(body)

	return os.WriteFile(filepath.Join(dir, base+".mdx"), []byte(doc.String()), 0o644)
}

// mdxYamlString double-quotes s as a single-line YAML scalar for chapter
// frontmatter (SPECS §5.4, mirrors typstStringLiteral, typst.go:157-178,
// for the analogous string-context escaping need): "\\" and '"' are
// escaped; any newline/tab is collapsed to a single space and the result is
// then trimmed, so a multi-line YAML block-scalar project.Description
// (e.g. ebook.yml's "description: |") still produces one clean single-line
// frontmatter value instead of a value with a dangling trailing space.
func mdxYamlString(s string) string {
	var collapsed strings.Builder
	for _, r := range s {
		switch r {
		case '\n', '\t', '\r':
			collapsed.WriteByte(' ')
		default:
			collapsed.WriteRune(r)
		}
	}
	trimmed := strings.TrimSpace(collapsed.String())

	var b strings.Builder
	b.WriteByte('"')
	for _, r := range trimmed {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// sectionSlug derives a filesystem-safe, deterministic, COLLISION-FREE
// subfolder name for a SectionItem when a project has more than one
// section (SPECS §7.2/Decision D7): "<NN>-<slug>", where NN is the
// zero-padded SectionIdx (guarantees uniqueness on its own, even if two
// sections happen to share a title/basename) and slug is derived from the
// section's H1 title, falling back to the section file's basename and
// finally to the literal "section" when both are empty.
func sectionSlug(sectionIdx int, title, sectionFile string) string {
	slug := slugify(title)
	if slug == "" {
		slug = slugify(strings.TrimSuffix(filepath.Base(sectionFile), filepath.Ext(sectionFile)))
	}
	if slug == "" {
		slug = "section"
	}
	return fmt.Sprintf("%02d-%s", sectionIdx, slug)
}

// slugify lowercases s and collapses every run of characters outside
// [a-z0-9] into a single '-', trimming any leading/trailing '-'. It is a
// simple ASCII-only slugifier (non-ASCII letters, e.g. Polish diacritics in
// a real section title, become a separator like any other non [a-z0-9]
// rune) - acceptable here because the result is only ever a directory name
// (SPECS D7 is an extension for the multi-section case; no real project in
// this codebase has more than one section today), not user-facing prose.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
