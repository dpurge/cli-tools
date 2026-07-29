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

// renderModels emits the `<div class="models">` wrapper (SPECS decision:
// like vocabulary minus grammar/notes). Per item:
//   - phrase only (no transcription, no translation) renders as a plain,
//     non-tabular line ("models-item", no col1/col2 wrapper) — it is never
//     given an empty second column.
//   - every other combination renders as a "models-item paired" row with
//     "models-col1"/"models-col2" wrapper divs (so CSS can lay them out as
//     top-aligned table cells): col1 is the phrase, plus the transcription
//     stacked BELOW it (separated by a "<br/>") only when BOTH
//     transcription and translation are present; col2 is the translation
//     if present, else the transcription, else empty.
//
// Consecutive paired items are grouped into a single "models-group" wrapper
// (mirrors renderQuestions' "questions-group"), so CSS can lay them out as
// ONE table and their col1/col2 widths align across items — matching the
// Typst side, which already grids every item together (book.typ's
// `models(..items)`). A phrase-only item flushes the current group, exactly
// like a question-only item flushes "questions-group".
//
// Spans are written raw (no HTML-escaping), matching renderVocabulary.
func renderModels(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Models)

	io.WriteString(w, "<div class=\"models\">\n")
	inGroup := false
	for _, item := range n.Items {
		if item.Transcription == "" && item.Translation == "" {
			if inGroup {
				io.WriteString(w, "</div>\n")
				inGroup = false
			}
			io.WriteString(w, "<div class=\"models-item\">\n")
			if item.Phrase != "" {
				io.WriteString(w, "<span class=\"models-phrase\">")
				io.WriteString(w, item.Phrase)
				io.WriteString(w, "</span>\n")
			}
			io.WriteString(w, "</div>\n")
			continue
		}

		if !inGroup {
			io.WriteString(w, "<div class=\"models-group\">\n")
			inGroup = true
		}
		io.WriteString(w, "<div class=\"models-item paired\">\n")
		io.WriteString(w, "<div class=\"models-col1\">\n")
		if item.Phrase != "" {
			io.WriteString(w, "<span class=\"models-phrase\">")
			io.WriteString(w, item.Phrase)
			io.WriteString(w, "</span>\n")
		}
		if item.Transcription != "" && item.Translation != "" {
			io.WriteString(w, "<br/>\n")
			io.WriteString(w, "<span class=\"models-transcription\">")
			io.WriteString(w, item.Transcription)
			io.WriteString(w, "</span>\n")
		}
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "<div class=\"models-col2\">\n")
		switch {
		case item.Translation != "":
			io.WriteString(w, "<span class=\"models-translation\">")
			io.WriteString(w, item.Translation)
			io.WriteString(w, "</span>\n")
		case item.Transcription != "":
			io.WriteString(w, "<span class=\"models-transcription\">")
			io.WriteString(w, item.Transcription)
			io.WriteString(w, "</span>\n")
		}
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "</div>\n")
	}
	if inGroup {
		io.WriteString(w, "</div>\n")
	}
	io.WriteString(w, "</div>\n")

	return gast.WalkContinue, nil
}

// renderQuestions emits the `<div class="questions">` wrapper. A
// question-only item ("questions-item", no answer) renders as a plain
// paragraph-style line, in normal body-font style; a question+answer item
// renders as a "questions-item paired" row. Consecutive question+answer
// items are grouped into a single "questions-group" wrapper (one shared
// aligned two-column block per maximal run), flushed whenever a
// question-only item or the end of the block is reached — so a mixed block
// may contain several independent aligned runs, never one grid spanning a
// question-only line with an empty answer column.
func renderQuestions(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Questions)

	io.WriteString(w, "<div class=\"questions\">\n")
	inGroup := false
	for _, item := range n.Items {
		if item.Answer == "" {
			if inGroup {
				io.WriteString(w, "</div>\n")
				inGroup = false
			}
			io.WriteString(w, "<div class=\"questions-item\">\n")
			io.WriteString(w, "<span class=\"questions-question\">")
			io.WriteString(w, item.Question)
			io.WriteString(w, "</span>\n")
			io.WriteString(w, "</div>\n")
			continue
		}

		if !inGroup {
			io.WriteString(w, "<div class=\"questions-group\">\n")
			inGroup = true
		}
		io.WriteString(w, "<div class=\"questions-item paired\">\n")
		io.WriteString(w, "<div class=\"questions-col1\">\n")
		io.WriteString(w, "<span class=\"questions-question\">")
		io.WriteString(w, item.Question)
		io.WriteString(w, "</span>\n")
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "<div class=\"questions-col2\">\n")
		io.WriteString(w, "<span class=\"questions-answer\">")
		io.WriteString(w, item.Answer)
		io.WriteString(w, "</span>\n")
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "</div>\n")
	}
	if inGroup {
		io.WriteString(w, "</div>\n")
	}
	io.WriteString(w, "</div>\n")

	return gast.WalkContinue, nil
}

// vocabularyRenderer, dialogRenderer, parallelRenderer, modelsRenderer and
// questionsRenderer are thin renderer.NodeRenderer adapters that register
// the render funcs above.

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

type modelsRenderer struct{}

func (r *modelsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindModels, renderModels)
}

type questionsRenderer struct{}

func (r *questionsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindQuestions, renderQuestions)
}
