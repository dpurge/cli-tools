package ebook

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/go-shiori/go-epub"
)

func buildEPub(projectfile string) (string, error) {
	// cd C:\jdp\src\local\ebook-example
	//ebook-cli build-project -f .\ebook.yml

	project, err := readProject(projectfile)
	if err != nil {
		return "", err
	}
	// fmt.Printf("%#v", proj)

	// Create a new EPUB
	book, err := epub.NewEpub(project.Title)
	if err != nil {
		return "", err
	}

	// Set the author
	book.SetAuthor(project.Author)

	for _, val := range project.Stylesheet {
		_, basename := filepath.Split(val)
		_, err := book.AddCSS(val, basename)
		if err != nil {
			return "", err
		}
	}

	// for i, val := range project.Font {
	// }

	// for i, val := range project.Image {
	// }

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
