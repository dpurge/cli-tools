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

// mdxListState tracks one level of List nesting for renderListItem:
// whether the enclosing list is ordered, and (for an ordered list) the
// next item number to emit. mdxNodeRenderer.listStack is a stack of these
// so a nested List (inside a ListItem) gets its own counter without
// disturbing the enclosing list's.
type mdxListState struct {
	ordered bool
	counter int
}

// mdxNodeRenderer registers an MDX NodeRendererFunc for every node kind
// in SPECS §4: the standard CommonMark+GFM subset the ebooks use
// (re-emitted as markdown), plus the 3 custom blocks (re-emitted as
// phraseforge fenced blocks, SPECS §4.2-§4.4). Unlike typstNodeRenderer
// (typst_render.go), one instance is constructed PER ToMDX call (mdx.go's
// newMdxRenderer) because emission needs the call's lang/script, and it
// carries render-time state (atLineStart, listStack) that must not leak
// between independent ToMDX calls.
//
// Safety invariant (mirrors typst_render.go's typstNodeRenderer comment):
// the underlying renderer's nodeRendererFuncs slice is sized to (highest
// kind ever registered)+1 (renderer/renderer.go:150); walking a node whose
// Kind() exceeds every registered kind would index out of bounds and
// panic, NOT gracefully skip. This is safe here only because
// RegisterFuncs below registers every node kind that md's shared parser
// configuration (converter.go) can ever produce — the goldmark core
// kinds, the Table/Strikethrough/DefinitionList extension kinds, and this
// package's own Vocabulary/Dialog/Parallel kinds — so the registered
// maximum always covers every kind actually walked, exactly as it does
// for typstNodeRenderer. Kinds deliberately left unregistered (RawHTML,
// HTMLBlock, LinkReferenceDefinition, ...) are lower-ordinal than that
// maximum, so the shared renderer's "unregistered-but-in-range kind ->
// emit nothing, still walk children" fallback (renderer/renderer.go:164)
// applies to them instead of a panic: for RawHTML/HTMLBlock this mirrors
// the HTML renderer's Unsafe=false omission (SPECS §4 last two rows); raw
// HTML must never be passed through into MDX/JSX unescaped.
type mdxNodeRenderer struct {
	lang, script string

	// self is set by newMdxRenderer (mdx.go) right after construction, so
	// renderChildrenToBuf can recurse into this SAME dispatch table for a
	// node's CHILDREN (Blockquote/ListItem, which need to buffer their
	// rendered body before post-processing it with a "> "/indent prefix)
	// without re-invoking the parent node's own wrapper function — which
	// would infinite-loop if the parent node itself, rather than only its
	// children, were re-rendered through it.
	self renderer.Renderer

	// atLineStart tracks whether the NEXT piece of Text/String content
	// would land at the very start of an emitted MDX line (SPECS §5.3):
	// true at the start of a fresh Paragraph/TextBlock/DefinitionTerm and
	// immediately after a Soft/Hard line break; false once any non-break
	// content has been written. renderText/renderString consult it to
	// decide whether escapeMdxLineStart applies to their leading run.
	atLineStart bool

	// listStack is the stack of enclosing List states; renderListItem
	// consults its top for the current marker/ordered-counter.
	listStack []*mdxListState
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *mdxNodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(gast.KindDocument, r.renderDocument)
	reg.Register(gast.KindHeading, r.renderHeading)
	reg.Register(gast.KindParagraph, r.renderParagraph)
	reg.Register(gast.KindTextBlock, r.renderTextBlock)
	reg.Register(gast.KindText, r.renderText)
	reg.Register(gast.KindString, r.renderString)
	reg.Register(gast.KindEmphasis, r.renderEmphasis)
	reg.Register(gast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(gast.KindLink, r.renderLink)
	reg.Register(gast.KindAutoLink, r.renderAutoLink)
	reg.Register(gast.KindImage, r.renderImage)
	reg.Register(gast.KindList, r.renderList)
	reg.Register(gast.KindListItem, r.renderListItem)
	reg.Register(gast.KindBlockquote, r.renderBlockquote)
	reg.Register(gast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(gast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(gast.KindThematicBreak, r.renderThematicBreak)

	reg.Register(extast.KindTable, r.renderTable)
	reg.Register(extast.KindTableHeader, r.renderTableHeader)
	reg.Register(extast.KindTableRow, r.renderTableRow)
	reg.Register(extast.KindTableCell, r.renderTableCell)
	reg.Register(extast.KindStrikethrough, r.renderStrikethrough)
	reg.Register(extast.KindDefinitionList, r.renderDefinitionList)
	reg.Register(extast.KindDefinitionTerm, r.renderDefinitionTerm)
	reg.Register(extast.KindDefinitionDescription, r.renderDefinitionDescription)

	reg.Register(KindVocabulary, r.renderVocabulary)
	reg.Register(KindDialog, r.renderDialog)
	reg.Register(KindParallel, r.renderParallel)
	reg.Register(KindModels, r.renderModels)
	reg.Register(KindQuestions, r.renderQuestions)
	// KindText MUST be registered last (highest ordinal, ASR-1 panic-gate).
	reg.Register(KindText, r.renderTextblock)
}

// renderChildrenToBuf renders node's CHILDREN (never node itself) into a
// fresh buffer via r.self, for callers that need to post-process the
// result: renderBlockquote (prefix every line with "> ") and
// renderListItem (prepend a marker to the first line, indent the rest).
// Rendering each child through r.self (rather than relying on the
// surrounding ast.Walk's own recursion) is what lets the caller return
// gast.WalkSkipChildren and substitute its own post-processed text.
func (r *mdxNodeRenderer) renderChildrenToBuf(source []byte, node gast.Node) (string, error) {
	var buf bytes.Buffer
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if err := r.self.Render(&buf, source, c); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// renderDocument: Document walks its children with no wrapper. ToMDX
// never calls Render on the Document node itself (it walks top-level
// children individually, mdx.go), but this is registered anyway to
// satisfy the panic invariant documented on mdxNodeRenderer.
func (r *mdxNodeRenderer) renderDocument(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

// renderHeading emits "#"xLevel + " " + inline children + "\n\n" (SPECS
// §4.1). Heading text can never be misread as a NEW line-start block
// marker (it is always preceded by "# ..." on the same line), so
// atLineStart is explicitly cleared rather than left in whatever state
// preceded this call.
func (r *mdxNodeRenderer) renderHeading(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Heading)
	if entering {
		r.atLineStart = false
		io.WriteString(w, strings.Repeat("#", n.Level))
		io.WriteString(w, " ")
	} else {
		io.WriteString(w, "\n\n")
	}
	return gast.WalkContinue, nil
}

// renderParagraph emits inline children followed by a blank line. Entering
// a Paragraph is always a fresh MDX line start (SPECS §5.3).
func (r *mdxNodeRenderer) renderParagraph(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.atLineStart = true
	} else {
		io.WriteString(w, "\n\n")
	}
	return gast.WalkContinue, nil
}

// renderTextBlock emits inline children followed by a single newline;
// used for tight list-item bodies. Entering a TextBlock is likewise a
// fresh line start.
func (r *mdxNodeRenderer) renderTextBlock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.atLineStart = true
	} else {
		io.WriteString(w, "\n")
	}
	return gast.WalkContinue, nil
}

// renderText emits the Text node's escaped value, then a break marker: a
// hard break emits CommonMark's two-trailing-space hard break ("  \n"); a
// soft break emits a literal "\n" (D2: preserves the authored per-line
// layout, matching phraseforge's own source style) — either way,
// atLineStart becomes true again for whatever follows, since it starts a
// new emitted line.
//
// The raw segment value is passed through unescapeMarkdownBackslash
// (typst_escape.go, reused as-is per SPECS §3.2) BEFORE escapeMdxText:
// goldmark's Text.Value keeps a source backslash-escape's backslash
// byte-for-byte (see unescapeMarkdownBackslash's doc comment for the full
// discrepancy vs a naive reading of the AST), so skipping this step would
// double-escape (the source's own backslash surviving alongside
// escapeMdxText's newly-inserted one).
func (r *mdxNodeRenderer) renderText(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.Text)
	value := unescapeMarkdownBackslash(string(n.Value(source)))
	escaped := escapeMdxText(value)
	if r.atLineStart {
		escaped = escapeMdxLineStart(escaped)
	}
	io.WriteString(w, escaped)

	switch {
	case n.HardLineBreak():
		io.WriteString(w, "  \n")
		r.atLineStart = true
	case n.SoftLineBreak():
		io.WriteString(w, "\n")
		r.atLineStart = true
	default:
		if escaped != "" {
			r.atLineStart = false
		}
	}
	return gast.WalkContinue, nil
}

