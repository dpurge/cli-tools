package markdown

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// typstNodeRenderer registers a Typst NodeRendererFunc for every node kind
// in SPECS §4: the standard CommonMark+GFM subset the ebooks use, plus the
// 3 custom blocks. Kinds not registered here (RawHTML, HTMLBlock,
// LinkReferenceDefinition, and anything from an extension not enabled by
// md, converter.go) are deliberately left unregistered: the shared
// renderer's walk treats an unregistered-but-in-range kind as "emit
// nothing, still walk children" (renderer/renderer.go:164), which for
// RawHTML/HTMLBlock (no markdown children of their own) is exactly the
// desired "emit nothing" mirror of the HTML renderer's Unsafe=false
// omission (SPECS §4 last two rows), and for any genuinely unknown kind is
// the ASR-6 graceful-fallback behavior.
type typstNodeRenderer struct{}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *typstNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(gast.KindDocument, renderDocumentTypst)
	reg.Register(gast.KindHeading, renderHeadingTypst)
	reg.Register(gast.KindParagraph, renderParagraphTypst)
	reg.Register(gast.KindTextBlock, renderTextBlockTypst)
	reg.Register(gast.KindText, renderTextTypst)
	reg.Register(gast.KindString, renderStringTypst)
	reg.Register(gast.KindEmphasis, renderEmphasisTypst)
	reg.Register(gast.KindCodeSpan, renderCodeSpanTypst)
	reg.Register(gast.KindLink, renderLinkTypst)
	reg.Register(gast.KindAutoLink, renderAutoLinkTypst)
	reg.Register(gast.KindImage, renderImageTypst)
	reg.Register(gast.KindList, renderListTypst)
	reg.Register(gast.KindListItem, renderListItemTypst)
	reg.Register(gast.KindBlockquote, renderBlockquoteTypst)
	reg.Register(gast.KindFencedCodeBlock, renderFencedCodeBlockTypst)
	reg.Register(gast.KindCodeBlock, renderCodeBlockTypst)
	reg.Register(gast.KindThematicBreak, renderThematicBreakTypst)

	reg.Register(extast.KindTable, renderTableTypst)
	reg.Register(extast.KindTableHeader, renderTableHeaderTypst)
	reg.Register(extast.KindTableRow, renderTableRowTypst)
	reg.Register(extast.KindTableCell, renderTableCellTypst)
	reg.Register(extast.KindStrikethrough, renderStrikethroughTypst)
	reg.Register(extast.KindDefinitionList, renderDefinitionListTypst)
	reg.Register(extast.KindDefinitionTerm, renderDefinitionTermTypst)
	reg.Register(extast.KindDefinitionDescription, renderDefinitionDescriptionTypst)

	reg.Register(KindVocabulary, renderVocabularyTypst)
	reg.Register(KindDialog, renderDialogTypst)
	reg.Register(KindParallel, renderParallelTypst)
	reg.Register(KindModels, renderModelsTypst)
	reg.Register(KindQuestions, renderQuestionsTypst)
	// KindText MUST be registered last (highest ordinal, ASR-1 panic-gate).
	reg.Register(KindText, renderTextblockTypst)
}

// renderDocumentTypst: Document walks its children with no wrapper.
func renderDocumentTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

// renderHeadingTypst emits `=`x Level + " " + inline children + "\n\n".
func renderHeadingTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Heading)
	if entering {
		io.WriteString(w, strings.Repeat("=", n.Level))
		io.WriteString(w, " ")
	} else {
		io.WriteString(w, "\n\n")
	}
	return gast.WalkContinue, nil
}

// renderParagraphTypst emits inline children followed by a blank line.
func renderParagraphTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		io.WriteString(w, "\n\n")
	}
	return gast.WalkContinue, nil
}

// renderTextBlockTypst emits inline children followed by a single newline;
// used for tight list-item bodies.
func renderTextBlockTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		io.WriteString(w, "\n")
	}
	return gast.WalkContinue, nil
}

