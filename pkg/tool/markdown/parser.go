package markdown

import (
	"bytes"
	"fmt"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Start/end markers for the three active custom blocks.
var (
	startVocabulary = []byte("{start-vocabulary}")
	endVocabulary   = []byte("{end-vocabulary}")

	startDialog = []byte("{start-dialog}")
	endDialog   = []byte("{end-dialog}")

	startParallel = []byte("{start-parallel}")
	endParallel   = []byte("{end-parallel}")

	startModels = []byte("{start-models}")
	endModels   = []byte("{end-models}")

	startQuestions = []byte("{start-questions}")
	endQuestions   = []byte("{end-questions}")
)

// opensRawBlock reports whether the reader is positioned at a line that
// starts with start AND a matching end marker exists later in the source.
//
// The second condition is parity-critical: if a block is never terminated,
// the `{start-X}` line must fall through to ordinary paragraph text instead
// of swallowing the remainder of the document (this mirrors the previous
// gomarkdown parser hook's `return nil, data, 0` fallback). Only when both
// conditions hold is the start-marker line consumed.
func opensRawBlock(reader text.Reader, start, end []byte) bool {
	line, segment := reader.PeekLine()
	if !bytes.HasPrefix(line, start) {
		return false
	}
	if !bytes.Contains(reader.Source()[segment.Stop:], end) {
		return false
	}
	reader.AdvanceToEOL()
	return true
}

// continueRawBlock accumulates the current line into node's body lines
// unless it is the end-marker line, in which case the block closes. Shared
// by all three block parsers below.
func continueRawBlock(node gast.Node, reader text.Reader, end []byte) parser.State {
	line, segment := reader.PeekLine()
	if bytes.HasPrefix(line, end) {
		reader.AdvanceToEOL()
		return parser.Close
	}
	node.Lines().Append(segment)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

// rawBlockText returns node's accumulated body lines, trimmed, ready for
// structural parsing.
func rawBlockText(node gast.Node, reader text.Reader) string {
	return strings.TrimSpace(string(node.Lines().Value(reader.Source())))
}

// ---------------------------------------------------------------------
// Vocabulary
// ---------------------------------------------------------------------

type vocabularyParser struct{}

func newVocabularyParser() parser.BlockParser { return &vocabularyParser{} }

func (b *vocabularyParser) Trigger() []byte { return []byte{'{'} }

func (b *vocabularyParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if !opensRawBlock(reader, startVocabulary, endVocabulary) {
		return nil, parser.NoChildren
	}
	return &Vocabulary{}, parser.NoChildren
}

func (b *vocabularyParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endVocabulary)
}

func (b *vocabularyParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Vocabulary)
	n.Items = parseVocabularyItems(rawBlockText(node, reader))
}

func (b *vocabularyParser) CanInterruptParagraph() bool { return true }
func (b *vocabularyParser) CanAcceptIndentedLine() bool { return false }

// parseVocabularyItems parses the dedented `{start-vocabulary}` body into
// items. Each non-empty line is parsed tail-to-head, in this order:
// trailing `= translation`, then trailing `[transcription]`, then trailing
// `{grammar}`; whatever remains is the phrase.
//
// NOTE: a line that becomes the empty string after the `=` split makes the
// trailing-`]`/`}` checks below index s[len(s)-1:] on an empty string,
// which panics. This mirrors a pre-existing bug in the ported gomarkdown
// code (markdown-vocabulary.go) and is deliberately left as-is per the
// approved migration plan.
func parseVocabularyItems(inner string) []VocabularyItem {
	var items []VocabularyItem

	for _, line := range strings.Split(inner, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}

		var item VocabularyItem

		if i := strings.LastIndex(s, "="); i != -1 {
			item.Translation = strings.TrimSpace(s[i+1:])
			s = strings.TrimSpace(s[:i])
		}
		if s[len(s)-1:] == "]" {
			i := strings.LastIndex(s, "[")
			item.Transcription = strings.TrimSpace(s[i+1 : len(s)-1])
			s = strings.TrimSpace(s[:i])
		}
		if s[len(s)-1:] == "}" {
			i := strings.LastIndex(s, "{")
			item.Grammar = strings.TrimSpace(s[i+1 : len(s)-1])
			s = strings.TrimSpace(s[:i])
		}
		item.Phrase = s

		items = append(items, item)
	}

	return items
}