// renderString maps a Typographer-emitted HTML entity to its Unicode
// codepoint (typographerEntities, typst_escape.go, reused as-is); any
// other String value is literal text and is escaped like Text (SPECS
// §4.1/§5.2). None of the typographer Unicode substitutions are
// mdxTextMetachars, so they never need escaping themselves.
func (r *mdxNodeRenderer) renderString(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*gast.String)
	var out string
	if u, ok := typographerEntities[string(n.Value)]; ok {
		out = u
	} else {
		out = escapeMdxText(string(n.Value))
		if r.atLineStart {
			out = escapeMdxLineStart(out)
		}
	}
	io.WriteString(w, out)
	if out != "" {
		r.atLineStart = false
	}
	return gast.WalkContinue, nil
}

// renderEmphasis uses the canonical "*"/"**" markup delimiters (Level 2 is
// strong, mirroring goldmark's own HTML renderer, which only
// special-cases Level==2). The delimiter itself can never be misread as a
// line-start marker requiring escaping, but it DOES mean any following
// child Text is no longer at a true line start.
func (r *mdxNodeRenderer) renderEmphasis(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Emphasis)
	delim := "*"
	if n.Level == 2 {
		delim = "**"
	}
	r.atLineStart = false
	io.WriteString(w, delim)
	return gast.WalkContinue, nil
}