// renderTextTypst emits the Text node's escaped value, then a break
// marker: a hard break emits Typst's own hard-break shorthand (a backslash
// followed by a space, empirically verified against Typst 0.15.1 to force
// a line break independent of the source's own newline placement); a soft
// break emits a single space rather than a newline, because a Typst line
// that starts with "- "/"+ " (or an unescaped ordinal) is parsed as a list
// item even when the previous line ended with a soft wrap or a hard break
// (FR-6; empirically verified) — so soft breaks must never surface as a
// line-leading position in the emitted Typst source.
//
// The raw segment value is passed through unescapeMarkdownBackslash
// (typst_escape.go) BEFORE escapeTypstMarkup: goldmark's Text.Value keeps
// a source backslash-escape's backslash byte-for-byte (verified against
// the installed goldmark — see unescapeMarkdownBackslash's doc comment
// for the discrepancy vs SPECS §5.2's assumption), so skipping this step
// would double-escape (e.g. `1\. text` would keep its own backslash AND
// gain Typst's metachar-escape backslash for the period, printing a
// visible backslash in the rendered PDF instead of none).
func renderTextTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.Text)
	value := unescapeMarkdownBackslash(string(n.Value(source)))
	io.WriteString(w, escapeTypstMarkup(value))
	switch {
	case n.HardLineBreak():
		io.WriteString(w, "\\ \n")
	case n.SoftLineBreak():
		io.WriteString(w, " ")
	}
	return gast.WalkContinue, nil
}

// renderStringTypst maps a Typographer-emitted HTML entity to its Unicode
// codepoint (typographerEntities, typst_escape.go); any other String value
// is literal text and is markup-escaped like Text (SPECS §4/§5.2).
func renderStringTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.String)
	if u, ok := typographerEntities[string(n.Value)]; ok {
		io.WriteString(w, u)
	} else {
		io.WriteString(w, escapeTypstMarkup(string(n.Value)))
	}
	return gast.WalkContinue, nil
}

// renderEmphasisTypst uses Typst's function form (#emph[...]/#strong[...])
// rather than the `_.../*...*` markup shortcuts, avoiding delimiter
// boundary/whitespace edge cases (SPECS Decision D2). Level 2 is strong;
// anything else (i.e. Level 1) is emph, mirroring goldmark's own HTML
// renderer, which only special-cases Level==2 (renderer/html/html.go:566).
func renderEmphasisTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Emphasis)
	fn := "emph"
	if n.Level == 2 {
		fn = "strong"
	}
	if entering {
		io.WriteString(w, "#")
		io.WriteString(w, fn)
		io.WriteString(w, "[")
	} else {
		io.WriteString(w, "]")
	}
	return gast.WalkContinue, nil
}

// renderCodeSpanTypst reads the CodeSpan's child Text segments directly
// (mirroring renderer/html/html.go's renderCodeSpan, including its
// trailing-newline-becomes-space rule) rather than walking them through
// the normal Text renderer, because CodeSpan content is a Typst string
// literal (`#raw("...")`), not markup: it must be escaped with
// escapeTypstString, not escapeTypstMarkup. WalkSkipChildren prevents the
// walker from separately visiting (and re-rendering) those Text children.
func renderCodeSpanTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.CodeSpan)
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		segment := c.(*gast.Text).Segment
		value := segment.Value(source)
		if bytes.HasSuffix(value, []byte("\n")) {
			buf.Write(value[:len(value)-1])
			buf.WriteByte(' ')
		} else {
			buf.Write(value)
		}
	}
	io.WriteString(w, `#raw("`)
	io.WriteString(w, escapeTypstString(buf.String()))
	io.WriteString(w, `")`)
	return gast.WalkSkipChildren, nil
}

// renderLinkTypst emits `#link("dest")[` children `]`; Title is ignored
// (irrelevant in a PDF) and link target attributes (linktarget.go) are
// likewise irrelevant off the HTML/EPUB path (SPECS §4).
func renderLinkTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Link)
	if entering {
		io.WriteString(w, `#link("`)
		io.WriteString(w, escapeTypstString(string(n.Destination)))
		io.WriteString(w, `")[`)
	} else {
		io.WriteString(w, `]`)
	}
	return gast.WalkContinue, nil
}

