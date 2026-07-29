package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// vocabularyExtension registers the vocabulary block parser and renderer.
type vocabularyExtension struct{}

func (e *vocabularyExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(newVocabularyParser(), 100),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&vocabularyRenderer{}, 100),
	))
}

// dialogExtension registers the dialog block parser and renderer.
type dialogExtension struct{}

func (e *dialogExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(newDialogParser(), 110),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&dialogRenderer{}, 110),
	))
}

// parallelExtension registers the parallel block parser and renderer.
type parallelExtension struct{}

func (e *parallelExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(newParallelParser(), 120),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&parallelRenderer{}, 120),
	))
}

// modelsExtension registers the models block parser and renderer.
type modelsExtension struct{}

func (e *modelsExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(newModelsParser(), 130),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&modelsRenderer{}, 130),
	))
}

// questionsExtension registers the questions block parser and renderer.
type questionsExtension struct{}

func (e *questionsExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(newQuestionsParser(), 140),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&questionsRenderer{}, 140),
	))
}

// Extenders wired into the shared converter (converter.go). Interlinear is
// deliberately excluded — it remains an inactive stub (interlinear.go).
var (
	vocabularyExtender goldmark.Extender = &vocabularyExtension{}
	dialogExtender     goldmark.Extender = &dialogExtension{}
	parallelExtender   goldmark.Extender = &parallelExtension{}
	modelsExtender     goldmark.Extender = &modelsExtension{}
	questionsExtender  goldmark.Extender = &questionsExtension{}
)
