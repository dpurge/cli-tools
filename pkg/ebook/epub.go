package ebook

import (
	"fmt"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/go-shiori/go-epub"
)

func buildEPub(projectfile string) (string, error) {
	project, err := readProject(projectfile)
	if err != nil {
		return "", err
	}
	// fmt.Printf("%#v", project)

	book, err := epub.NewEpub(project.Title)
	if err != nil {
		return "", err
	}

	book.SetIdentifier(project.Identifier)
	book.SetAuthor(project.Author)
	book.SetDescription(project.Description)

	err = setLanguage(book, project.Language, project.Script)
	if err != nil {
		return "", err
	}

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
		_, err = setCover(book, project.Cover, stylesheets.Cover)
		if err != nil {
			return "", err
		}
	}

	_, err = addTexts(book, project.Text, stylesheets)
	if err != nil {
		return "", err
	}

	err = book.Write(project.Filename)
	if err != nil {
		return "", err
	}

	return project.Filename, nil
}

func setLanguage(book *epub.Epub, language string, script string) error {
	book.SetLang("en")
	book.SetPpd("ltr")

	return nil
}

func setCover(book *epub.Epub, cover string, style string) (string, error) {
	var err error
	coverPath, _ := book.AddImage(cover, filepath.Base(cover))
	if style == "" {
		err = book.SetCover(coverPath, "")
	} else {
		err = book.SetCover(coverPath, style)
	}

	if err != nil {
		return "", err
	}
	return coverPath, nil
}

func addStylesheets(book *epub.Epub, stylesheets EBookStyles) (EBookStyles, error) {
	var styles EBookStyles

	if stylesheets.Section != "" {
		style, err := book.AddCSS(stylesheets.Section, "section.css")
		if err != nil {
			return styles, err
		}
		styles.Section = style
	}

	if stylesheets.Chapter != "" {
		style, err := book.AddCSS(stylesheets.Chapter, "chapter.css")
		if err != nil {
			return styles, err
		}
		styles.Chapter = style
	}

	return styles, nil
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

func addTexts(book *epub.Epub, textfiles [][]string, styles EBookStyles) ([]string, error) {
	var texts = make([]string, 0, len(textfiles))
	var sectionId = 0
	var chapterId = 0
	for _, items := range textfiles {
		if len(items) > 0 {

			sectionId++
			section, err := addSection(book, items[0], styles.Section, sectionId)
			if err != nil {
				return nil, err
			}
			texts = append(texts, section)

			for _, filename := range items[1:] {
				chapterId++
				chapter, err := addChapter(book, section, filename, styles.Chapter, chapterId)
				if err != nil {
					return nil, err
				}
				texts = append(texts, chapter)
			}
		}
	}

	return texts, nil
}

func addSection(book *epub.Epub, fileName string, stylesheet string, id int) (string, error) {
	body, err := tool.MarkdownFileToHTML(fileName)
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
	body, err := tool.MarkdownFileToHTML(fileName)
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