// renderAutoLinkTypst mirrors renderer/html/html.go's renderAutoLink: an
// email autolink gets a "mailto:" prefix if it doesn't already have one.
// AutoLink carries its label/URL internally (ast/inline.go) rather than as
// child nodes, so everything is written on the entering call.
func renderAutoLinkTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.AutoLink)
	url := string(n.URL(source))
	if n.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(url), "mailto:") {
		url = "mailto:" + url
	}
	io.WriteString(w, `#link("`)
	io.WriteString(w, escapeTypstString(url))
	io.WriteString(w, `")[`)
	io.WriteString(w, escapeTypstMarkup(string(n.Label(source))))
	io.WriteString(w, `]`)
	return gast.WalkContinue, nil
}

// renderImageTypst emits `#image("dest")`; alt text is dropped (Decision
// D6, v1: minimal now). WalkSkipChildren mirrors the HTML renderer, which
// also never walks Image's children as ordinary content.
func renderImageTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.Image)
	io.WriteString(w, `#image("`)
	io.WriteString(w, escapeTypstString(string(n.Destination)))
	io.WriteString(w, `")`)
	return gast.WalkSkipChildren, nil
}

// renderListTypst emits the function form `#list(...)`/`#enum(...)`
// (Decision D2), with each ListItem contributing one comma-terminated
// positional argument (renderListItemTypst).
func renderListTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.List)
	fn := "list"
	if n.IsOrdered() {
		fn = "enum"
	}
	if entering {
		io.WriteString(w, "#")
		io.WriteString(w, fn)
		io.WriteString(w, "(\n")
	} else {
		io.WriteString(w, ")\n\n")
	}
	return gast.WalkContinue, nil
}

// renderListItemTypst wraps the item's rendered body as one positional
// argument to the parent #list/#enum call.
func renderListItemTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "[")
	} else {
		io.WriteString(w, "],\n")
	}
	return gast.WalkContinue, nil
}

// renderBlockquoteTypst emits `#quote(block: true)[` children `]`.
func renderBlockquoteTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "#quote(block: true)[")
	} else {
		io.WriteString(w, "]\n\n")
	}
	return gast.WalkContinue, nil
}

// renderFencedCodeBlockTypst emits `#raw(block: true, lang: "..", "..")`,
// omitting the `lang:` argument entirely when there is no info string
// (SPECS §4). FencedCodeBlock's body is raw (never inline-parsed), so
// n.Text is read directly and string-escaped; WalkSkipChildren is
// defensive (raw blocks carry no markdown children to walk).
func renderFencedCodeBlockTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.FencedCodeBlock)
	io.WriteString(w, "#raw(block: true, ")
	if lang := n.Language(source); len(lang) > 0 {
		io.WriteString(w, `lang: "`)
		io.WriteString(w, escapeTypstString(string(lang)))
		io.WriteString(w, `", `)
	}
	io.WriteString(w, `"`)
	io.WriteString(w, escapeTypstString(string(n.Text(source))))
	io.WriteString(w, "\")\n\n")
	return gast.WalkSkipChildren, nil
}

// renderCodeBlockTypst emits `#raw(block: true, "..")` for an indented
// code block (no info string, so never a `lang:` argument).
func renderCodeBlockTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.CodeBlock)
	io.WriteString(w, `#raw(block: true, "`)
	io.WriteString(w, escapeTypstString(string(n.Text(source))))
	io.WriteString(w, "\")\n\n")
	return gast.WalkSkipChildren, nil
}

// renderThematicBreakTypst emits `#line(length: 100%)` (SPECS §4, exact
// string including its single trailing newline).
func renderThematicBreakTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "#line(length: 100%)\n")
	return gast.WalkContinue, nil
}

