package tool

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
)

type Parallel struct {
	ast.Container
}

type ParallelBlock struct {
	ast.Container
}

type ParallelLeft struct {
	ast.Container
}

type ParallelRight struct {
	ast.Container
}

var startParallel = []byte("{start-parallel}")
var endParallel = []byte("{end-parallel}")

func (n *Parallel) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *ParallelBlock:
		return true
	}
}

func (n *ParallelBlock) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *ParallelLeft:
		return true
	case *ParallelRight:
		return true
	}
}

func ParseParallel(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, startParallel) {
		return nil, nil, 0
	}
	start := bytes.Index(data, startParallel)
	end := bytes.Index(data[start:], endParallel)
	if end < 0 {
		return nil, data, 0
	}
	end = end + start

	res := &Parallel{}

	// todo

	return res, nil, end + len(endParallel)
}
