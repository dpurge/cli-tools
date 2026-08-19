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
	KindVocabulary     = gast.NewNodeKind("Vocabulary")
	KindDialog         = gast.NewNodeKind("Dialog")
	KindParallel       = gast.NewNodeKind("Parallel")
	KindInterlinear    = gast.NewNodeKind("Interlinear")
	KindModels         = gast.NewNodeKind("Models")
	KindQuestions      = gast.NewNodeKind("Questions")
	KindParallelDialog = gast.NewNodeKind("ParallelDialog")
	KindText           = gast.NewNodeKind("Text") // MUST be last; highest ordinal (ASR-1)
)

// ItemKind discriminates the kind of item within a structured block.
// The zero value is ItemData, so all pre-existing item struct literals that
// omit Kind remain ItemData, producing byte-identical output for
// header/note-free blocks (ASR-3).
type ItemKind uint8

const (
	ItemData   ItemKind = iota // default: existing data fields populated
	ItemHeader                 // Level (1-6) + Text carry the heading
	ItemNote                   // Text carries the note (parens stripped); never produced for VocabularyItem (D1)
)

// BlockAnnotation carries the discriminated-union fields added to every
// structured-block item type. Embedding it in an item struct promotes Kind,
// Level, and Text with zero values (ItemData, 0, "") that leave all
// pre-existing struct literals unchanged (ASR-3).
type BlockAnnotation struct {
	Kind  ItemKind
	Level int    // 1-6 for ItemHeader; 0 otherwise
	Text  string // heading/note text; "" for ItemData
}

// VocabularyItem is one parsed `{start-vocabulary}` line: a phrase plus its
// optional grammar tag, transcription and translation.
type VocabularyItem struct {
	BlockAnnotation
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
	BlockAnnotation
	Header  string
	Content string
}

// Dialog is the block node for a `{start-dialog}` ... `{end-dialog}`
// block. Err is set when a content line has invalid indentation (D3)
// or when the marker attributes are malformed; the renderer surfaces it
// as a real error out of ToHTML/FileToHTML. Lang and Script are populated
// from marker attributes (M1); direction/font wiring uses them in M2.
// As unifies the block on the shared as= attribute grammar (SPECS §5):
// accepted values are "source" (default) and "translation" only — dialog
// has no transcription/grammar concept, unlike {start-text}.
type Dialog struct {
	gast.BaseBlock

	Lang, Script, As string
	Items            []DialogItem
	Err              error
}

// Kind implements ast.Node.
func (n *Dialog) Kind() gast.NodeKind { return KindDialog }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *Dialog) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *Dialog) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// ParallelRow is one row of a {start-parallel} block. Fields are split on
// every lone "---" (up to 3, SPECS §3.2): the source-cell markdown, the
// optional translation-cell markdown, and the optional transcription markdown
// (stacked under the source in the primary column). All are converted
// recursively at render time.
type ParallelRow struct {
	SourceRaw        string // field 1 — marker lang/script (was: MainRaw)
	TranslationRaw   string // field 2 — book language      (was: SecondaryRaw)
	TranscriptionRaw string // field 3 — pinned Latin/LTR romanization (NEW)
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

// ParallelDialogItem is one field of a {start-parallel-dialog} row: either a
// dialog turn (Header+Content, same grammar as DialogItem) or a title
// (BlockAnnotation.Kind==ItemHeader). Exactly one item per present field —
// unlike Dialog, a field never holds a run of several turns.
type ParallelDialogItem struct {
	BlockAnnotation
	Header  string
	Content string
}

// ParallelDialogRow is one row of a {start-parallel-dialog} block, split on
// every lone "---" (up to 3, same field count as ParallelRow): Source and
// Translation are mandatory; Transcription is optional. Unlike
// ParallelRow's Transcription (raw markdown), all three fields here share
// the identical turn/heading grammar — Transcription differs only in which
// font it renders with (pinned Latin/LTR), not in how it is authored.
type ParallelDialogRow struct {
	Source           ParallelDialogItem
	Translation      ParallelDialogItem
	Transcription    ParallelDialogItem
	HasTranscription bool // Transcription field was present in the source (3rd "---" field supplied)
}

// ParallelDialog is the block node for a `{start-parallel-dialog}` ...
// `{end-parallel-dialog}` block: {start-parallel}'s row/field grid where
// each field carries a {start-dialog} turn instead of arbitrary markdown.
// Lang and Script are populated from marker attributes (M1); as= is
// rejected, mirroring Parallel (both columns' languages are fixed). Err is
// set when marker attributes or a row/field are malformed.
type ParallelDialog struct {
	gast.BaseBlock

	Lang, Script string
	Err          error
	Rows         []ParallelDialogRow
}

// Kind implements ast.Node.
func (n *ParallelDialog) Kind() gast.NodeKind { return KindParallelDialog }

// IsRaw marks the block as raw; see Vocabulary.IsRaw.
func (n *ParallelDialog) IsRaw() bool { return true }

// Dump implements ast.Node.
func (n *ParallelDialog) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// ModelsItem is one parsed `{start-models}` line: a phrase plus its
// optional transcription and translation. Like VocabularyItem, minus
// Grammar (Cycle 1 scope: no grammar tag, no notes).
type ModelsItem struct {
	BlockAnnotation
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
	BlockAnnotation
	Question string
	Answer   string
}

// Questions is the block node for a `{start-questions}` ... `{end-questions}`
// block. Lang and Script are populated from marker attributes (M1);
// direction/font wiring uses them in M2. Err is set when marker attributes
// are malformed. As unifies the block on the shared as= attribute grammar
// (SPECS §5): accepted values are "source" (default) and "translation" only.
type Questions struct {
	gast.BaseBlock

	Lang, Script, As string
	Err              error
	Items            []QuestionItem
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
	Raw                      string
	Err                      error
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
