// Package markdown converts DPurge project markdown into HTML with goldmark.
//
// Besides CommonMark (plus tables, strikethrough, autolinks, definition
// lists and typographer substitutions), it understands six project-specific
// block extensions, each delimited by start/end markers that must appear on
// their own lines:
//
//	{start-vocabulary [lang=… script=…]} ... {end-vocabulary}
//	{start-dialog     [lang=… script=…]} ... {end-dialog}
//	{start-parallel   [lang=… script=…]} ... {end-parallel}
//	{start-models     [lang=… script=…]} ... {end-models}
//	{start-questions  [lang=… script=…]} ... {end-questions}
//	{start-text as=… [lang=… script=… system=…]} ... {end-text}
//
// Parsing (parser.go) captures raw text/structure into nodes (ast.go);
// rendering (renderer.go) emits HTML, recursively invoking ToHTML to
// render dialog/parallel/text cell content. See interlinear.go for a
// seventh, inactive block type.
package markdown

import (
	"bytes"
	"os"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// md is the single converter shared by top-level documents and the
// recursive rendering of dialog/parallel cell content. Reuse across those
// recursive calls is safe because rendering only ever recurses after the
// outer document's parse phase has fully completed (parse-then-render, not
// interleaved).
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Linkify,
		extension.DefinitionList,
		extension.Typographer,
		vocabularyExtender,
		dialogExtender,
		parallelExtender,
		modelsExtender,
		questionsExtender,
		textExtender,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(
			util.Prioritized(&linkTargetTransformer{}, 100),
		),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
	),
)

// normalizeNewlines converts CRLF (and lone CR) to LF. The custom block
// parsers (dialog/vocabulary/parallel) split on "\n", so without this a
// CRLF-encoded source leaves a trailing "\r" on every line and dialog
// headers fail to match.
func normalizeNewlines(source []byte) []byte {
	source = bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(source, []byte("\r"), []byte("\n"))
}

// ToHTML converts markdown source into HTML.
func ToHTML(source []byte) ([]byte, error) {
	source = normalizeNewlines(source)
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FileToHTML reads filename and converts its content into HTML.
func FileToHTML(filename string) (string, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	body, err := ToHTML(source)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
