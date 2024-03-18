package ebook

import (
	"fmt"
	"log"
	"path/filepath"

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

	book.SetAuthor(project.Author)

	_, err = addStylesheets(book, project.Stylesheet)
	if err != nil {
		return "", err
	}
	// fmt.Println(stylesheets)

	_, err = addFonts(book, project.Font)
	if err != nil {
		return "", err
	}

	_, err = addImages(book, project.Image)
	if err != nil {
		return "", err
	}

	for _, val := range project.Text {
		// _, basename := filepath.Split(val)
		// body := "Test"
		// title := "Test"
		// _, err := book.AddSection(body, title, basename, "")
		// if err != nil {
		// 	return "", err
		// }
		fmt.Println(val)
	}

	// // Add a section
	// section1Body := `<h1>Section 1</h1>
	// <p>This is a paragraph in section 1.</p>`
	// _, err = book.AddSection(section1Body, "Section 1", "", "")
	// if err != nil {
	// 	log.Println(err)
	// }

	// Add a section. The CSS path is optional
	section1Body := `<h1>Section 1</h1><p>This is a paragraph.</p>`
	section1Path, err := book.AddSection(section1Body, "Section 1", "firstsection.xhtml", "")
	if err != nil {
		log.Println(err)
	}

	// Link to the first section
	section2Body := fmt.Sprintf(`<h1>Section 2</h1><a href="%s">Link to section 1</a>`, section1Path)
	// The title and filename are also optional
	section2Path, err := book.AddSubSection(section1Path, section2Body, "Section 2", "secondsection.xhtml", "")
	if err != nil {
		log.Println(err)
	}

	fmt.Println(section1Path)
	fmt.Println(section2Path)

	// Write the EPUB
	err = book.Write(project.Filename)
	if err != nil {
		log.Fatal(err)
	}

	return project.Filename, err
}

func addStylesheets(book *epub.Epub, stylesheets []string) ([]string, error) {
	var styles = make([]string, 0, len(stylesheets))
	for _, val := range stylesheets {
		_, basename := filepath.Split(val)
		style, err := book.AddCSS(val, basename)
		if err != nil {
			return nil, err
		}
		styles = append(styles, style)
	}
	return styles, nil
}

func addFonts(book *epub.Epub, fontfiles []string) ([]string, error) {
	var fonts = make([]string, 0, len(fontfiles))
	return fonts, nil
}

func addImages(book *epub.Epub, imagefiles []string) ([]string, error) {
	var images = make([]string, 0, len(imagefiles))
	return images, nil
}

// func addTexts(book *epub.Epub, textfiles []string) ([]string, error) {
// 	var texts = make([]string, 0, len(textfiles))
// 	return texts, nil
// }
