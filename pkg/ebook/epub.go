package ebook

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/dpurge/cli-tools/pkg/tool/markdown"
	"github.com/go-shiori/go-epub"
)

// epubExporter implements Exporter, producing the project's EPUB file with
// go-epub. Its steps are unchanged from the pre-refactor buildEPub; the
// section/chapter loop now consumes the shared WalkTexts (exporter.go) and
// language/direction now comes from the shared languageInfo (exporter.go)
// instead of the former project-local setLanguage.
type epubExporter struct{}

func (epubExporter) Export(project *EBookProject) (string, error) {
	book, err := epub.NewEpub(project.Title)
	if err != nil {
		return "", err
	}

	book.SetIdentifier(project.Identifier)
	book.SetAuthor(project.Author)
	book.SetDescription(project.Description)

	lang, dir := languageInfo(project.Language, project.Script)
	book.SetLang(lang)
	book.SetPpd(dir)

	stylesheets, err := addStylesheets(book, project.Stylesheet)
	if err != nil {
		return "", err
	}

	_, err = addFonts(book, project.Font)
	if err != nil {
		return "", err
	}

	_, err = addImages(book, project.Image)
	if err != nil {
		return "", err
	}

	if project.Cover != "" {
		_, err = setCover(book, project.Cover, stylesheets)
		if err != nil {
			return "", err
		}
	}

	_, err = addTexts(book, project.Text, stylesheets)
	if err != nil {
		return "", err
	}

	outfile := baseOutputName(project.Filename) + ".epub"
	err = book.Write(outfile)
	if err != nil {
		return "", err
	}

	return outfile, nil
}

func setCover(book *epub.Epub, cover string, stylesheets EBookStyles) (string, error) {
	var err error
	coverPath, _ := book.AddImage(cover, filepath.Base(cover))
	err = book.SetCover(coverPath, stylesheets.Cover)
	if err != nil {
		return "", err
	}
	return coverPath, nil
}

const epubLayoutCSS = `
.models-group,
.vocabulary-group,
.questions-group {
  display: table;
  width: 100%;
  border-collapse: separate;
  border-spacing: 0 0.35em;
}
.models-item.paired,
.vocabulary-item,
.questions-item {
  display: table-row;
}
.models-col1,
.models-col2,
.vocabulary-col1,
.vocabulary-col2,
.questions-col1,
.questions-col2 {
  display: table-cell;
  width: 50%;
  vertical-align: top;
}
.models-col1,
.vocabulary-col1,
.questions-col1 {
  padding-right: 0.7em;
}
.models-col2,
.vocabulary-col2,
.questions-col2 {
  padding-left: 0.7em;
}
.models-phrase,
.vocabulary-phrase {
  font-weight: bold;
}
.block-header,
.block-note,
.section-title,
.section-authors,
.section-year {
  text-align: center;
}
`

func addStylesheets(book *epub.Epub, stylesheets EBookStyles) (EBookStyles, error) {
	var styles EBookStyles

	layoutStyle, err := book.AddCSS("data:text/css,"+url.PathEscape(epubLayoutCSS), "ebook-layout.css")
	if err != nil {
		return styles, err
	}
	styles.Section = layoutStyle
	styles.Chapter = layoutStyle

	for i, val := range stylesheets.Common {
		style, err := book.AddCSS(stylesheets.Common[i], filepath.Base(val))
		if err != nil {
			return styles, err
		}
		styles.Common = append(styles.Common, style)
	}

	if stylesheets.Cover != "" {
		style, err := book.AddCSS(stylesheets.Cover, filepath.Base(stylesheets.Cover))
		if err != nil {
			return styles, err
		}
		styles.Cover = style
	}

	if stylesheets.Section != "" {
		style, err := addCSSWithLayout(book, stylesheets.Section, "section")
		if err != nil {
			return styles, err
		}
		styles.Section = style
	}

	if stylesheets.Chapter != "" {
		style, err := addCSSWithLayout(book, stylesheets.Chapter, "chapter")
		if err != nil {
			return styles, err
		}
		styles.Chapter = style
	}

	return styles, nil
}

func addCSSWithLayout(book *epub.Epub, source, prefix string) (string, error) {
	b, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	css := epubLayoutCSS + "\n" + string(b)
	return book.AddCSS("data:text/css,"+url.PathEscape(css), prefix+"-ebook-layout.css")
}

func addFonts(book *epub.Epub, fontfiles []string) ([]string, error) {
	var fonts = make([]string, 0, len(fontfiles))
	for _, val := range fontfiles {
		_, basename := filepath.Split(val)
		font, err := book.AddFont(val, basename)
		if err != nil {
			return nil, err
		}
		fonts = append(fonts, font)
	}
	return fonts, nil
}

func addImages(book *epub.Epub, imagefiles []string) ([]string, error) {
	var images = make([]string, 0, len(imagefiles))
	for _, val := range imagefiles {
		_, basename := filepath.Split(val)
		image, err := book.AddImage(val, basename)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

// addTexts walks project.Text via the shared WalkTexts (exporter.go),
// tracking the most recently added section's internal filename so each
// chapter is nested under its own section with AddSubSection. WalkTexts
// preserves the exact GLOBAL/continuous section/chapter counters the
// pre-refactor loop used, so the generated internal
// "section%04d.xhtml"/"chapter%04d.xhtml" filenames are unchanged.
func addTexts(book *epub.Epub, textfiles [][]string, styles EBookStyles) ([]string, error) {
	items := WalkTexts(textfiles)
	texts := make([]string, 0, len(items))
	var currentSection string

	for _, item := range items {
		switch item.Kind {
		case SectionItem:
			section, err := addSection(book, item.File, styles.Section, item.SectionIdx)
			if err != nil {
				return nil, err
			}
			currentSection = section
			texts = append(texts, section)
		case ChapterItem:
			chapter, err := addChapter(book, currentSection, item.File, styles.Chapter, item.ChapterIdx)
			if err != nil {
				return nil, err
			}
			texts = append(texts, chapter)
		}
	}

	return texts, nil
}

func addSection(book *epub.Epub, fileName string, stylesheet string, id int) (string, error) {
	body, err := markdown.FileToHTML(fileName)
	if err != nil {
		return "", err
	}

	title, err := tool.GetHtmlTitle(body)
	if err != nil {
		return "", err
	}

	internalFile, err := book.AddSection(body, title, fmt.Sprintf("section%04d.xhtml", id), stylesheet)
	if err != nil {
		return "", err
	}

	return internalFile, nil
}

func addChapter(book *epub.Epub, section string, fileName string, stylesheet string, id int) (string, error) {
	body, err := markdown.FileToHTML(fileName)
	if err != nil {
		return "", err
	}

	title, err := tool.GetHtmlTitle(body)
	if err != nil {
		return "", err
	}

	internalFile, err := book.AddSubSection(section, body, title, fmt.Sprintf("chapter%04d.xhtml", id), stylesheet)
	if err != nil {
		return "", err
	}

	return internalFile, nil
}
