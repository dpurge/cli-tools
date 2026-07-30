package markdown

import (
	"bytes"
	"os"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// newMdxRenderer builds a per-call MDX renderer bound to lang/script
// (SPECS §3.1 Decision D1). Unlike typstRenderer (typst.go, a package-level
// var shared across all ToTypst calls), this is constructed fresh inside
// EVERY ToMDX call: the custom-block fences (§4.2-§4.4) and the <Text>
// wrapper (§7.5) need to carry the CALLER's lang/script, and
// mdxNodeRenderer also carries per-render mutable state (atLineStart,
// listStack, mdx_render.go) that must never leak between independent
// ToMDX calls. The cost of one renderer.NewRenderer per call is
// negligible for a CLI exporting a handful of files (SPECS §3.1).
//
// self is wired up AFTER construction so renderChildrenToBuf
// (mdx_render.go, used by renderBlockquote/renderListItem) can recurse
// into this SAME dispatch table for a node's children. This is safe
// against renderer.Renderer's internal sync.Once (renderer/renderer.go):
// that Once always finishes building the dispatch table before the FIRST
// Render call's ast.Walk begins, and every recursive call this package
// makes happens from INSIDE a node-renderer callback — i.e. strictly
// after that first Once.Do has already returned — so a nested Render
// call only ever sees Once already-fired and proceeds straight to its own
// ast.Walk; there is no reentrancy or deadlock risk.
func newMdxRenderer(lang, script string) renderer.Renderer {
	nr := &mdxNodeRenderer{lang: lang, script: script}
	rend := renderer.NewRenderer(renderer.WithNodeRenderers(
		util.Prioritized(nr, 100),
	))
	nr.self = rend
	return rend
}

// ToMDX converts markdown source into an MDX document BODY (no
// frontmatter — that is the exporter's concern, SPECS §6) for a
// phraseforge lesson, given the chapter's raw ISO-639-3 lang and
// lowercase-ISO-15924 script codes (passed through verbatim into the
// fences and the <Text> tag, SPECS §7.1).
//
// It parses with the SAME parser instance ToHTML/ToTypst use (md.Parser(),
// converter.go) so the AST is identical; only the emission differs. This
// is NOT a single whole-document Render (SPECS §3.1 Decision D1): it walks
// the Document's TOP-LEVEL children and classifies each one —
//
//	Heading                        -> flush prose, emit "#"xLevel standalone
//	Vocabulary/Dialog/Parallel/
//	Models/Questions                -> flush prose, emit its fence (§4.2-4.4,
//	                                    models/questions mirror the same pattern)
//	ThematicBreak                   -> flush prose, DROP (D3: the ebook's
//	                                    own vocab/reading separator; no
//	                                    phraseforge lesson places an <hr>
//	                                    between blocks)
//	anything else (Paragraph, List,
//	Blockquote, Table, CodeBlock...) -> accumulate into the current
//	                                    contiguous prose run
//
// and on every boundary (or at EOF) wraps the accumulated run's re-emitted
// markdown in exactly one "<Text lang=\"L\" script=\"S\">...</Text>"
// (SPECS §7.5) — matching the phraseforge single-Text-per-passage
// convention (docs/lat/a1/2026-06-10-a.mdx). A source-slice approach was
// rejected (SPECS §3.1): the custom-block nodes' Lines() exclude their
// "{start-*}"/"{end-*}" markers (parser.go), so a reliable source span
// cannot cleanly bound them; re-emitting from the already-parsed AST has
// no such fragility.
//
// Every top-level element (heading, fence, "<Text>...</Text>" group) is
// emitted with a trailing blank line for uniform separation, matching the
// real phraseforge corpus's between-block spacing; ToMDX then trims the
// document's OWN final trailing blank line down to a single newline, so
// the body never ends with dangling blank lines.
func ToMDX(source []byte, lang, script string) ([]byte, error) {
	source = normalizeNewlines(source)
	doc := md.Parser().Parse(text.NewReader(source))
	r := newMdxRenderer(lang, script)

	var out bytes.Buffer
	var proseRun []gast.Node

	flush := func() error {
		if len(proseRun) == 0 {
			return nil
		}
		out.WriteString(`<Text lang="`)
		out.WriteString(escapeMdxAttr(lang))
		out.WriteString(`" script="`)
		out.WriteString(escapeMdxAttr(script))
		out.WriteString("\">\n\n")
		for _, n := range proseRun {
			if err := r.Render(&out, source, n); err != nil {
				return err
			}
		}
		out.WriteString("</Text>\n\n")
		proseRun = nil
		return nil
	}

	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		switch n.Kind() {
		case gast.KindHeading:
			if err := flush(); err != nil {
				return nil, err
			}
			if err := r.Render(&out, source, n); err != nil {
				return nil, err
			}
		case KindVocabulary, KindDialog, KindParallel, KindModels, KindQuestions, KindText:
			if err := flush(); err != nil {
				return nil, err
			}
			if err := r.Render(&out, source, n); err != nil {
				return nil, err
			}
		case gast.KindThematicBreak:
			if err := flush(); err != nil {
				return nil, err
			}
			// D3: a top-level thematic break is a group boundary that is
			// dropped, never rendered.
		default:
			proseRun = append(proseRun, n)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}

	result := bytes.TrimRight(out.Bytes(), "\n")
	if len(result) == 0 {
		return result, nil
	}
	return append(result, '\n'), nil
}

// FileToMDX reads filename and converts its content into an MDX body via
// ToMDX (mirrors FileToTypst/FileToHTML, typst.go/converter.go).
func FileToMDX(filename, lang, script string) (string, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	body, err := ToMDX(source, lang, script)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Title returns the first top-level level-1 Heading's plain text, or ""
// if the document has none (SPECS §6). Unlike tool.GetHtmlTitle
// (tool/html.go), it never errors when no H1 is present — the caller
// decides a fallback (pkg/ebook's exporter falls back to the file
// basename, Batch 3b) — and it works directly off the AST rather than
// scraping rendered HTML.
//
// h.Text (ast/ast.go's BaseNode.Text) is deprecated in the installed
// goldmark v1.8.4 ("Use other properties of the node to get the text
// value ... i.e. Text.Value") but remains fully functional: for a Heading
// it concatenates all descendant inline nodes' own text (rejoining a
// Soft-broken run with "\n"), which is exactly the "drop the markup, keep
// the plain text" behavior a title field needs (e.g. a heading
// "# *Emphasized*" yields the plain string "Emphasized") — reimplementing
// that walk by hand would just duplicate BaseNode.Text's logic for no
// behavioral gain, so using it here (and in renderImage's alt-text
// extraction, mdx_render.go, for the identical reason) is acceptable.
func Title(source []byte) (string, error) {
	doc := md.Parser().Parse(text.NewReader(source))
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*gast.Heading); ok && h.Level == 1 {
			return string(h.Text(source)), nil
		}
	}
	return "", nil
}
