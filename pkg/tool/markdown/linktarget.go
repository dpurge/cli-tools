package markdown

import (
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// linkTargetTransformer adds target="_blank" to every link in the
// document. goldmark has no built-in option for this (unlike gomarkdown's
// html.HrefTargetBlank renderer flag), so it is applied as an
// ASTTransformer instead. Both explicit links (*ast.Link) and bare-URL
// autolinks (*ast.AutoLink, produced by extension.Linkify) are covered —
// gomarkdown's renderer flag applied to every anchor regardless of type,
// and goldmark's default renderers for both honor the "target" attribute
// (it is in LinkAttributeFilter).
type linkTargetTransformer struct{}

// Transform implements parser.ASTTransformer.
func (t *linkTargetTransformer) Transform(doc *gast.Document, reader text.Reader, pc parser.Context) {
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			switch n.(type) {
			case *gast.Link, *gast.AutoLink:
				n.SetAttributeString("target", "_blank")
			}
		}
		return gast.WalkContinue, nil
	})
}
