package ebook

import (
	"log"

	"github.com/go-shiori/go-epub"
)

func buildEPub(projectFile string) (string, error) {
	// cd C:\jdp\src\local\ebook-example
	//ebook-cli build-project -f .\ebook.yml

	proj, err := readProject(projectFile)
	if err != nil {
		return "", err
	}
	// fmt.Printf("%#v", proj)

	// Create a new EPUB
	book, err := epub.NewEpub(proj.Title)
	if err != nil {
		return "", err
	}

	// Set the author
	book.SetAuthor(proj.Author)

	// // Add a section
	// section1Body := `<h1>Section 1</h1>
	// <p>This is a paragraph.</p>`
	// _, err = book.AddSection(section1Body, "Section 1", "", "")
	// if err != nil {
	// 	log.Println(err)
	// }

	// Write the EPUB
	err = book.Write(proj.Filename)
	if err != nil {
		log.Fatal(err)
	}

	return proj.Filename, err
}
