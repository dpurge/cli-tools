package tool

import (
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func MarkdownFileToHTML(filename string) (string, error) {
	md, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	html, err := MarkdownToHTML(md)
	if err != nil {
		return "", err
	}
	return string(html), nil
}

func MarkdownToHTML(md []byte) ([]byte, error) {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	h := markdown.Render(doc, renderer)

	return h, nil
}
