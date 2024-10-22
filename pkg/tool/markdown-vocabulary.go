package tool

import (
	"bytes"
	"io"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

type VocabularyPhrase struct {
	ast.Leaf
}

type VocabularyGrammar struct {
	ast.Leaf
}

type VocabularyTranscription struct {
	ast.Leaf
}

type VocabularyTranslation struct {
	ast.Leaf
}

type VocabularyItem struct {
	ast.Container
}

type Vocabulary struct {
	ast.Container
}

func (n *Vocabulary) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *VocabularyPhrase:
		return true
	case *VocabularyGrammar:
		return true
	case *VocabularyTranscription:
		return true
	case *VocabularyTranslation:
		return true
	}
}

func (n *VocabularyItem) CanContain(v ast.Node) bool {
	switch v.(type) {
	default:
		return false
	case *VocabularyItem:
		return true
	}
}

var startVocabulary = []byte("{start-vocabulary}")
var endVocabulary = []byte("{end-vocabulary}")

func ParseVocabulary(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, startVocabulary) {
		return nil, nil, 0
	}
	start := bytes.Index(data, startVocabulary)
	end := bytes.Index(data[start:], endVocabulary)
	if end < 0 {
		return nil, data, 0
	}
	end = end + start

	block := data[start+len(startVocabulary) : end]
	lines := strings.Split(strings.TrimSpace(string(block)), "\n")

	res := &Vocabulary{}
	items := []ast.Node{}
	for _, s := range lines {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		item := &VocabularyItem{}
		item.SetParent(res)
		children := []ast.Node{}

		phrase := ""
		grammar := ""
		transcription := ""
		translation := ""

		i := strings.LastIndex(s, "=")
		if i != -1 {
			translation = strings.TrimSpace(s[i+1:])
			s = strings.TrimSpace(s[:i])
		}

		if s[len(s)-1:] == "]" {
			i = strings.LastIndex(s, "[")
			transcription = strings.TrimSpace(s[i+1 : len(s)-1])
			s = strings.TrimSpace(s[:i])
		}

		if s[len(s)-1:] == "}" {
			i = strings.LastIndex(s, "{")
			grammar = strings.TrimSpace(s[i+1 : len(s)-1])
			s = strings.TrimSpace(s[:i])
		}

		phrase = s
		s = ""

		if phrase != "" {
			n := &VocabularyPhrase{}
			n.Content = []byte(phrase)
			n.SetParent(item)
			children = append(children, n)
		}

		if grammar != "" {
			n := &VocabularyGrammar{}
			n.Content = []byte(grammar)
			n.SetParent(item)
			children = append(children, n)
		}

		if transcription != "" {
			n := &VocabularyTranscription{}
			n.Content = []byte(transcription)
			n.SetParent(item)
			children = append(children, n)
		}

		if translation != "" {
			n := &VocabularyTranslation{}
			n.Content = []byte(translation)
			n.SetParent(item)
			children = append(children, n)
		}

		item.SetChildren(children)
		items = append(items, item)
	}
	res.SetChildren(items)

	return res, nil, end + len(endVocabulary)
}

func RenderVocabulary(w io.Writer, n *Vocabulary, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"vocabulary\">\n")
	} else {
		io.WriteString(w, "</div>\n")
	}
}

func RenderVocabularyItem(w io.Writer, n *VocabularyItem, entering bool) {
	if entering {
		io.WriteString(w, "<div class=\"vocabulary-item\">\n")
	} else {
		io.WriteString(w, "</div>\n")
	}
}

func RenderVocabularyPhrase(w io.Writer, n *VocabularyPhrase, entering bool) {
	if entering {
		io.WriteString(w, "<span class=\"vocabulary-phrase\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</span>\n")
	}
}

func RenderVocabularyGrammar(w io.Writer, n *VocabularyGrammar, entering bool) {
	if entering {
		io.WriteString(w, "<span class=\"vocabulary-grammar\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</span>\n")
	}
}

func RenderVocabularyTranscription(w io.Writer, n *VocabularyTranscription, entering bool) {
	if entering {
		io.WriteString(w, "<span class=\"vocabulary-transcription\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</span>\n")
	}
}

func RenderVocabularyTranslation(w io.Writer, n *VocabularyTranslation, entering bool) {
	if entering {
		io.WriteString(w, "<span class=\"vocabulary-translation\">")
		io.Writer.Write(w, n.Content)
		io.WriteString(w, "</span>\n")
	}
}
