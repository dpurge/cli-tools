package ebook

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
// or "rtl"). It reproduces, verbatim, every case of the pre-refactor
// setLanguage (epub.go) — including its one quirk: "heb" (the real ISO
// 639-3 code for Hebrew, used by the heb sample project) is NOT one of the
// handled cases and therefore falls through to the default "en", exactly as
// it did before this refactor. That quirk is pre-existing and out of scope
// to fix here (SPECS AC7 requires reproducing setLanguage, not correcting
// it).
//
// Both exporters (epubExporter, typstExporter) call this single function,
// so the EPUB and PDF outputs of the same project always agree on
// language/direction.
func languageInfo(language, script string) (lang, dir string) {
	switch language {
	case "ajp", "apc", "arb":
		lang = "ar"
	case "bul":
		lang = "bg"
	case "ces":
		lang = "cs"
	case "cmn":
		if script == "hant" {
			lang = "zh-Hant"
		} else {
			lang = "zh-Hans"
		}
	case "dan":
		lang = "da"
	case "deu":
		lang = "de"
	case "ell":
		lang = "el"
	case "fas":
		lang = "fa"
	case "fra":
		lang = "fr"
	case "grc":
		lang = "el"
	case "hin":
		lang = "hi"
	case "ind":
		lang = "id"
	case "ita":
		lang = "it"
	case "kaz":
		lang = "kk"
	case "lat":
		lang = "la"
	case "lit":
		lang = "lt"
	case "mon":
		lang = "mn"
	case "nld":
		lang = "nl"
	case "ron":
		lang = "ro"
	case "spa":
		lang = "es"
	case "srp":
		lang = "sr"
	case "tgk":
		lang = "tg"
	case "tha":
		lang = "th"
	case "tur":
		lang = "tr"
	case "uig":
		lang = "ug"
	case "ukr":
		lang = "uk"
	case "uzb":
		lang = "uz"
	case "vie":
		lang = "vi"
	case "yid":
		lang = "yi"
	case "yue":
		if script == "hans" {
			lang = "zh-Hans"
		} else {
			lang = "zh-Hant"
		}
	default:
		lang = "en"
	}

	switch script {
	case "arab":
		dir = "rtl"
	case "hebr":
		dir = "rtl"
	default:
		dir = "ltr"
	}

	return lang, dir
}