// renderCodeSpan reads the CodeSpan's child Text segments directly
// (mirroring typst_render.go's renderCodeSpanTypst, including its
// trailing-newline-becomes-space rule for a span crossing a soft line
// break) rather than walking them through the normal Text renderer,
// because CodeSpan content is CommonMark code-span content: literal, not
// subject to escapeMdxText. The backtick delimiter is widened past any
// backtick run already present in the content (mirrors mdxFence's rule
// for block fences, SPECS §5.1.3), with a single padding space on each
// side when the content itself starts/ends with a backtick or is empty
// (the standard CommonMark code-span disambiguation rule). WalkSkipChildren
// prevents the walker from separately visiting (and re-rendering) those
// Text children.
func (r *mdxNodeRenderer) renderCodeSpan(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
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
	content := buf.String()
	delim := strings.Repeat("`", longestBacktickRun(content)+1)
	pad := ""
	if content == "" || strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
		pad = " "
	}
	io.WriteString(w, delim)
	io.WriteString(w, pad)
	io.WriteString(w, content)
	io.WriteString(w, pad)
	io.WriteString(w, delim)
	return gast.WalkSkipChildren, nil
}

// renderLink emits "[" children "](" escDest [" \"" escTitle "\""] ")"
// (SPECS §4.1): Destination via escapeMdxUrl (§5.5), Title (if present)
// via escapeMdxText since it renders as prose inside the title string.
func (r *mdxNodeRenderer) renderLink(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.Link)
	if entering {
		r.atLineStart = false
		io.WriteString(w, "[")
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "](")
	io.WriteString(w, escapeMdxUrl(string(n.Destination)))
	if len(n.Title) > 0 {
		io.WriteString(w, ` "`)
		io.WriteString(w, escapeMdxText(string(n.Title)))
		io.WriteString(w, `"`)
	}
	io.WriteString(w, ")")
	return gast.WalkContinue, nil
}

// renderAutoLink emits an autolink as a NORMAL link "[label](escUrl)"
// rather than CommonMark's own "<url>" autolink form (SPECS §4.1): a bare
// "<" starting a token is exactly the JSX-element hazard escapeMdxText
// otherwise guards against, so autolinks are normalized to the safe
// bracket form instead. An email autolink gets a "mailto:" prefix if it
// doesn't already have one (mirrors goldmark's own HTML renderer). AutoLink
// carries its label/URL internally (ast/inline.go) rather than as child
// nodes, so everything is written on the entering call.
func (r *mdxNodeRenderer) renderAutoLink(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
	n := node.(*gast.AutoLink)
	url := string(n.URL(source))
	if n.AutoLinkType == gast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(url), "mailto:") {
		url = "mailto:" + url
	}
	io.WriteString(w, "[")
	io.WriteString(w, escapeMdxText(string(n.Label(source))))
	io.WriteString(w, "](")
	io.WriteString(w, escapeMdxUrl(url))
	io.WriteString(w, ")")
	return gast.WalkContinue, nil
}

