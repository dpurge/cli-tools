package tool

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

type Interlinear struct {
	ast.Container
}

var startInterlinear = []byte("{start-interlinear}")
var endInterlinear = []byte("{end-interlinear}")

func (n *Interlinear) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	}
}

func ParseInterlinear(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, startInterlinear) {
		return nil, nil, 0
	}
	start := bytes.Index(data, startInterlinear)
	end := bytes.Index(data[start:], endInterlinear)
	if end < 0 {
		return nil, data, 0
	}
	end = end + start

	res := &Parallel{}

	// todo

	return res, nil, end + len(endInterlinear)
}
