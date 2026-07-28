package markdown

import (
	gast "github.com/yuin/goldmark/ast"
)

// Markers for a future `{start-interlinear}` ... `{end-interlinear}` block
// (word-by-word glossed text). Ported from the unfinished
// markdown-interlinear.go stub, which was never wired into parsing either.
var (
	startInterlinear = []byte("{start-interlinear}")
	endInterlinear   = []byte("{end-interlinear}")
)

// Interlinear is an inactive stub node type. No BlockParser or NodeRenderer
// is registered for KindInterlinear (see extension.go / converter.go), so
// it has no effect on conversion — it exists only so the eventual feature
// has a place to grow into.
type Interlinear struct {
	gast.BaseBlock
}

// Kind implements ast.Node.
func (n *Interlinear) Kind() gast.NodeKind { return KindInterlinear }

// Dump implements ast.Node.
func (n *Interlinear) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