// renderImage emits "![" escAlt "](" escDest ")" (SPECS §4.1). Unlike the
// Typst renderer (which drops alt text entirely, typst_render.go), MDX
// keeps it: n.Text (ast/ast.go) is deprecated in the installed goldmark
// v1.8.4 ("Use other properties of the node ... i.e. Text.Value") but
// remains fully functional, and is the simplest way to collapse an
// Image's inline children (typically a single Text run) into the plain
// alt string CommonMark expects — mirrors Title's use of the same method
// (mdx.go) for the same reason. WalkSkipChildren mirrors the HTML
// renderer, which also never walks Image's children as ordinary content.
func (r *mdxNodeRenderer) renderImage(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
	n := node.(*gast.Image)
	alt := escapeMdxText(string(n.Text(source)))
	io.WriteString(w, "![")
	io.WriteString(w, alt)
	io.WriteString(w, "](")
	io.WriteString(w, escapeMdxUrl(string(n.Destination)))
	io.WriteString(w, ")")
	return gast.WalkSkipChildren, nil
}

// renderList pushes/pops one mdxListState per List (SPECS §4.1: "- " for
// unordered, "N. " for ordered starting at n.Start), so a nested List
// (inside a ListItem) gets independent numbering from its enclosing list.
// The trailing "\n" on exit gives the customary blank-line separation
// once concatenated after the last item's own trailing newline (mirrors
// Paragraph's "\n\n" trailer).
func (r *mdxNodeRenderer) renderList(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	n := node.(*gast.List)
	if entering {
		r.listStack = append(r.listStack, &mdxListState{ordered: n.IsOrdered(), counter: n.Start})
		return gast.WalkContinue, nil
	}
	r.listStack = r.listStack[:len(r.listStack)-1]
	io.WriteString(w, "\n")
	return gast.WalkContinue, nil
}

// renderListItem buffers the item's own children (renderChildrenToBuf),
// then prepends the marker ("- " or "N. ", from the top of listStack) to
// the first line and indents every continuation line by the marker's
// width (indentContinuation) — the CommonMark rule for a list item body
// spanning more than one line or containing a nested block (e.g. a
// nested List). Buffering first (rather than writing the marker directly
// and streaming children through the normal walk) is what lets a nested
// List's own items end up indented under this one.
func (r *mdxNodeRenderer) renderListItem(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false

	marker := "- "
	if l := len(r.listStack); l > 0 {
		state := r.listStack[l-1]
		if state.ordered {
			marker = strconv.Itoa(state.counter) + ". "
			state.counter++
		}
	}

	body, err := r.renderChildrenToBuf(source, node)
	if err != nil {
		return gast.WalkStop, err
	}
	io.WriteString(w, indentContinuation(body, marker))
	return gast.WalkSkipChildren, nil
}