// renderTableTypst emits `#table(columns: (1fr, 1fr, ...), align: (...), <cells>)`;
// fractional column tracks make every markdown table fill the available text width
// (required for grammar tables per D12; acceptable globally since tables appear only
// in prose/text/grammar blocks in this handbook). The header row and body rows are
// transparent wrappers (renderTableHeaderTypst/renderTableRowTypst) that contribute
// no markup of their own, so TableCell's own `[...],` entries (renderTableCellTypst)
// end up as the table's positional arguments in document order: header cells first,
// then each row's cells.
func renderTableTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*extast.Table)
	if entering {
		io.WriteString(w, "#table(columns: (")
		for i := range n.Alignments {
			if i > 0 {
				io.WriteString(w, ", ")
			}
			io.WriteString(w, "1fr")
		}
		io.WriteString(w, "), align: (")
		for i, a := range n.Alignments {
			if i > 0 {
				io.WriteString(w, ", ")
			}
			io.WriteString(w, typstTableAlign(a))
		}
		io.WriteString(w, "),\n")
	} else {
		io.WriteString(w, ")\n\n")
	}
	return gast.WalkContinue, nil
}

// typstTableAlign maps a GFM column alignment to a Typst alignment
// keyword; AlignNone (no `:` in the delimiter row) maps to Typst's `auto`.
func typstTableAlign(a extast.Alignment) string {
	switch a {
	case extast.AlignLeft:
		return "left"
	case extast.AlignRight:
		return "right"
	case extast.AlignCenter:
		return "center"
	default:
		return "auto"
	}
}

// renderTableHeaderTypst is a transparent wrapper: see renderTableTypst.
func renderTableHeaderTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

// renderTableRowTypst is a transparent wrapper: see renderTableTypst.
func renderTableRowTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

// renderTableCellTypst emits `[` children `],`, i.e. one positional
// argument to the enclosing #table(...) call.
func renderTableCellTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "[")
	} else {
		io.WriteString(w, "],\n")
	}
	return gast.WalkContinue, nil
}

// renderStrikethroughTypst emits `#strike[` children `]`.
func renderStrikethroughTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "#strike[")
	} else {
		io.WriteString(w, "]")
	}
	return gast.WalkContinue, nil
}

// renderDefinitionListTypst emits `#terms(...)`; each contained term is
// paired with the description immediately following it by
// renderDefinitionTermTypst/renderDefinitionDescriptionTypst below.
//
// Simplifying assumption (definition lists are noted as "rare in ebooks",
// SPECS §4): this pairing is exact for the common case of one term
// followed by one description, which is what DefinitionList produces for
// straightforward PHP-Markdown-Extra input. A term followed by more than
// one description, or more than one term sharing a single description,
// is NOT specially handled and is out of scope here (YAGNI/ASR-6); the
// golden tests cover the common one-term/one-description case only.
func renderDefinitionListTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "#terms(\n")
	} else {
		io.WriteString(w, ")\n\n")
	}
	return gast.WalkContinue, nil
}

// renderDefinitionTermTypst opens a `terms.item[term][` pair, leaving the
// description bracket open for the DefinitionDescription that follows.
func renderDefinitionTermTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, "terms.item[")
	} else {
		io.WriteString(w, "][")
	}
	return gast.WalkContinue, nil
}

// renderDefinitionDescriptionTypst closes the `terms.item[term][desc],`
// pair opened by the preceding DefinitionTerm.
func renderDefinitionDescriptionTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		io.WriteString(w, "],\n")
	}
	return gast.WalkContinue, nil
}

