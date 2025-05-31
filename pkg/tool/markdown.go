package tool

import (
	"bytes"
	"io"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// var MDParser = newMarkdownParser()
// var HtmlRenderer = newHtmlRenderer()

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
	p := newMarkdownParser()
	doc := p.Parse(md)

	// var buf bytes.Buffer
	// ast.Print(&buf, doc)
	// fmt.Print(buf.String())

	renderer := newHtmlRenderer()

	h := markdown.Render(doc, renderer)
	h = bytes.ReplaceAll(h, []byte("<hr>"), []byte("<hr />"))

	// fmt.Printf("%s", h)

	return h, nil
}

func parserHook(data []byte) (ast.Node, []byte, int) {
	if node, d, n := ParseVocabulary(data); node != nil {
		return node, d, n
	}
	if node, d, n := ParseDialog(data); node != nil {
		return node, d, n
	}
	if node, d, n := ParseParallel(data); node != nil {
		return node, d, n
	}
	return nil, nil, 0
}

func renderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {

	// Vocabulary
	if leafNode, ok := node.(*Vocabulary); ok {
		RenderVocabulary(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*VocabularyItem); ok {
		RenderVocabularyItem(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*VocabularyPhrase); ok {
		RenderVocabularyPhrase(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*VocabularyGrammar); ok {
		RenderVocabularyGrammar(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*VocabularyTranscription); ok {
		RenderVocabularyTranscription(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*VocabularyTranslation); ok {
		RenderVocabularyTranslation(w, leafNode, entering)
		return ast.GoToNext, true
	}

	// Dialog
	if leafNode, ok := node.(*Dialog); ok {
		RenderDialog(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*DialogItem); ok {
		RenderDialogItem(w, leafNode, entering)
		return ast.GoToNext, true
	}

	// Parallel
	if leafNode, ok := node.(*Parallel); ok {
		RenderParallel(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*ParallelBlock); ok {
		RenderParallelBlock(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*ParallelFirst); ok {
		RenderParallelFirst(w, leafNode, entering)
		return ast.GoToNext, true
	}
	if leafNode, ok := node.(*ParallelLast); ok {
		RenderParallelLast(w, leafNode, entering)
		return ast.GoToNext, true
	}

	return ast.GoToNext, false
}

func newMarkdownParser() *parser.Parser {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	p.Opts.ParserHook = parserHook
	return p
}

func newHtmlRenderer() *html.Renderer {
	opts := html.RendererOptions{
		Flags:          html.CommonFlags | html.HrefTargetBlank,
		RenderNodeHook: renderHook,
	}
	return html.NewRenderer(opts)
}