// indentContinuation prepends marker to body's first line and indents
// every subsequent NON-BLANK line by spaces equal to marker's width,
// leaving blank lines bare (no trailing whitespace). A single trailing
// "\n" is emitted regardless of how many trailing blank lines body itself
// had (renderListItem's caller — the enclosing List/Paragraph/TextBlock —
// supplies its own blank-line separation, so a dangling blank line at the
// very end of an item's own body would only be redundant).
func indentContinuation(body, marker string) string {
	trimmed := strings.TrimRight(body, "\n")
	if trimmed == "" {
		return marker + "\n"
	}
	indent := strings.Repeat(" ", len(marker))
	lines := strings.Split(trimmed, "\n")
	var b strings.Builder
	for i, line := range lines {
		switch {
		case i == 0:
			b.WriteString(marker)
			b.WriteString(line)
		case line == "":
			// blank continuation line: no trailing whitespace
		default:
			b.WriteString(indent)
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderBlockquote buffers the quote's children (renderChildrenToBuf),
// then prefixes every resulting line with "> " (a bare ">" for an
// otherwise-blank line, to avoid trailing whitespace) — SPECS §4.1:
// "render inner blocks (line-start escaping applies to the INNER content,
// computed BEFORE the '> ' prefix is prepended, so the added '> ' is
// never itself escaped)". Supports multi-paragraph quotes (each inner
// Paragraph's own "\n\n" trailer becomes a bare ">" line between quoted
// paragraphs).
func (r *mdxNodeRenderer) renderBlockquote(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
	body, err := r.renderChildrenToBuf(source, node)
	if err != nil {
		return gast.WalkStop, err
	}
	io.WriteString(w, prefixLines(body, ">"))
	io.WriteString(w, "\n")
	return gast.WalkSkipChildren, nil
}

// prefixLines prefixes every line of body (after trimming its own
// trailing blank lines) with prefix: a bare prefix for an otherwise-blank
// line, prefix+" "+content otherwise.
func prefixLines(body, prefix string) string {
	trimmed := strings.TrimRight(body, "\n")
	if trimmed == "" {
		return prefix + "\n"
	}
	lines := strings.Split(trimmed, "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		if line != "" {
			b.WriteString(" ")
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderFencedCodeBlock re-emits a fenced code block literally (SPECS
// §4.1): content is never escaped (it is code, not prose), and the fence
// delimiter is widened past any backtick run already present in the
// content (mdxFence, SPECS §5.1.3). The language info string (if any) is
// carried over unchanged. n.Text (deprecated but functional, see
// renderImage) is the simplest way to read a raw block's full literal
// text, mirroring typst_render.go's identical use for the same node.
func (r *mdxNodeRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
	n := node.(*gast.FencedCodeBlock)
	content := string(n.Text(source))
	fence := mdxFence(content)
	io.WriteString(w, fence)
	if lang := n.Language(source); len(lang) > 0 {
		io.WriteString(w, string(lang))
	}
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	if !strings.HasSuffix(content, "\n") {
		io.WriteString(w, "\n")
	}
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	return gast.WalkSkipChildren, nil
}

// renderCodeBlock re-emits an indented code block as a fenced block (no
// info string, so never a language token); see renderFencedCodeBlock.
func (r *mdxNodeRenderer) renderCodeBlock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	r.atLineStart = false
	n := node.(*gast.CodeBlock)
	content := string(n.Text(source))
	fence := mdxFence(content)
	io.WriteString(w, fence)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	if !strings.HasSuffix(content, "\n") {
		io.WriteString(w, "\n")
	}
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	return gast.WalkSkipChildren, nil
}

// renderThematicBreak emits "---\n\n". ToMDX's top-level orchestration
// (mdx.go) never calls Render on a TOP-LEVEL ThematicBreak (D3: it is
// dropped as a group boundary, never reaches a node renderer at all); this
// func only ever fires for a NESTED thematic break (e.g. inside a
// blockquote or list item, SPECS §4.1).
func (r *mdxNodeRenderer) renderThematicBreak(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "---\n\n")
	r.atLineStart = true
	return gast.WalkContinue, nil
}

// mdxTableAlignMarker maps a GFM column alignment to its Markdown
// delimiter-row cell (SPECS §4.1); AlignNone (no ":" in the source
// delimiter row) maps to a plain "---".
func mdxTableAlignMarker(a extast.Alignment) string {
	switch a {
	case extast.AlignLeft:
		return ":--"
	case extast.AlignRight:
		return "--:"
	case extast.AlignCenter:
		return ":-:"
	default:
		return "---"
	}
}

// renderTable resets atLineStart on entry (table cell content is never at
// a true line start, see renderTableCell) and emits the closing blank
// line on exit, once the header/rows below have written every "| ... |"
// line.
func (r *mdxNodeRenderer) renderTable(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.atLineStart = false
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "\n")
	return gast.WalkContinue, nil
}

// renderTableHeader closes the header row's cells with a final "|\n",
// then emits the GFM alignment delimiter row (SPECS §4.1).
//
// GOLDMARK API DISCREPANCY (verified against the installed
// github.com/yuin/goldmark@v1.8.4): TableHeader.Alignments is NEVER
// populated by goldmark's own table extension. extension/table.go's
// tableParagraphTransformer.Transform builds the header row as a
// *ast.TableRow (which DOES get its Alignments set, via
// ast.NewTableRow(alignments)) and then wraps it with
// extast.NewTableHeader(headerRow) — but that constructor
// (extension/ast/table.go) only copies the row's CHILD cells across, not
// its Alignments field, leaving the resulting TableHeader's own
// Alignments permanently nil. Only the parent Table's Alignments (set
// directly in Transform: "table.Alignments = alignments") and each body
// TableRow's Alignments are reliably populated. This function therefore
// reads the alignment column count/order from node.Parent() (the
// enclosing *extast.Table) rather than from the TableHeader node itself.
func (r *mdxNodeRenderer) renderTableHeader(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "|\n|")
	if table, ok := node.Parent().(*extast.Table); ok {
		for _, a := range table.Alignments {
			io.WriteString(w, " ")
			io.WriteString(w, mdxTableAlignMarker(a))
			io.WriteString(w, " |")
		}
	}
	io.WriteString(w, "\n")
	return gast.WalkContinue, nil
}

// renderTableRow closes a body row's cells with a final "|\n".
func (r *mdxNodeRenderer) renderTableRow(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return gast.WalkContinue, nil
	}
	io.WriteString(w, "|\n")
	return gast.WalkContinue, nil
}

// renderTableCell emits "| " on entry and " " on exit, so consecutive
// cells in a row read as "| a | b | c " (the enclosing TableHeader/
// TableRow supplies the final closing "|"). atLineStart is cleared on
// entry: cell content is always preceded by "| " on the same line, so it
// can never be misread as a NEW block by a downstream MDX parser.
func (r *mdxNodeRenderer) renderTableCell(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.atLineStart = false
		io.WriteString(w, "| ")
	} else {
		io.WriteString(w, " ")
	}
	return gast.WalkContinue, nil
}