// renderVocabularyTypst emits `#vocabulary((phrase:"..",grammar:"..",
// transcription:"..",translation:".."), ...)`, string-escaping every
// field (SPECS §4). Vocabulary has no markdown children (IsRaw, ast.go),
// so the entire call is written on the entering pass, mirroring
// renderVocabulary (renderer.go).
func renderVocabularyTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Vocabulary)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyTypst("V", dir))
	io.WriteString(w, "#vocabulary(dir: ")
	io.WriteString(w, dir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, "\",\n")
	for _, item := range n.Items {
		switch item.Kind {
		case ItemHeader:
			// (kind: "header", level: N, text: "…") — data items carry no kind key (ASR-3).
			io.WriteString(w, `  (kind: "header", level: `)
			io.WriteString(w, strconv.Itoa(item.Level))
			io.WriteString(w, `, text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		case ItemNote:
			io.WriteString(w, `  (kind: "note", text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		default: // ItemData — unchanged dict shape (ASR-3)
			io.WriteString(w, `  (phrase: "`)
			io.WriteString(w, escapeTypstString(item.Phrase))
			io.WriteString(w, `", grammar: "`)
			io.WriteString(w, escapeTypstString(item.Grammar))
			io.WriteString(w, `", transcription: "`)
			io.WriteString(w, escapeTypstString(item.Transcription))
			io.WriteString(w, `", translation: "`)
			io.WriteString(w, escapeTypstString(item.Translation))
			io.WriteString(w, "\"),\n")
		}
	}
	io.WriteString(w, ")\n\n")

	return gast.WalkContinue, nil
}

// renderDialogTypst emits `#dialog((header:"..", content:[<ToTypst(item.
// Content)>]), ...)`. Dialog content recurses through ToTypst, mirroring
// how renderDialog (renderer.go:71) recurses through ToHTML. A parse-time
// bad-indentation error (Dialog.Err, ast.go:67) stops rendering immediately
// and surfaces the error, mirroring renderDialog's WalkStop (renderer.go:
// 65-67).
func renderDialogTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Dialog)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	role := n.As
	if role == "" {
		role = "source"
	}
	io.WriteString(w, badgeOnlyTypst("D", dir))
	io.WriteString(w, "#dialog(dir: ")
	io.WriteString(w, dir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, `", role: "`)
	io.WriteString(w, escapeTypstString(role))
	io.WriteString(w, "\",\n")
	for _, item := range n.Items {
		switch item.Kind {
		case ItemHeader:
			io.WriteString(w, `  (kind: "header", level: `)
			io.WriteString(w, strconv.Itoa(item.Level))
			io.WriteString(w, `, text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		case ItemNote:
			io.WriteString(w, `  (kind: "note", text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		default: // ItemData — unchanged dict shape (ASR-3)
			content, err := ToTypst([]byte(item.Content))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, `  (header: "`)
			io.WriteString(w, escapeTypstString(item.Header))
			io.WriteString(w, `", content: [`)
			w.Write(content)
			io.WriteString(w, "]),\n")
		}
	}
	io.WriteString(w, ")\n\n")

	return gast.WalkContinue, nil
}

