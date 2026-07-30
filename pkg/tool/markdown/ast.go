package markdown

import (
	gast "github.com/yuin/goldmark/ast"
)

// Node kinds for the custom block types. KindInterlinear is defined for
// completeness but never registered with the converter (see
// interlinear.go). KindText is the highest ordinal ever registered by this
// package; ALL THREE renderers (HTML, Typst, MDX) MUST register a
// NodeRendererFunc for EVERY kind through KindText, or a document
// containing a block whose kind exceeds the registered maximum panics
// (index out of range) — see the identical warning on typstNodeRenderer/
// mdxNodeRenderer and SPECS ASR-1.
var (
	KindVocabulary  = gast.NewNodeKind("Vocabulary")
	KindDialog      = gast.NewNodeKind("Dialog")
	KindParallel    = gast.NewNodeKind("Parallel")
	KindInterlinear = gast.NewNodeKind("Interlinear")
	KindModels      = gast.NewNodeKind("Models")
	KindQuestions   = gast.NewNodeKind("Questions")
	KindText        = gast.NewNodeKind("Text") // MUST be last; highest ordinal (ASR-1)
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
// attached. Lang and Script are populated from marker attributes (M1);
// direction/font wiring uses them in M2.
type Vocabulary struct {
	gast.BaseBlock

	Lang, Script string
	Err          error
	Items        []VocabularyItem
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
// or when the marker attributes are malformed; the renderer surfaces it
// as a real error out of ToHTML/FileToHTML. Lang and Script are populated
// from marker attributes (M1); direction/font wiring uses them in M2.
type Dialog struct {
	gast.BaseBlock

	Lang, Script string
	Items        []DialogItem
	Err          error
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
// block. Lang and Script are populated from marker attributes (M1);
// direction/font wiring uses them in M2. Err is set when marker attributes
// are malformed.
type Parallel struct {
	gast.BaseBlock

	Lang, Script string
	Err          error
	Rows         []ParallelRow
}

// Kind implements ast.Node.
func (n *Parallel) Kind() gast.NodeKind { return KindParallel }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Parallel) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Parallel) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// ModelsItem is one parsed `{start-models}` line: a phrase plus its
// optional transcription and translation. Like VocabularyItem, minus
// Grammar (Cycle 1 scope: no grammar tag, no notes).
type ModelsItem struct {
	Phrase        string
	Transcription string
	Translation   string
}

// Models is the block node for a `{start-models}` ... `{end-models}`
// block. Lang and Script are populated from marker attributes (M1);
// direction/font wiring uses them in M2. Err is set when marker attributes
// are malformed.
type Models struct {
	gast.BaseBlock

	Lang, Script string
	Err          error
	Items        []ModelsItem
}

// Kind implements ast.Node.
func (n *Models) Kind() gast.NodeKind { return KindModels }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Models) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Models) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// QuestionItem is one parsed `{start-questions}` line: a question plus its
// optional answer. A line with no answer is a question-only line, rendered
// in normal paragraph (body-font) style rather than as a two-column row.
type QuestionItem struct {
	Question string
	Answer   string
}

// Questions is the block node for a `{start-questions}` ... `{end-questions}`
// block. Lang and Script are populated from marker attributes (M1);
// direction/font wiring uses them in M2. Err is set when marker attributes
// are malformed.
type Questions struct {
	gast.BaseBlock

	Lang, Script string
	Err          error
	Items        []QuestionItem
}

// Kind implements ast.Node.
func (n *Questions) Kind() gast.NodeKind { return KindQuestions }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Questions) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Questions) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// Text is the block node for a `{start-text as=...}` ... `{end-text}` block
// (SPECS §3.2, D8). Unlike vocabulary/models/questions, Text is a raw-markdown
// block: its inner content is arbitrary markdown captured verbatim into Raw and
// recursed at render time (ToTypst/ToHTML/ToMDX), NOT parsed into items.
// As defaults to "source" when omitted on the marker. System is parsed and
// stored for forward-compat (OI-9) but ignored by the PDF/EPUB renderers in M1.
// Direction/font wiring is added in M2/M3. Err is set when marker attributes
// are malformed (surfaced at render time, mirroring Dialog.Err).
type Text struct {
	gast.BaseBlock

	As, Lang, Script, System string
	Raw                       string
	Err                       error
}

// Kind implements ast.Node.
func (n *Text) Kind() gast.NodeKind { return KindText }

// IsRaw marks the block as raw so goldmark does NOT run its generic inline
// pass over the captured Lines() — the body is arbitrary markdown that is
// recursed at render time, not goldmark-parsed into inline child nodes.
// Mirrors Dialog/Parallel's IsRaw convention.
func (n *Text) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Text) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