// renderStrikethrough emits "~~" children "~~".
func (r *mdxNodeRenderer) renderStrikethrough(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	r.atLineStart = false
	io.WriteString(w, "~~")
	return gast.WalkContinue, nil
}

// renderDefinitionList is a transparent wrapper: DefinitionTerm and
// DefinitionDescription (below) already supply all the newlines needed
// ("Term\n: Description\n\n", PHP-Markdown-Extra syntax, SPECS §4.1), so
// this adds nothing of its own.
//
// Simplifying assumption (definition lists are noted as "rare in
// ebooks", SPECS §4): this pairing is exact for the common case of one
// term followed by one TIGHT description. A term followed by more than
// one description, more than one term sharing a single description, or a
// LOOSE description (whose content is a Paragraph rather than a
// TextBlock, contributing its own "\n\n") is NOT specially handled and
// may add one extra blank line — out of scope here (YAGNI/ASR-5),
// mirroring typst_render.go's renderDefinitionListTypst's identical
// caveat.
func (r *mdxNodeRenderer) renderDefinitionList(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

// renderDefinitionTerm emits the term's inline content followed by a
// newline; entering is a fresh line start.
func (r *mdxNodeRenderer) renderDefinitionTerm(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		r.atLineStart = true
	} else {
		io.WriteString(w, "\n")
	}
	return gast.WalkContinue, nil
}

// renderDefinitionDescription emits ": " before the description's inline
// content. The description's own body is a TextBlock in the common tight
// case (SPECS §4/verified against the installed goldmark: DefinitionList
// parses a single-line description into a TextBlock child, mirroring a
// tight ListItem), which already contributes its own trailing "\n"; this
// adds exactly one more "\n" on exit so the pair combines to a blank-line
// block trailer ("Term\n: Description\n\n"), matching every other
// top-level block's trailer convention (Paragraph, List, ...).
func (r *mdxNodeRenderer) renderDefinitionDescription(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		io.WriteString(w, ": ")
		r.atLineStart = false
	} else {
		io.WriteString(w, "\n")
	}
	return gast.WalkContinue, nil
}

// renderVocabulary emits the `vocabulary` fence (SPECS §4.2): one line per
// VocabularyItem, "phrase[ {grammar}][ [transcription]][ = translation]"
// with each bracketed part omitted when its field is empty. This
// round-trips phraseforge's StructuredBody.parseStructuredEntry (splits
// translation at the first " = "/"=", extracts the first "{...}" as
// grammar and the LAST "[...]" as transcription): our field order plus
// our OWN parser's tail-to-head split (parser.go's parseVocabularyItems)
// invert exactly for the common case. Vocabulary has no markdown children
// (IsRaw, ast.go), so the entire fence is written on the entering call,
// mirroring renderVocabulary (renderer.go) and renderVocabularyTypst
// (typst_render.go). Fence content is LITERAL — never escaped (SPECS
// §5.1) — only the fence delimiter itself is widened (mdxFence) past any
// backtick run the content might contain.
func (r *mdxNodeRenderer) renderVocabulary(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Vocabulary)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	var body strings.Builder
	for i, item := range n.Items {
		if i > 0 {
			body.WriteString("\n")
		}
		body.WriteString(item.Phrase)
		if item.Grammar != "" {
			body.WriteString(" {")
			body.WriteString(item.Grammar)
			body.WriteString("}")
		}
		if item.Transcription != "" {
			body.WriteString(" [")
			body.WriteString(item.Transcription)
			body.WriteString("]")
		}
		if item.Translation != "" {
			body.WriteString(" = ")
			body.WriteString(item.Translation)
		}
	}
	content := body.String()
	fence := mdxFence(content)

	lang := r.lang
	if n.Lang != "" {
		lang = n.Lang
	}
	script := r.script
	if n.Script != "" {
		script = n.Script
	}
	io.WriteString(w, fence)
	io.WriteString(w, "vocabulary lang=")
	io.WriteString(w, lang)
	io.WriteString(w, " script=")
	io.WriteString(w, script)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	io.WriteString(w, "\n")
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	r.atLineStart = true

	return gast.WalkContinue, nil
}

