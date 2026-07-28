package markdown

import (
	"io"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Vocabulary/Dialog/Parallel have no markdown child nodes (parser.go
// returns parser.NoChildren for all three), so each render func writes its
// entire wrapper on the "entering" call and does nothing on the matching
// "exiting" call.

// renderVocabulary emits the byte-identical `<div class="vocabulary">`
// wrapper. Spans are written raw (no HTML-escaping), matching the ported
// gomarkdown renderer.
func renderVocabulary(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Vocabulary)

	io.WriteString(w, "<div class=\"vocabulary\">\n")
	for _, item := range n.Items {
		io.WriteString(w, "<div class=\"vocabulary-item\">\n")
		if item.Phrase != "" {
			io.WriteString(w, "<span class=\"vocabulary-phrase\">")
			io.WriteString(w, item.Phrase)
			io.WriteString(w, "</span>\n")
		}
		if item.Grammar != "" {
			io.WriteString(w, "<span class=\"vocabulary-grammar\">")
			io.WriteString(w, item.Grammar)
			io.WriteString(w, "</span>\n")
		}
		if item.Transcription != "" {
			io.WriteString(w, "<span class=\"vocabulary-transcription\">")
			io.WriteString(w, item.Transcription)
			io.WriteString(w, "</span>\n")
		}
		if item.Translation != "" {
			io.WriteString(w, "<span class=\"vocabulary-translation\">")
			io.WriteString(w, item.Translation)
			io.WriteString(w, "</span>\n")
		}
		io.WriteString(w, "</div>\n")
	}
	io.WriteString(w, "</div>\n")

	return gast.WalkContinue, nil
}

// renderDialog emits the byte-identical `<div class="dialog">` wrapper. If
// parsing recorded a bad-indentation error (D3), rendering stops
// immediately and returns it, which surfaces out of renderer.Render ->
// goldmark.Markdown.Convert -> ToHTML/FileToHTML instead of the ported
// gomarkdown code's log.Fatal.
func renderDialog(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Dialog)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	io.WriteString(w, "<div class=\"dialog\">\n")
	for _, item := range n.Items {
		content, err := ToHTML([]byte(item.Content))
		if err != nil {
			return gast.WalkStop, err
		}
		io.WriteString(w, "<div class=\"dialog-item\">\n<div class=\"dialog-header\">")
		io.WriteString(w, item.Header)
		io.WriteString(w, "</div>\n<div class=\"dialog-content\">")
		w.Write(content)
		io.WriteString(w, "</div>\n</div>\n")
	}
	io.WriteString(w, "</div>\n")

	return gast.WalkContinue, nil
}

// renderParallel emits the byte-identical `<div class="parallel">` wrapper.
func renderParallel(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Parallel)

	io.WriteString(w, "<div class=\"parallel\">\n")
	for _, row := range n.Rows {
		io.WriteString(w, "<div class=\"parallel-row\">\n")
		if row.MainRaw != "" {
			content, err := ToHTML([]byte(row.MainRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, "<div class=\"parallel-cell main\">\n")
			w.Write(content)
			io.WriteString(w, "\n</div>\n")
		}
		if row.SecondaryRaw != "" {
			content, err := ToHTML([]byte(row.SecondaryRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, "<div class=\"parallel-cell secondary\">\n")
			w.Write(content)
			io.WriteString(w, "\n</div>\n")
		}
		io.WriteString(w, "</div>\n")
	}
	io.WriteString(w, "</div>\n")

	return gast.WalkContinue, nil
}

// vocabularyRenderer, dialogRenderer and parallelRenderer are thin
// renderer.NodeRenderer adapters that register the render funcs above.

type vocabularyRenderer struct{}

func (r *vocabularyRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindVocabulary, renderVocabulary)
}

type dialogRenderer struct{}

func (r *dialogRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindDialog, renderDialog)
}

type parallelRenderer struct{}

func (r *parallelRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindParallel, renderParallel)
}
