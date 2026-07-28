package markdown

import (
	"bytes"
	"os"

	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// typstRenderer is a dedicated Typst renderer for the AST produced by the
// package's shared parser (md.Parser(), converter.go). It is built
// directly with renderer.NewRenderer, registering ONLY Typst
// NodeRendererFuncs (typst_render.go) — NOT a second goldmark.New(), which
// would auto-install goldmark's HTML node renderers at priority 1000
// (SPECS §3.1 Decision D1) and contaminate a Typst-only renderer.
//
// Safety invariant (SPECS §3.1, review-flagged): the underlying
// renderer's nodeRendererFuncs slice is sized to (highest kind ever
// passed to Register)+1 (renderer/renderer.go:150); walking a node whose
// Kind() exceeds every registered kind would index out of bounds and
// panic, NOT gracefully skip. This is safe here only because
// typstNodeRenderer.RegisterFuncs (typst_render.go) registers every node
// kind that md's shared parser configuration (converter.go) can ever
// produce — the goldmark core kinds, the Table/Strikethrough/
// DefinitionList extension kinds, and this package's own
// Vocabulary/Dialog/Parallel kinds (whose ast.NewNodeKind calls run,
// per Go's package-init order, after goldmark's own packages, so they
// carry the highest ordinals of the set) — so the registered maximum
// always covers every kind actually walked. Adding an extension to md
// that introduces a new node kind without a corresponding registration
// in typst_render.go would reintroduce this panic risk.
var typstRenderer = renderer.NewRenderer(renderer.WithNodeRenderers(
	util.Prioritized(&typstNodeRenderer{}, 100),
))

// ToTypst converts markdown source into Typst markup. It parses with the
// SAME parser instance ToHTML uses (md.Parser(), converter.go) so the AST
// is identical to ToHTML's; only the renderer differs (typstRenderer
// above, never md's own HTML renderer). Dialog/Parallel cell content
// recurses through ToTypst exactly as the HTML renderer recurses through
// ToHTML (renderer.go:71,97-112).
//
// ToTypst is a pure function of its input and is safe under the same
// parse-then-render invariant as ToHTML/FileToHTML (converter.go:29-33):
// recursive calls only ever happen once the outer document's parse phase
// has fully completed.
func ToTypst(source []byte) ([]byte, error) {
	doc := md.Parser().Parse(text.NewReader(source))
	var buf bytes.Buffer
	if err := typstRenderer.Render(&buf, source, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FileToTypst reads filename and converts its content into Typst markup.
func FileToTypst(filename string) (string, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	body, err := ToTypst(source)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