// renderDialog emits the `dialog` fence (SPECS §4.3): a marker line per
// DialogItem ("--:" when Header=="—", the em-dash getDialogItemHeader
// produces for an anonymous "--:" turn; otherwise "@"+Header, Header
// already retaining its trailing colon per parser.go), followed by its
// Content re-indented 2 spaces per non-blank line (blank lines kept
// blank) — this round-trips phraseforge's lessonElements.ts
// indentPattern/joinParagraphs turn-parsing rules. Content is the ORIGINAL
// raw markdown captured at parse time, emitted verbatim (never re-run
// through a renderer): fence bodies are literal in MDX (SPECS §5.1), so
// inline markdown like "**bold**" inside a turn survives untouched. A
// parse-time bad-indentation error (Dialog.Err, ast.go) stops rendering
// immediately and surfaces the error, mirroring renderDialog's WalkStop
// (renderer.go) and renderDialogTypst's identical handling.
func (r *mdxNodeRenderer) renderDialog(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Dialog)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	var body strings.Builder
	for _, item := range n.Items {
		if item.Header == "—" {
			body.WriteString("--:\n")
		} else {
			body.WriteString("@")
			body.WriteString(item.Header)
			body.WriteString("\n")
		}
		for _, line := range strings.Split(item.Content, "\n") {
			if line == "" {
				body.WriteString("\n")
			} else {
				body.WriteString("  ")
				body.WriteString(line)
				body.WriteString("\n")
			}
		}
	}
	content := strings.TrimRight(body.String(), "\n")
	fence := mdxFence(content)

	lang := r.lang
	if n.Lang != "" {
		lang = n.Lang
	}
	script := r.script
	if n.Script != "" {
		script = n.Script
	}
	io.WriteString(w, fence)
	io.WriteString(w, "dialog lang=")
	io.WriteString(w, lang)
	io.WriteString(w, " script=")
	io.WriteString(w, script)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	io.WriteString(w, "\n")
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	r.atLineStart = true

	return gast.WalkContinue, nil
}

// renderParallel emits the `parallel` fence (SPECS §4.4, a NEW format —
// phraseforge has no remark case for it yet, D5): rows joined by a lone
// "===" line; within a row, MainRaw and (if present) SecondaryRaw joined
// by a lone "---" line. This losslessly mirrors parseParallelRows
// (parser.go), which splits on "\n===\n" between rows and the LAST
// "\n---\n" within a row — so a "---" thematic break inside a row's own
// MainRaw content round-trips correctly. Row/cell content is the raw
// markdown captured at parse time, emitted verbatim (fence bodies are
// literal, SPECS §5.1) — never re-run through a renderer, unlike
// renderParallelTypst/renderParallel (renderer.go/typst_render.go), which
// recurse through ToHTML/ToTypst because THEIR output format is not
// itself markdown.
func (r *mdxNodeRenderer) renderParallel(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Parallel)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	rowStrs := make([]string, 0, len(n.Rows))
	for _, row := range n.Rows {
		if row.SecondaryRaw != "" {
			rowStrs = append(rowStrs, row.MainRaw+"\n---\n"+row.SecondaryRaw)
		} else {
			rowStrs = append(rowStrs, row.MainRaw)
		}
	}
	content := strings.Join(rowStrs, "\n===\n")
	fence := mdxFence(content)

	lang := r.lang
	if n.Lang != "" {
		lang = n.Lang
	}
	script := r.script
	if n.Script != "" {
		script = n.Script
	}
	io.WriteString(w, fence)
	io.WriteString(w, "parallel lang=")
	io.WriteString(w, lang)
	io.WriteString(w, " script=")
	io.WriteString(w, script)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	io.WriteString(w, "\n")
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	r.atLineStart = true

	return gast.WalkContinue, nil
}