// ---------------------------------------------------------------------
// Dialog
// ---------------------------------------------------------------------

type dialogParser struct{}

func newDialogParser() parser.BlockParser { return &dialogParser{} }

func (b *dialogParser) Trigger() []byte { return []byte{'{'} }

func (b *dialogParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if !opensRawBlock(reader, startDialog, endDialog) {
		return nil, parser.NoChildren
	}
	return &Dialog{}, parser.NoChildren
}

func (b *dialogParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endDialog)
}

func (b *dialogParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Dialog)
	n.Items, n.Err = parseDialogItems(rawBlockText(node, reader))
}

func (b *dialogParser) CanInterruptParagraph() bool { return true }
func (b *dialogParser) CanAcceptIndentedLine() bool { return false }

// parseDialogItems parses the dedented `{start-dialog}` body into items.
//
// D3: the ported gomarkdown code called log.Fatal on a badly indented
// content line, killing the whole process. Here an error is returned
// instead (and parsing of the remaining lines stops, mirroring the
// original's hard stop as closely as an error-based design allows); the
// Dialog NodeRenderer surfaces it out of ToHTML instead of exiting the
// program.
func parseDialogItems(inner string) ([]DialogItem, error) {
	var items []DialogItem
	var buf []string
	header := ""

	flush := func() {
		if len(buf) > 0 {
			items = append(items, DialogItem{
				Header:  header,
				Content: strings.TrimSpace(strings.Join(buf, "\n")),
			})
			buf = nil
		}
	}

	for _, line := range strings.Split(inner, "\n") {
		s := strings.TrimRight(line, " *")

		if isDialogItemHeader(s) {
			flush()
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

		return items, fmt.Errorf("Wrong line indentation for dialog item: %s", s)
	}
	flush()

	return items, nil
}

// isDialogItemHeader reports whether a (already `TrimRight(line, " *")`-ed)
// line is a dialog header: either exactly "--:", or it starts with "@"/"＠"
// and ends with ":"/"︰"/"：".
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

// getDialogItemHeader returns the display text for a header line: "--:"
// becomes "—"; otherwise only a leading "@"/"＠" is stripped — the trailing
// colon is kept (e.g. "@Bob:" -> "Bob:").
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

// ---------------------------------------------------------------------
// Parallel
// ---------------------------------------------------------------------

type parallelParser struct{}

func newParallelParser() parser.BlockParser { return &parallelParser{} }

func (b *parallelParser) Trigger() []byte { return []byte{'{'} }

func (b *parallelParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if !opensRawBlock(reader, startParallel, endParallel) {
		return nil, parser.NoChildren
	}
	return &Parallel{}, parser.NoChildren
}

func (b *parallelParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endParallel)
}

func (b *parallelParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Parallel)
	n.Rows = parseParallelRows(rawBlockText(node, reader))
}

func (b *parallelParser) CanInterruptParagraph() bool { return true }
func (b *parallelParser) CanAcceptIndentedLine() bool { return false }

// parseParallelRows parses the dedented `{start-parallel}` body into rows.
// Rows are separated by a "===" line; within a row, an optional secondary
// cell is split off at the LAST "---" line (`strings.LastIndex`, not the
// first), so a `---` thematic break inside the main cell's own markdown is
// preserved.
//
// D2: unlike the original gomarkdown port, this design never slices the
// block by a marker's byte length (the block parser's Open/Continue consume
// the block line-by-line instead, see opensRawBlock/continueRawBlock
// above), so the old "used vocabulary's marker length instead of
// parallel's" bug class cannot recur here.
func parseParallelRows(inner string) []ParallelRow {
	var rows []ParallelRow

	for _, chunk := range strings.Split(inner, "\n===\n") {
		s := strings.TrimSpace(chunk)
		if s == "" {
			continue
		}

		var row ParallelRow
		if i := strings.LastIndex(s, "\n---\n"); i != -1 {
			row.SecondaryRaw = strings.TrimSpace(s[i+len("\n---\n"):])
			s = strings.TrimSpace(s[:i])
		}
		row.MainRaw = s

		rows = append(rows, row)
	}

	return rows
}

