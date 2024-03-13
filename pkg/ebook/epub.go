package ebook

import (
	"log"
	"path/filepath"

	"github.com/go-shiori/go-epub"
)

func buildEPub(projectFile string) (string, error) {
	// cd C:\jdp\src\local\ebook-example
	//ebook-cli build-project -f .\ebook.yml

	project, err := readProject(projectFile)
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
		_, basename := filepath.Split(val)
		body := "Test"
		title := "Test"
		_, err := book.AddSection(body, title, basename, "")
		if err != nil {
			return "", err
		}
	}

	// // Add a section
	// section1Body := `<h1>Section 1</h1>
	// <p>This is a paragraph.</p>`
	// _, err = book.AddSection(section1Body, "Section 1", "", "")
	// if err != nil {
	// 	log.Println(err)
	// }

	// Write the EPUB
	err = book.Write(project.Filename)
	if err != nil {
		log.Fatal(err)
	}

	return project.Filename, err
}