// renderModels emits the `models` fence: one line per ModelsItem,
// "phrase[ [transcription]][ = translation]" with each bracketed part
// omitted when its field is empty (mirrors renderVocabulary, minus the
// `{grammar}` part). This round-trips parser.go's parseModelsItems, which
// splits translation off at the FIRST " = " occurrence and transcription
// at a trailing "[...]" — the field order emitted here (phrase, then
// "[transcription]", then " = translation") inverts exactly for the
// common case. Models has no markdown children (IsRaw, ast.go), so the
// entire fence is written on the entering call. Fence content is LITERAL
// — never escaped (mirrors renderVocabulary/renderParallel) — only the
// fence delimiter itself is widened (mdxFence) past any backtick run the
// content might contain.
func (r *mdxNodeRenderer) renderModels(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Models)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	var body strings.Builder
	for i, item := range n.Items {
		if i > 0 {
			body.WriteString("\n")
		}
		body.WriteString(item.Phrase)
		if item.Transcription != "" {
			body.WriteString(" [")
			body.WriteString(item.Transcription)
			body.WriteString("]")
		}
		if item.Translation != "" {
			body.WriteString(" = ")
			body.WriteString(item.Translation)
		}
	}
	content := body.String()
	fence := mdxFence(content)

	lang := r.lang
	if n.Lang != "" {
		lang = n.Lang
	}
	script := r.script
	if n.Script != "" {
		script = n.Script
	}
	io.WriteString(w, fence)
	io.WriteString(w, "models lang=")
	io.WriteString(w, lang)
	io.WriteString(w, " script=")
	io.WriteString(w, script)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	io.WriteString(w, "\n")
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	r.atLineStart = true

	return gast.WalkContinue, nil
}

// renderQuestions emits the `questions` fence: one line per QuestionItem,
// "question[ = answer]" with the " = answer" part omitted for a
// question-only line. This round-trips parser.go's parseQuestionsItems,
// which splits at the FIRST " = " occurrence. Questions has no markdown
// children (IsRaw, ast.go), so the entire fence is written on the entering
// call. Fence content is LITERAL — never escaped — only the fence
// delimiter itself is widened (mdxFence) past any backtick run the
// content might contain.
func (r *mdxNodeRenderer) renderQuestions(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Questions)
	if n.Err != nil {
		return gast.WalkStop, n.Err
	}

	var body strings.Builder
	for i, item := range n.Items {
		if i > 0 {
			body.WriteString("\n")
		}
		body.WriteString(item.Question)
		if item.Answer != "" {
			body.WriteString(" = ")
			body.WriteString(item.Answer)
		}
	}
	content := body.String()
	fence := mdxFence(content)

	lang := r.lang
	if n.Lang != "" {
		lang = n.Lang
	}
	script := r.script
	if n.Script != "" {
		script = n.Script
	}
	io.WriteString(w, fence)
	io.WriteString(w, "questions lang=")
	io.WriteString(w, lang)
	io.WriteString(w, " script=")
	io.WriteString(w, script)
	io.WriteString(w, "\n")
	io.WriteString(w, content)
	io.WriteString(w, "\n")
	io.WriteString(w, fence)
	io.WriteString(w, "\n\n")
	r.atLineStart = true

	return gast.WalkContinue, nil
}

// renderTextblock emits `<Text [as="X"] lang="L" script="S">` for a
// {start-text as=X} node (SPECS §7.7, OI-7, M3). as="source" omits the
// as= attribute (phraseforge corpus default, Text.tsx:23-43). lang/script
// come from the node when parsed off the marker; otherwise fall back to the
// call-level r.lang/r.script. The Raw inner markdown is emitted verbatim
// (fence bodies are literal in MDX, SPECS §5.1) — not re-run through a
// renderer — then the closing </Text> tag closes the element.
func (r *mdxNodeRenderer) renderTextblock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
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
	lang := n.Lang
	if lang == "" {
		lang = r.lang
	}
	script := n.Script
	if script == "" {
		script = r.script
	}
	io.WriteString(w, "<Text")
	if as != "source" {
		io.WriteString(w, " as=\"")
		io.WriteString(w, as)
		io.WriteString(w, "\"")
	}
	if lang != "" {
		io.WriteString(w, " lang=\"")
		io.WriteString(w, lang)
		io.WriteString(w, "\"")
	}
	if script != "" {
		io.WriteString(w, " script=\"")
		io.WriteString(w, script)
		io.WriteString(w, "\"")
	}
	io.WriteString(w, ">\n\n")
	if n.Raw != "" {
		io.WriteString(w, n.Raw)
		if !strings.HasSuffix(n.Raw, "\n") {
			io.WriteString(w, "\n")
		}
	}
	io.WriteString(w, "\n</Text>\n\n")
	r.atLineStart = true
	return gast.WalkContinue, nil
}