// ---------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------

type modelsParser struct{}

func newModelsParser() parser.BlockParser { return &modelsParser{} }

func (b *modelsParser) Trigger() []byte { return []byte{'{'} }

func (b *modelsParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if !opensRawBlock(reader, startModels, endModels) {
		return nil, parser.NoChildren
	}
	return &Models{}, parser.NoChildren
}

func (b *modelsParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endModels)
}

func (b *modelsParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Models)
	n.Items = parseModelsItems(rawBlockText(node, reader))
}

func (b *modelsParser) CanInterruptParagraph() bool { return true }
func (b *modelsParser) CanAcceptIndentedLine() bool { return false }

// parseModelsItems parses the dedented `{start-models}` body into items:
// vocabulary's tail-to-head algorithm minus the `{grammar}` step. Unlike
// vocabulary, the trailing `= translation` is split off at the FIRST
// " = " (space-delimited) occurrence rather than the last bare "=" — a
// model's translation is prose and may itself contain "=" (agreed
// migration decision, mirroring parseQuestionsItems below for the same
// reason).
//
// Guard (agreed migration decision, deliberately diverging from
// vocabulary's preserved empty-phrase panic): if the `=` split leaves s
// empty, the trailing-`]` check is skipped instead of indexing s[len(s)-1:]
// on an empty string, and the `[` lookup is skipped entirely when no
// matching `[` is found, so a malformed line never panics.
func parseModelsItems(inner string) []ModelsItem {
	var items []ModelsItem

	for _, line := range strings.Split(inner, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}

		var item ModelsItem

		if i := strings.Index(s, " = "); i != -1 {
			item.Translation = strings.TrimSpace(s[i+len(" = "):])
			s = strings.TrimSpace(s[:i])
		}
		if s != "" && s[len(s)-1:] == "]" {
			if i := strings.LastIndex(s, "["); i != -1 {
				item.Transcription = strings.TrimSpace(s[i+1 : len(s)-1])
				s = strings.TrimSpace(s[:i])
			}
		}
		item.Phrase = s

		items = append(items, item)
	}

	return items
}

// ---------------------------------------------------------------------
// Questions
// ---------------------------------------------------------------------

type questionsParser struct{}

func newQuestionsParser() parser.BlockParser { return &questionsParser{} }

func (b *questionsParser) Trigger() []byte { return []byte{'{'} }

func (b *questionsParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	if !opensRawBlock(reader, startQuestions, endQuestions) {
		return nil, parser.NoChildren
	}
	return &Questions{}, parser.NoChildren
}

func (b *questionsParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endQuestions)
}

func (b *questionsParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Questions)
	n.Items = parseQuestionsItems(rawBlockText(node, reader))
}

func (b *questionsParser) CanInterruptParagraph() bool { return true }
func (b *questionsParser) CanAcceptIndentedLine() bool { return false }

// parseQuestionsItems splits each dedented `{start-questions}` line at the
// FIRST " = " (space-delimited) occurrence into question/answer: an answer
// is prose and may itself contain "=", so splitting at the LAST occurrence
// (vocabulary's convention) would mis-split it. A line with no " = " is a
// question-only line (Answer stays "").
func parseQuestionsItems(inner string) []QuestionItem {
	var items []QuestionItem

	for _, line := range strings.Split(inner, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}

		var item QuestionItem
		if i := strings.Index(s, " = "); i != -1 {
			item.Question = strings.TrimSpace(s[:i])
			item.Answer = strings.TrimSpace(s[i+len(" = "):])
		} else {
			item.Question = s
		}

		items = append(items, item)
	}

	return items
}
