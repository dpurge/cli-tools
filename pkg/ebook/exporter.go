package ebook

import (
	"path/filepath"
	"strings"

	"github.com/dpurge/cli-tools/pkg/catalog"
)

// baseOutputName strips the extension from a project Filename to obtain a
// base path suitable for appending any output extension. If the filename ends
// in ".epub" that suffix is stripped; for any other extension the generic
// filepath.Ext suffix is stripped. This is the single shared derivation used
// by all exporters — EPUB, PDF/Typst, MDX, and vocabulary CSV.
func baseOutputName(filename string) string {
	if strings.HasSuffix(filename, ".epub") {
		return strings.TrimSuffix(filename, ".epub")
	}
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// Exporter builds one output artifact (EPUB, PDF, ...) from an already
// loaded EBookProject. Each output format (epubExporter, typstExporter)
// implements this interface so build-cmd.go can read the project once and
// dispatch it to every requested format.
type Exporter interface {
	Export(project *EBookProject) (outfile string, err error)
}

// ItemKind identifies whether a ProjectItem is a section or a chapter file.
type ItemKind int

const (
	SectionItem ItemKind = iota
	ChapterItem
)

// ProjectItem is one file from an EBookProject's Text tree, flattened into
// document order by WalkTexts.
type ProjectItem struct {
	File       string
	Kind       ItemKind
	SectionIdx int
	ChapterIdx int
}

// WalkTexts flattens project.Text ([][]string; each inner slice is one
// section, its first element the section file and the remainder its
// chapter files) into an ordered slice of ProjectItem.
//
// CRITICAL: SectionIdx/ChapterIdx are running counters, and ChapterIdx is
// GLOBAL/CONTINUOUS across section boundaries (it is never reset when a new
// section starts) — this reproduces the pre-refactor addTexts loop
// (epub.go), whose chapterId/sectionId feed directly into the
// "section%04d.xhtml"/"chapter%04d.xhtml" internal EPUB filenames. Resetting
// ChapterIdx per section would silently renumber every chapter file after
// the first section and change the generated EPUB. Every real project in
// this codebase has exactly one section, so only a synthetic multi-section
// fixture (see typst_export_test.go) can catch a regression here.
func WalkTexts(text [][]string) []ProjectItem {
	items := make([]ProjectItem, 0, len(text))
	sectionIdx := 0
	chapterIdx := 0
	for _, group := range text {
		if len(group) == 0 {
			continue
		}

		sectionIdx++
		items = append(items, ProjectItem{
			File:       group[0],
			Kind:       SectionItem,
			SectionIdx: sectionIdx,
		})

		for _, file := range group[1:] {
			chapterIdx++
			items = append(items, ProjectItem{
				File:       file,
				Kind:       ChapterItem,
				SectionIdx: sectionIdx,
				ChapterIdx: chapterIdx,
			})
		}
	}
	return items
}

// languageInfo maps an EBookProject's ISO 639-3 Language code and ISO 15924
// Script code to a BCP-47-ish language tag and a paragraph direction ("ltr"
// or "rtl"), by consulting pkg/catalog — the single declared source of
// truth for languages/scripts (previously this was its own independent,
// hand-maintained switch statement, one of several scattered, inconsistent
// closed sets; see pkg/catalog's package doc for the full history).
//
// This function's signature and every call site are unchanged; only the
// body now defers to catalog.HTMLLang/catalog.Direction. One behavior
// change is intentional and disclosed: "heb" (the real ISO 639-3 code for
// Hebrew, used by the heb sample project) was never a case in the old
// switch and silently fell through to "en" — a documented, deliberately
// preserved quirk before pkg/catalog existed. Now that the catalog is the
// single source of truth (no more reproduce-the-old-switch constraint),
// heb correctly resolves to "he".
//
// Both exporters (epubExporter, typstExporter) call this single function,
// so the EPUB and PDF outputs of the same project always agree on
// language/direction.
func languageInfo(language, script string) (lang, dir string) {
	return catalog.HTMLLang(language, script), catalog.Direction(script)
}
