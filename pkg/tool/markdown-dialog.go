package tool

import (
	"bytes"
	"io"
	"log"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

type Dialog struct {
	ast.Container
}

type DialogItem struct {
	ast.Leaf
	Header string
}

var startDialog = []byte("{start-dialog}")
var endDialog = []byte("{end-dialog}")

func (n *Dialog) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *DialogItem:
		return true
	}
}

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

	block := data[start+len(startDialog) : end]
	lines := strings.Split(strings.TrimSpace(string(block)), "\n")

	res := &Dialog{}

	items := []ast.Node{}
	buf := []string{}
	header := ""
	for _, s := range lines {
		s = strings.TrimRight(s, " *")
		if isDialogItemHeader(s) {
			if len(buf) > 0 {
				n := getDialogItem(header, buf)
				n.SetParent(res)
				items = append(items, n)
				buf = nil
			}
			header = getDialogItemHeader(s)
			continue
		}
		if s == "" {
			buf = append(buf, s)
			continue
		}
		if len(s) > 2 && s[:2] == "  " {
			buf = append(buf, s[2:])
			continue
		}

		log.Fatal("Wrong line indentation for dialog item: " + s)
	}
	if len(buf) > 0 {
		n := getDialogItem(header, buf)
		n.SetParent(res)
		items = append(items, n)
		buf = nil
	}

	res.SetChildren(items)
	return res, nil, end + len(endDialog)
}

func RenderDialog(w io.Writer, n *Dialog, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"dialog\">\n")
	} else {
		io.WriteString(w, "</div>\n")
	}
}

func RenderDialogItem(w io.Writer, n *DialogItem, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"dialog-item\">\n")
		io.WriteString(w, "<div class=\"dialog-header\">")
		io.WriteString(w, n.Header)
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "<div class=\"dialog-content\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "</div>\n")
	}
}

func getDialogItem(header string, lines []string) ast.Node {
	txt := strings.TrimSpace(strings.Join(lines, "\n"))
	doc, _ := MarkdownToHTML([]byte(txt))
	n := &DialogItem{}
	n.Header = header
	n.Content = []byte(doc)
	return n
}

func isDialogItemHeader(header string) bool {
	if len(header) < 3 {
		return false
	}
	if header == "--:" {
		return true
	}
	if !(strings.HasPrefix(header, "@") || strings.HasPrefix(header, "＠")) {
		return false
	}
	if !(strings.HasSuffix(header, ":") || strings.HasSuffix(header, "︰") || strings.HasSuffix(header, "：")) {
		return false
	}
	return true
}

func getDialogItemHeader(header string) string {
	res := "—"
	if strings.HasPrefix(header, "@") {
		res = strings.TrimLeft(header, "@")
	}
	if strings.HasPrefix(header, "＠") {
		res = strings.TrimLeft(header, "＠")
	}
	return res
}
