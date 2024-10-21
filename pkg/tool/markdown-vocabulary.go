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
		io.WriteString(w, "<vocabulary>\n")
	} else {
		io.WriteString(w, "</vocabulary>\n")
	}
}

func RenderVocabularyItem(w io.Writer, n *VocabularyItem, entering bool) {
	if entering {
		io.WriteString(w, "<item>\n")
	} else {
		io.WriteString(w, "</item>\n")
	}
}

func RenderVocabularyPhrase(w io.Writer, n *VocabularyPhrase, entering bool) {
	if entering {
		io.WriteString(w, "<phrase>")
		io.WriteString(w, string(n.Content))
		io.WriteString(w, "</phrase>\n")
	}
}

func RenderVocabularyGrammar(w io.Writer, n *VocabularyGrammar, entering bool) {
	if entering {
		io.WriteString(w, "<grammar>")
		io.WriteString(w, string(n.Content))
		io.WriteString(w, "</grammar>\n")
	}
}

func RenderVocabularyTranscription(w io.Writer, n *VocabularyTranscription, entering bool) {
	if entering {
		io.WriteString(w, "<transcription>")
		io.WriteString(w, string(n.Content))
		io.WriteString(w, "</transcription>\n")
	}
}

func RenderVocabularyTranslation(w io.Writer, n *VocabularyTranslation, entering bool) {
	if entering {
		io.WriteString(w, "<translation>")
		io.WriteString(w, string(n.Content))
		io.WriteString(w, "</translation>\n")
	}
}
