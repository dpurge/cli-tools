package markdown

import (
	"fmt"
	"io"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Vocabulary/Dialog/Parallel have no markdown child nodes (parser.go
// returns parser.NoChildren for all three), so each render func writes its
// entire wrapper on the "entering" call and does nothing on the matching
// "exiting" call.

// scriptClass returns the SPECS §7.1 "s-<script>" class token, sourced
// directly from the block's own n.Script (with a leading space, ready to
// append to a class attribute) — e.g. "arab" -> " s-arab". Empty script
// returns "" (no token): a single-language block/book then applies only
// its base-role fonts, unaffected by this hook. This is the ONLY new EPUB
// hook `dir` can't provide on its own (`dir` cannot distinguish arab from
// hebr, both rtl), letting component CSS target `.s-<script> .field { ... }`
// for a two-language book (SPECS §4/§7.2).
func scriptClass(script string) string {
	if script == "" {
		return ""
	}
	return " s-" + script
}

// asClass returns the SPECS §7.1 "as-<value>" class token for a block that
// carries the shared as= attribute (text/dialog/questions), e.g.
// "translation" -> " as-translation". Empty or "source" (the default role)
// returns "" (no token): the block's default Body-role CSS then applies
// unchanged. WITHOUT this hook, EPUB has no way to tell that a dialog/
// questions block is as=translation (unlike text, whose wrapper class
// already changes name per as=), so PDF's as=translation Body->Translation
// swap (SPECS §4.1) silently failed to mirror into EPUB (ASR-4 divergence,
// code-review finding #4).
func asClass(as string) string {
	if as == "" || as == "source" {
		return ""
	}
	return " as-" + as
}

// renderVocabulary emits the byte-identical `<div class="vocabulary">`
// wrapper. Spans are written raw (no HTML-escaping), matching the ported
// gomarkdown renderer.
func renderVocabulary(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Vocabulary)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyHTML("V"))
	io.WriteString(w, "<div class=\"vocabulary")
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, "\" dir=\"")
	io.WriteString(w, dir)
	io.WriteString(w, "\">\n")
	for _, item := range n.Items {
		// ItemHeader: emit <hN> with no id (ASR-6) and raw text (ASR-4).
		if item.Kind == ItemHeader {
			fmt.Fprintf(w, "<h%d>%s</h%d>\n", item.Level, item.Text, item.Level)
			continue
		}
		// ItemData: existing vocabulary-item emission (no notes in vocabulary, D1).
		io.WriteString(w, "<div class=\"vocabulary-item\">\n")
		if item.Phrase != "" {
			io.WriteString(w, "<span class=\"vocabulary-phrase\">")
			io.WriteString(w, item.Phrase)
			io.WriteString(w, "</span>\n")
		}
		if item.Grammar != "" {
			io.WriteString(w, "<span class=\"vocabulary-grammar\" dir=\"ltr\">")
			io.WriteString(w, item.Grammar)
			io.WriteString(w, "</span>\n")
		}
		if item.Transcription != "" {
			io.WriteString(w, "<span class=\"vocabulary-transcription\" dir=\"ltr\">")
			io.WriteString(w, item.Transcription)
			io.WriteString(w, "</span>\n")
		}
		if item.Translation != "" {
			io.WriteString(w, "<span class=\"vocabulary-translation\" dir=\"ltr\">")
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

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyHTML("D"))
	io.WriteString(w, "<div class=\"dialog")
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, asClass(n.As))
	io.WriteString(w, "\" dir=\"")
	io.WriteString(w, dir)
	io.WriteString(w, "\">\n")
	for _, item := range n.Items {
		// ItemHeader: raw <hN>, no id (ASR-6/ASR-4).
		if item.Kind == ItemHeader {
			fmt.Fprintf(w, "<h%d>%s</h%d>\n", item.Level, item.Text, item.Level)
			continue
		}
		// ItemNote: centered block-note paragraph (SPECS §6).
		if item.Kind == ItemNote {
			io.WriteString(w, "<p class=\"block-note\">")
			io.WriteString(w, item.Text)
			io.WriteString(w, "</p>\n")
			continue
		}
		// ItemData: existing dialog-item emission.
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

// renderParallel emits the `<div class="parallel">` wrapper (SPECS §6).
// Per row:
//   - .parallel-cell.main always contains a .parallel-source inner wrapper
//     carrying dir="<blockDirection(n.Script)>" (source direction, from marker).
//     When TranscriptionRaw is non-empty, a .parallel-transcription wrapper
//     follows inside .main, with dir="ltr" pinned (matches vocabulary/models,
//     ASR-6/D5). The .main cell carries NO dir itself (ASR-8: the transcription
//     must not inherit the source's marker direction).
//   - .parallel-cell.secondary is emitted only when TranslationRaw is non-empty,
//     and carries NO dir attribute (translation inherits book direction, ASR-1).
func renderParallel(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Parallel)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	io.WriteString(w, badgeOnlyHTML("P"))
	io.WriteString(w, "<div class=\"parallel")
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, "\">\n")
	for _, row := range n.Rows {
		io.WriteString(w, "<div class=\"parallel-row\">\n")

		// Primary column: source always; transcription stacked below when present.
		sourceContent, err := ToHTML([]byte(row.SourceRaw))
		if err != nil {
			return gast.WalkStop, err
		}
		io.WriteString(w, "<div class=\"parallel-cell main\">\n")
		io.WriteString(w, "<div class=\"parallel-source\" dir=\"")
		io.WriteString(w, blockDirection(n.Script))
		io.WriteString(w, "\">\n")
		w.Write(sourceContent)
		io.WriteString(w, "\n</div>\n")
		if row.TranscriptionRaw != "" {
			transcriptionContent, err := ToHTML([]byte(row.TranscriptionRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, "<div class=\"parallel-transcription\" dir=\"ltr\">\n")
			w.Write(transcriptionContent)
			io.WriteString(w, "\n</div>\n")
		}
		io.WriteString(w, "</div>\n")

		// Secondary column: translation only when present; NO dir attribute (ASR-1).
		if row.TranslationRaw != "" {
			translationContent, err := ToHTML([]byte(row.TranslationRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, "<div class=\"parallel-cell secondary\">\n")
			w.Write(translationContent)
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
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyHTML("M"))
	io.WriteString(w, "<div class=\"models")
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, "\" dir=\"")
	io.WriteString(w, dir)
	io.WriteString(w, "\">\n")
	inGroup := false
	for _, item := range n.Items {
		// ItemHeader/ItemNote: flush any open models-group, then emit at full
		// block width outside the two-column group (SPECS §6 group-flush).
		if item.Kind == ItemHeader || item.Kind == ItemNote {
			if inGroup {
				io.WriteString(w, "</div>\n")
				inGroup = false
			}
			if item.Kind == ItemHeader {
				fmt.Fprintf(w, "<h%d>%s</h%d>\n", item.Level, item.Text, item.Level)
			} else {
				io.WriteString(w, "<p class=\"block-note\">")
				io.WriteString(w, item.Text)
				io.WriteString(w, "</p>\n")
			}
			continue
		}
		// ItemData: existing models-item emission.
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
			io.WriteString(w, "<span class=\"models-transcription\" dir=\"ltr\">")
			io.WriteString(w, item.Transcription)
			io.WriteString(w, "</span>\n")
		}
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "<div class=\"models-col2\">\n")
		switch {
		case item.Translation != "":
			io.WriteString(w, "<span class=\"models-translation\" dir=\"ltr\">")
			io.WriteString(w, item.Translation)
			io.WriteString(w, "</span>\n")
		case item.Transcription != "":
			io.WriteString(w, "<span class=\"models-transcription\" dir=\"ltr\">")
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
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyHTML("Q"))
	io.WriteString(w, "<div class=\"questions")
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, asClass(n.As))
	io.WriteString(w, "\" dir=\"")
	io.WriteString(w, dir)
	io.WriteString(w, "\">\n")
	inGroup := false
	for _, item := range n.Items {
		// ItemHeader/ItemNote: flush any open questions-group, then emit at full
		// block width outside the two-column group (SPECS §6 group-flush).
		if item.Kind == ItemHeader || item.Kind == ItemNote {
			if inGroup {
				io.WriteString(w, "</div>\n")
				inGroup = false
			}
			if item.Kind == ItemHeader {
				fmt.Fprintf(w, "<h%d>%s</h%d>\n", item.Level, item.Text, item.Level)
			} else {
				io.WriteString(w, "<p class=\"block-note\">")
				io.WriteString(w, item.Text)
				io.WriteString(w, "</p>\n")
			}
			continue
		}
		// ItemData: existing questions-item emission.
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

// renderTextblock emits `<div class="<cls>" dir="<dir>">` for the Text block
// (SPECS §7.1, §9.1, M3). CSS class mapping: as=source → "text",
// transcription/translation/grammar → their own name (OI-8). Direction rule
// (D9): as=transcription pinned ltr; source/translation/grammar derive
// direction from the block's own script. The Raw inner markdown is recursed
// through ToHTML. No inline CSS is emitted — centering and table styling
// live in the M5 CSS bundle, consuming the emitted class + dir.
func renderTextblock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Text)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}
	as := n.As
	if as == "" {
		as = "source"
	}
	cls := as
	if as == "source" {
		cls = "text"
	}
	dir := blockDirection(n.Script)
	if as == "transcription" {
		dir = "ltr"
	}
	// FR-2/FR-3: always emit the "T" badge as a standalone element before the
	// wrapper div — never injected into a heading. This keeps the badge out of
	// any heading element and at the same fixed size as V/D/M/Q/P badges.
	var body string
	if n.Raw != "" {
		content, err := ToHTML([]byte(n.Raw))
		if err != nil {
			return gast.WalkStop, err
		}
		body = string(content)
	}
	io.WriteString(w, badgeOnlyHTML("T"))
	io.WriteString(w, "<div class=\"")
	io.WriteString(w, cls)
	io.WriteString(w, scriptClass(n.Script))
	io.WriteString(w, asClass(as))
	io.WriteString(w, "\" dir=\"")
	io.WriteString(w, dir)
	io.WriteString(w, "\">\n")
	io.WriteString(w, body)
	io.WriteString(w, "</div>\n")
	return gast.WalkContinue, nil
}

// vocabularyRenderer, dialogRenderer, parallelRenderer, modelsRenderer,
// questionsRenderer and textHTMLRenderer are thin renderer.NodeRenderer
// adapters that register the render funcs above.

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

// textHTMLRenderer registers the KindText HTML render func. It is wired into
// the shared goldmark instance via textExtension (extension.go), satisfying
// ASR-1 for the HTML path.
type textHTMLRenderer struct{}

func (r *textHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindText, renderTextblock)
}
