package markdown

import (
	gast "github.com/yuin/goldmark/ast"
)

// Node kinds for the custom block types. KindInterlinear is defined for
// completeness but never registered with the converter (see
// interlinear.go).
var (
	KindVocabulary  = gast.NewNodeKind("Vocabulary")
	KindDialog      = gast.NewNodeKind("Dialog")
	KindParallel    = gast.NewNodeKind("Parallel")
	KindInterlinear = gast.NewNodeKind("Interlinear")
)

// VocabularyItem is one parsed `{start-vocabulary}` line: a phrase plus its
// optional grammar tag, transcription and translation.
type VocabularyItem struct {
	Phrase        string
	Grammar       string
	Transcription string
	Translation   string
}

// Vocabulary is the block node for a `{start-vocabulary}` ...
// `{end-vocabulary}` block. Parsing (parser.go) fills Items; rendering
// (renderer.go) only loops over Items — no markdown child nodes are ever
// attached.
type Vocabulary struct {
	gast.BaseBlock

	Items []VocabularyItem
}

// Kind implements ast.Node.
func (n *Vocabulary) Kind() gast.NodeKind { return KindVocabulary }

// IsRaw marks the block as raw so goldmark does NOT run its generic inline
// pass over the captured Lines() and attach them as child text nodes. The
// block's content is parsed structurally into Items instead; without this,
// goldmark would append the raw source after the rendered wrapper. Mirrors
// goldmark's own CodeBlock/HTMLBlock convention.
func (n *Vocabulary) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Vocabulary) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// DialogItem is one turn of a `{start-dialog}` block. Content is the raw,
// dedented markdown of that turn, captured verbatim during parsing; it is
// converted to HTML recursively (via the package converter) only at render
// time, never during parsing.
type DialogItem struct {
	Header  string
	Content string
}

// Dialog is the block node for a `{start-dialog}` ... `{end-dialog}`
// block. Err is set when a content line has invalid indentation (D3)
// instead of the original gomarkdown port's log.Fatal; the renderer
// surfaces it as a real error out of ToHTML/FileToHTML.
type Dialog struct {
	gast.BaseBlock

	Items []DialogItem
	Err   error
}

// Kind implements ast.Node.
func (n *Dialog) Kind() gast.NodeKind { return KindDialog }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Dialog) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Dialog) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// ParallelRow is one row of a `{start-parallel}` block: the raw main-cell
// markdown, and — if the row has a `---` separator — the raw secondary-cell
// markdown. Both are converted to HTML recursively at render time.
type ParallelRow struct {
	MainRaw      string
	SecondaryRaw string
}

// Parallel is the block node for a `{start-parallel}` ... `{end-parallel}`
// block.
type Parallel struct {
	gast.BaseBlock

	Rows []ParallelRow
}

// Kind implements ast.Node.
func (n *Parallel) Kind() gast.NodeKind { return KindParallel }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Parallel) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Parallel) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
