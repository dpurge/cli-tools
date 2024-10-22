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
	PersonName string
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
	personName := ""
	for _, s := range lines {
		s = strings.TrimRight(s, " *")
		if s == "--:" {
			if len(buf) > 0 {
				n := getDialogItem(personName, buf)
				n.SetParent(res)
				items = append(items, n)
				buf = nil
			}
			personName = ""
			continue
		}
		if len(s) > 3 && s[0] == '@' && s[len(s)-1] == ':' {
			if len(buf) > 0 {
				n := getDialogItem(personName, buf)
				n.SetParent(res)
				items = append(items, n)
				buf = nil
			}
			personName = s[1 : len(s)-1]
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
		n := getDialogItem(personName, buf)
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
		io.WriteString(w, "<div class=\"dialog-person\">")
		io.WriteString(w, n.PersonName)
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "<div class=\"dialog-content\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</div>\n")
		io.WriteString(w, "</div>\n")
	}
}

func getDialogItem(person string, lines []string) ast.Node {
	txt := strings.TrimSpace(strings.Join(lines, "\n"))
	n := &DialogItem{}
	n.PersonName = person
	n.Content = []byte(txt)
	return n
}