// renderParallelTypst emits `#parallel(source-dir: <dir>, script: "<script>",
// (<row dict>, ...), ...)` (SPECS §7.1). Per row, the dict always carries
// `source:` and `translation:` (empty `[]` when absent); `transcription:` is
// emitted ONLY when TranscriptionRaw != "" — key omission (not empty content)
// is how book.typ's parallel() detects transcription presence via
// "transcription" in r (§7.2, reviewer-flagged dict-key-presence idiom).
func renderParallelTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Parallel)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	sourceDir := blockDirection(n.Script)
	io.WriteString(w, "#parallel(source-dir: ")
	io.WriteString(w, sourceDir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, "\",\n")
	for _, row := range n.Rows {
		sourceContent, err := ToTypst([]byte(row.SourceRaw))
		if err != nil {
			return gast.WalkStop, err
		}
		io.WriteString(w, "  (source: [")
		w.Write(sourceContent)
		io.WriteString(w, "], translation: [")
		if row.TranslationRaw != "" {
			translationContent, err := ToTypst([]byte(row.TranslationRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			w.Write(translationContent)
		}
		io.WriteString(w, "]")
		if row.TranscriptionRaw != "" {
			transcriptionContent, err := ToTypst([]byte(row.TranscriptionRaw))
			if err != nil {
				return gast.WalkStop, err
			}
			io.WriteString(w, ", transcription: [")
			w.Write(transcriptionContent)
			io.WriteString(w, "]")
		}
		io.WriteString(w, "),\n")
	}
	io.WriteString(w, ")\n\n")

	return gast.WalkContinue, nil
}

// renderModelsTypst emits `#models((phrase:"..", transcription:"..",
// translation:".."), ...)`, string-escaping every field (mirrors
// renderVocabularyTypst, minus the `grammar` field). Models has no
// markdown children (IsRaw, ast.go), so the entire call is written on the
// entering pass.
func renderModelsTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Models)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	io.WriteString(w, badgeOnlyTypst("M", dir))
	io.WriteString(w, "#models(dir: ")
	io.WriteString(w, dir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, "\",\n")
	for _, item := range n.Items {
		switch item.Kind {
		case ItemHeader:
			io.WriteString(w, `  (kind: "header", level: `)
			io.WriteString(w, strconv.Itoa(item.Level))
			io.WriteString(w, `, text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		case ItemNote:
			io.WriteString(w, `  (kind: "note", text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		default: // ItemData — unchanged dict shape (ASR-3)
			io.WriteString(w, `  (phrase: "`)
			io.WriteString(w, escapeTypstString(item.Phrase))
			io.WriteString(w, `", transcription: "`)
			io.WriteString(w, escapeTypstString(item.Transcription))
			io.WriteString(w, `", translation: "`)
			io.WriteString(w, escapeTypstString(item.Translation))
			io.WriteString(w, "\"),\n")
		}
	}
	io.WriteString(w, ")\n\n")

	return gast.WalkContinue, nil
}

// renderTextblockTypst emits `#textblock(role: "..", dir: <kw>, [ <body> ])`
// (SPECS §7.1, M3). Direction rule (D9): as=transcription is pinned ltr
// (romanization); source/translation/grammar derive direction from the
// block's own script via blockDirection. The body is recursed through
// ToTypst, mirroring renderDialogTypst/renderParallelTypst. The grammar
// table full-width + source-dir rule is handled by book.typ's _sourceDir
// state (D13) — no source-dir argument is passed here.
func renderTextblockTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
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
	dir := blockDirection(n.Script)
	if as == "transcription" {
		dir = "ltr"
	}
	// F-MARK: lift the block's first heading into a "T"-badged title line
	// (badge injected in place, heading otherwise intact); when the block has
	// no heading, prepend a standalone "T" badge line before the wrapper.
	var body string
	if n.Raw != "" {
		content, err := ToTypst([]byte(n.Raw))
		if err != nil {
			return gast.WalkStop, err
		}
		body = string(content)
	}
	body, had := injectBadgeIntoFirstHeadingTypst(body, "T")
	if !had {
		io.WriteString(w, badgeOnlyTypst("T", dir))
	}
	io.WriteString(w, "#textblock(role: \"")
	io.WriteString(w, as)
	io.WriteString(w, "\", dir: ")
	io.WriteString(w, dir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, "\", [\n")
	io.WriteString(w, body)
	io.WriteString(w, "])\n\n")
	return gast.WalkContinue, nil
}

// renderQuestionsTypst emits `#questions((question:"..", answer:".."),
// ...)`, string-escaping every field. Questions has no markdown children
// (IsRaw, ast.go), so the entire call is written on the entering pass;
// book.typ's `questions(..items)` (Cycle 1) decides per item whether to
// render a plain paragraph (no answer) or join it into the current
// aligned two-column run (answer present).
func renderQuestionsTypst(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Questions)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	dir := blockDirection(n.Script)
	role := n.As
	if role == "" {
		role = "source"
	}
	io.WriteString(w, badgeOnlyTypst("Q", dir))
	io.WriteString(w, "#questions(dir: ")
	io.WriteString(w, dir)
	io.WriteString(w, `, script: "`)
	io.WriteString(w, escapeTypstString(n.Script))
	io.WriteString(w, `", role: "`)
	io.WriteString(w, escapeTypstString(role))
	io.WriteString(w, "\",\n")
	for _, item := range n.Items {
		switch item.Kind {
		case ItemHeader:
			io.WriteString(w, `  (kind: "header", level: `)
			io.WriteString(w, strconv.Itoa(item.Level))
			io.WriteString(w, `, text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		case ItemNote:
			io.WriteString(w, `  (kind: "note", text: "`)
			io.WriteString(w, escapeTypstString(item.Text))
			io.WriteString(w, "\"),\n")
		default: // ItemData — unchanged dict shape (ASR-3)
			io.WriteString(w, `  (question: "`)
			io.WriteString(w, escapeTypstString(item.Question))
			io.WriteString(w, `", answer: "`)
			io.WriteString(w, escapeTypstString(item.Answer))
			io.WriteString(w, "\"),\n")
		}
	}
	io.WriteString(w, ")\n\n")

	return gast.WalkContinue, nil
}
