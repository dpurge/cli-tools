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
		_, err = setCover(book, project.Cover, stylesheets)
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
	switch language {
	case "ajp":
		book.SetLang("ar")
	case "apc":
		book.SetLang("ar")
	case "arb":
		book.SetLang("ar")
	case "bul":
		book.SetLang("bg")
	case "ces":
		book.SetLang("cs")
	case "cmn":
		if script == "hant" {
			book.SetLang("zh-Hant")
		} else {
			book.SetLang("zh-Hans")
		}
	case "dan":
		book.SetLang("da")
	case "deu":
		book.SetLang("de")
	case "ell":
		book.SetLang("el")
	case "fas":
		book.SetLang("fa")
	case "fra":
		book.SetLang("fr")
	case "grc":
		book.SetLang("el")
	case "hin":
		book.SetLang("hi")
	case "ind":
		book.SetLang("id")
	case "ita":
		book.SetLang("it")
	case "kaz":
		book.SetLang("kk")
	case "lat":
		book.SetLang("la")
	case "lit":
		book.SetLang("lt")
	case "mon":
		book.SetLang("mn")
	case "nld":
		book.SetLang("nl")
	case "ron":
		book.SetLang("ro")
	case "spa":
		book.SetLang("es")
	case "srp":
		book.SetLang("sr")
	case "tgk":
		book.SetLang("tg")
	case "tha":
		book.SetLang("th")
	case "tur":
		book.SetLang("tr")
	case "uig":
		book.SetLang("ug")
	case "ukr":
		book.SetLang("uk")
	case "uzb":
		book.SetLang("uz")
	case "vie":
		book.SetLang("vi")
	case "yid":
		book.SetLang("yi")
	case "yue":
		if script == "hans" {
			book.SetLang("zh-Hans")
		} else {
			book.SetLang("zh-Hant")
		}
	default:
		book.SetLang("en")
	}

	switch script {
	case "arab":
		book.SetPpd("rtl")
	case "hebr":
		book.SetPpd("rtl")
	default:
		book.SetPpd("ltr")
	}

	return nil
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

func addStylesheets(book *epub.Epub, stylesheets EBookStyles) (EBookStyles, error) {
	var styles EBookStyles

	for i, val := range stylesheets.Common {
		// fmt.Println("Stylesheet: ", i, "=>", val)
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
		style, err := book.AddCSS(stylesheets.Section, filepath.Base(stylesheets.Section))
		if err != nil {
			return styles, err
		}
		styles.Section = style
	}

	if stylesheets.Chapter != "" {
		style, err := book.AddCSS(stylesheets.Chapter, filepath.Base(stylesheets.Chapter))
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
