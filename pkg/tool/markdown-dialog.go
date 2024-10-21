package tool

import (
	"bytes"
	"io"

	"github.com/gomarkdown/markdown/ast"
)

type Dialog struct {
	ast.Container
}

type DialogItem struct {
	ast.Container
	Person string
}

var startDialog = []byte("{start-dialog}")
var endDialog = []byte("{end-dialog}")

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

func RenderDialog(w io.Writer, n *Dialog, entering bool) {
	if entering {
		io.WriteString(w, "<dialog>")
	} else {
		io.WriteString(w, "</dialog>")
	}
}

func RenderDialogItem(w io.Writer, n *DialogItem, entering bool) {
	if entering {
		io.WriteString(w, "<item>")
	} else {
		io.WriteString(w, "</item>")
	}
}
