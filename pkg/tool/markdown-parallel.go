package tool

import (
	"bytes"
	"io"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

type Parallel struct {
	ast.Container
}

type ParallelRow struct {
	ast.Container
}

type ParallelCellMain struct {
	ast.Leaf
}

type ParallelCellSecondary struct {
	ast.Leaf
}

var startParallel = []byte("{start-parallel}")
var endParallel = []byte("{end-parallel}")

func (n *Parallel) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *ParallelRow:
		return true
	}
}

func (n *ParallelRow) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *ParallelCellMain:
		return true
	case *ParallelCellSecondary:
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

	block := data[start+len(startVocabulary) : end]
	chunks := strings.Split(strings.TrimSpace(string(block)), "\n===\n")

	res := &Parallel{}
	items := []ast.Node{}
	for _, s := range chunks {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		item := &ParallelRow{}
		item.SetParent(res)
		children := []ast.Node{}

		first := ""
		last := ""

		i := strings.LastIndex(s, "\n---\n")
		if i != -1 {
			last = strings.TrimSpace(s[i+5:])
			s = strings.TrimSpace(s[:i])
		}

		first = s
		s = ""

		if first != "" {
			doc, _ := MarkdownToHTML([]byte(first))
			n := &ParallelCellMain{}
			n.Content = []byte(doc)
			n.SetParent(item)
			children = append(children, n)
		}

		if last != "" {
			doc, _ := MarkdownToHTML([]byte(last))
			n := &ParallelCellSecondary{}
			n.Content = []byte(doc)
			n.SetParent(item)
			children = append(children, n)
		}

		item.SetChildren(children)
		items = append(items, item)
	}
	res.SetChildren(items)
	return res, nil, end + len(endParallel)
}

func RenderParallel(w io.Writer, n *Parallel, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"parallel\">\n")
	} else {
		io.WriteString(w, "</div>\n")
	}
}

func RenderParallelRow(w io.Writer, n *ParallelRow, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"parallel-row\">\n")
	} else {
		io.WriteString(w, "</div>\n")
	}
}

func RenderParallelCellMain(w io.Writer, n *ParallelCellMain, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"parallel-cell main\">\n")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "\n</div>\n")
	}
}

func RenderParallelCellSecondary(w io.Writer, n *ParallelCellSecondary, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"parallel-cell secondary\">\n")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "\n</div>\n")
	}
}
