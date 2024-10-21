package tool

import (
	"bytes"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

type Dialog struct {
	ast.Container
	// ImageURLS []string
}

var startDialog = []byte(":::Start-Dialog")
var endDialog = []byte(":::End")

func ParseDialog(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, startDialog) {
		return nil, nil, 0
	}
	start := bytes.Index(data, startDialog)
	end := bytes.Index(data[start:], endDialog)
	if end < 0 {
		return nil, data, 0
	}
	end = end + start
	res := &Dialog{}
	return res, data[start+len(startDialog) : end], end + len(endDialog)
}

func RenderDialog(w io.Writer, s *Dialog, entering bool) {
	if entering {
		io.WriteString(w, "<dialog>")
	} else {
		io.WriteString(w, "</dialog>")
	}
}
