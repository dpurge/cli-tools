package markdown

import (
	"bytes"
	"fmt"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Start/end markers for the custom blocks. Start markers intentionally
// omit the closing '}' so that attributed forms like
// {start-vocabulary lang=arb script=arab} are recognised by the HasPrefix
// check in opensRawBlock. End markers remain exact literals.
var (
	startVocabulary = []byte("{start-vocabulary")
	endVocabulary   = []byte("{end-vocabulary}")

	startDialog = []byte("{start-dialog")
	endDialog   = []byte("{end-dialog}")

	startParallel = []byte("{start-parallel")
	endParallel   = []byte("{end-parallel}")

	// startParallelDialog must be checked before/independent of startParallel
	// collision: opensRawBlock's boundary guard (below) already rejects
	// "{start-parallel-dialog...}" as a match for the shorter startParallel
	// prefix (the byte after "{start-parallel" is '-', not '}'/whitespace/EOL),
	// so registration order between the two block parsers does not matter.
	startParallelDialog = []byte("{start-parallel-dialog")
	endParallelDialog   = []byte("{end-parallel-dialog}")

	startModels = []byte("{start-models")
	endModels   = []byte("{end-models}")

	startQuestions = []byte("{start-questions")
	endQuestions   = []byte("{end-questions}")

	startText = []byte("{start-text")
	endText   = []byte("{end-text}")
)

// opensRawBlock reports whether the reader is positioned at a line that
// starts with start AND a matching end marker exists later in the source.
// When both conditions hold it consumes the start-marker line and returns
// (markerLine, true); markerLine is the raw line slice (including any
// trailing \r\n) valid for attribute parsing via parseMarkerAttrs (attr.go).
// When either condition fails it returns (nil, false) leaving the reader
// position unchanged so the line falls through to ordinary paragraph text.
//
// The start slice must NOT include the closing '}' (the start vars above
// omit it) so that attributed forms like {start-vocabulary lang=arb} are
// matched. To prevent spurious prefix collision with longer block names the
// byte immediately following the prefix must be '}', whitespace, or a line
// terminator — any other byte rejects the match.
//
// The end-marker presence check is parity-critical: an unterminated block
// must fall through rather than swallow the remainder of the document,
// mirroring the previous gomarkdown hook's `return nil, data, 0` fallback.
func opensRawBlock(reader text.Reader, start, end []byte) ([]byte, bool) {
	line, segment := reader.PeekLine()
	if !bytes.HasPrefix(line, start) {
		return nil, false
	}
	// Boundary guard: the byte immediately after the prefix must be '}',
	// whitespace, or a line terminator — not another name character.
	if rest := line[len(start):]; len(rest) > 0 &&
		rest[0] != '}' && rest[0] != ' ' && rest[0] != '\t' &&
		rest[0] != '\r' && rest[0] != '\n' {
		return nil, false
	}
	if !bytes.Contains(reader.Source()[segment.Stop:], end) {
		return nil, false
	}
	reader.AdvanceToEOL()
	return line, true
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
	markerLine, ok := opensRawBlock(reader, startVocabulary, endVocabulary)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Vocabulary{}
	attrs, err := parseMarkerAttrs(markerLine, "vocabulary")
	if err != nil {
		n.Err = err
	} else {
		// SPECS §5: vocabulary's field languages are fixed (phrase=foreign,
		// translation=base), so as= is rejected entirely, not just out-of-set
		// values — this avoids a no-op attribute that "exists but does nothing".
		if attrs.As != "" {
			n.Err = fmt.Errorf("as= not applicable to {start-vocabulary}: its field languages are fixed")
		}
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
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

		// Header recognition (SPECS §3.1/§5); no note support for vocabulary (D1).
		if level, text, ok := isBlockHeader(s); ok {
			items = append(items, VocabularyItem{BlockAnnotation: BlockAnnotation{Kind: ItemHeader, Level: level, Text: text}})
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
	markerLine, ok := opensRawBlock(reader, startDialog, endDialog)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Dialog{}
	attrs, err := parseMarkerAttrs(markerLine, "dialog")
	if err != nil {
		n.Err = err
	} else {
		// SPECS §5: dialog unifies onto as=source|translation (default
		// source) — accepted values are a STRICT SUBSET of the shared
		// attr.go grammar (source/transcription/translation/grammar), so an
		// in-grammar-but-not-accepted-here value (transcription/grammar)
		// needs its own precise, block-specific error naming dialog's set.
		as := attrs.As
		if as == "" {
			as = "source"
		}
		if as != "source" && as != "translation" {
			n.Err = fmt.Errorf("as=%q is not valid on {start-dialog}: must be source|translation", as)
		}
		n.As = as
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
}

func (b *dialogParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endDialog)
}

func (b *dialogParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Dialog)
	items, err := parseDialogItems(rawBlockText(node, reader))
	n.Items = items
	// Preserve an Open-time error (malformed attribute, or the new as=
	// validation, SPECS §5): Close previously always overwrote n.Err from
	// parseDialogItems, silently discarding it whenever the content itself
	// parsed without a bad-indentation error — a pre-existing bug this
	// increment's as= error-surfacing acceptance criterion depends on.
	if n.Err == nil {
		n.Err = err
	}
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

		// Block-level header/note recognition on un-indented lines (SPECS §3.3/§5).
		// Indented lines (starting with spaces) are preserved as turn content by
		// the existing " " prefix check below — isBlockHeader/isBlockNote never
		// match them because they require a leading '#' or '(' with no indent.
		// flush() the current turn buffer and reset header="" to prevent a stale
		// speaker label from producing a spurious empty DialogItem (SPECS §5 F6).
		if level, text, ok := isBlockHeader(s); ok {
			flush()
			items = append(items, DialogItem{BlockAnnotation: BlockAnnotation{Kind: ItemHeader, Level: level, Text: text}})
			header = ""
			continue
		}
		if text, ok := isBlockNote(s); ok {
			flush()
			items = append(items, DialogItem{BlockAnnotation: BlockAnnotation{Kind: ItemNote, Text: text}})
			header = ""
			continue
		}

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

// isBlockHeader reports whether line is a CommonMark ATX heading (1–6 '#'
// characters followed by a space/tab or end-of-string). Returns the heading
// level (1-6), trimmed text, and ok=true. Seven or more '#' → ok=false.
// Operates on the already-scope-trimmed line (per-block trimming rules,
// SPECS §3.1/§3.3).
func isBlockHeader(line string) (level int, text string, ok bool) {
	if len(line) == 0 || line[0] != '#' {
		return 0, "", false
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i > 6 {
		return 0, "", false
	}
	level = i
	if i == len(line) {
		// Bare "####" with no trailing text is a valid header with empty text.
		return level, "", true
	}
	if line[i] != ' ' && line[i] != '\t' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[i+1:]), true
}

// isBlockNote reports whether line is a parenthesized note: the entire line
// starts with '(' and ends with ')' (length ≥ 2). Returns the inner text
// (parentheses stripped, TrimSpaced) and ok=true (SPECS §3.2).
func isBlockNote(line string) (text string, ok bool) {
	if len(line) < 2 || line[0] != '(' || line[len(line)-1] != ')' {
		return "", false
	}
	return strings.TrimSpace(line[1 : len(line)-1]), true
}

// ---------------------------------------------------------------------
// Parallel
// ---------------------------------------------------------------------

type parallelParser struct{}

func newParallelParser() parser.BlockParser { return &parallelParser{} }

func (b *parallelParser) Trigger() []byte { return []byte{'{'} }

func (b *parallelParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	markerLine, ok := opensRawBlock(reader, startParallel, endParallel)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Parallel{}
	attrs, err := parseMarkerAttrs(markerLine, "parallel")
	if err != nil {
		n.Err = err
	} else {
		// SPECS §5: parallel's main/secondary columns already carry both
		// languages, so as= is rejected entirely, with a clearer message.
		if attrs.As != "" {
			n.Err = fmt.Errorf("as= not applicable to {start-parallel}: its field languages are fixed")
		}
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
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
// Rows are separated by a "===" line; within each row the record is split on
// every lone "---" line, capped at 3 fields (strings.SplitN(..., 3)):
//   - field 1 (SourceRaw): always present.
//   - field 2 (TranslationRaw): present when the record has ≥2 "---"-separated fields.
//   - field 3 (TranscriptionRaw): present when the record has exactly 3 "---"-separated fields.
//
// A record with 4+ "---" lines absorbs the excess into field 3 (the extra
// separators remain verbatim inside TranscriptionRaw). This retires the
// prior strings.LastIndex thematic-break-preservation trick: a "---" inside
// the source of a 2-field record now splits source from translation (ASR-3,
// SPECS §5.2).
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

		fields := strings.SplitN(s, "\n---\n", 3)
		var row ParallelRow
		row.SourceRaw = strings.TrimSpace(fields[0])
		if len(fields) >= 2 {
			row.TranslationRaw = strings.TrimSpace(fields[1])
		}
		if len(fields) == 3 {
			row.TranscriptionRaw = strings.TrimSpace(fields[2])
		}

		rows = append(rows, row)
	}

	return rows
}

// ---------------------------------------------------------------------
// ParallelDialog
// ---------------------------------------------------------------------

type parallelDialogParser struct{}

func newParallelDialogParser() parser.BlockParser { return &parallelDialogParser{} }

func (b *parallelDialogParser) Trigger() []byte { return []byte{'{'} }

func (b *parallelDialogParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	markerLine, ok := opensRawBlock(reader, startParallelDialog, endParallelDialog)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &ParallelDialog{}
	attrs, err := parseMarkerAttrs(markerLine, "parallel-dialog")
	if err != nil {
		n.Err = err
	} else {
		// as= is rejected entirely, mirroring Parallel: both columns'
		// languages are already fixed by the row/field position.
		if attrs.As != "" {
			n.Err = fmt.Errorf("as= not applicable to {start-parallel-dialog}: its field languages are fixed")
		}
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
}

func (b *parallelDialogParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endParallelDialog)
}

func (b *parallelDialogParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*ParallelDialog)
	rows, err := parseParallelDialogRows(rawBlockText(node, reader))
	n.Rows = rows
	// Preserve an Open-time error (malformed attribute, or the as=
	// rejection above), mirroring Dialog.Close's identical precedence rule
	// (an Open-time error must never be silently overwritten by a
	// content-parse result, even a nil one).
	if n.Err == nil {
		n.Err = err
	}
}

func (b *parallelDialogParser) CanInterruptParagraph() bool { return true }
func (b *parallelDialogParser) CanAcceptIndentedLine() bool { return false }

// parseParallelDialogRows parses the dedented `{start-parallel-dialog}` body
// into rows. Rows are separated by a "===" line; within each row the record
// is split on every lone "---" line, capped at 3 fields — source,
// translation, transcription — identical to parseParallelRows's row/field
// grammar. Unlike Parallel's raw fields, every present field here must
// parse as exactly one dialog turn or heading (parseParallelDialogField);
// translation is mandatory (a row with no "---" errors, unlike plain
// {start-parallel} where the translation field is optional prose).
func parseParallelDialogRows(inner string) ([]ParallelDialogRow, error) {
	var rows []ParallelDialogRow

	for _, chunk := range strings.Split(inner, "\n===\n") {
		s := strings.TrimSpace(chunk)
		if s == "" {
			continue
		}

		fields := strings.SplitN(s, "\n---\n", 3)
		if len(fields) < 2 {
			return nil, fmt.Errorf("parallel-dialog row is missing its translation field: %q", s)
		}

		var row ParallelDialogRow
		var err error
		if row.Source, err = parseParallelDialogField(fields[0]); err != nil {
			return nil, fmt.Errorf("parallel-dialog source field: %w", err)
		}
		if row.Translation, err = parseParallelDialogField(fields[1]); err != nil {
			return nil, fmt.Errorf("parallel-dialog translation field: %w", err)
		}
		if len(fields) == 3 {
			if row.Transcription, err = parseParallelDialogField(fields[2]); err != nil {
				return nil, fmt.Errorf("parallel-dialog transcription field: %w", err)
			}
			row.HasTranscription = true
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// parseParallelDialogField parses one already-"---"-delimited field into
// exactly one item — a dialog turn (Header+Content) or a heading — using
// the SAME line grammar as parseDialogItems (isBlockHeader/
// isDialogItemHeader/getDialogItemHeader, unchanged), just scoped to a
// single field instead of a whole block. Zero or more-than-one resulting
// item is an error: unlike a {start-dialog} block, a parallel-dialog field
// never holds a run of several turns.
func parseParallelDialogField(field string) (ParallelDialogItem, error) {
	var items []ParallelDialogItem
	var buf []string
	header := ""

	flush := func() {
		if len(buf) > 0 {
			items = append(items, ParallelDialogItem{
				Header:  header,
				Content: strings.TrimSpace(strings.Join(buf, "\n")),
			})
			buf = nil
		}
	}

	for _, line := range strings.Split(field, "\n") {
		s := strings.TrimRight(line, " *")

		if level, text, ok := isBlockHeader(s); ok {
			flush()
			items = append(items, ParallelDialogItem{BlockAnnotation: BlockAnnotation{Kind: ItemHeader, Level: level, Text: text}})
			header = ""
			continue
		}

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

		return ParallelDialogItem{}, fmt.Errorf("wrong line indentation for parallel-dialog item: %s", s)
	}
	flush()

	if len(items) != 1 {
		return ParallelDialogItem{}, fmt.Errorf("field must contain exactly one turn or heading, got %d item(s): %q", len(items), field)
	}
	return items[0], nil
}

// ---------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------

type modelsParser struct{}

func newModelsParser() parser.BlockParser { return &modelsParser{} }

func (b *modelsParser) Trigger() []byte { return []byte{'{'} }

func (b *modelsParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	markerLine, ok := opensRawBlock(reader, startModels, endModels)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Models{}
	attrs, err := parseMarkerAttrs(markerLine, "models")
	if err != nil {
		n.Err = err
	} else {
		// SPECS §5: models' field languages are fixed (like vocabulary), so
		// as= is rejected entirely, with a clearer message.
		if attrs.As != "" {
			n.Err = fmt.Errorf("as= not applicable to {start-models}: its field languages are fixed")
		}
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
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

		// Header then note recognition (SPECS §3.1/§3.2/§5).
		if level, text, ok := isBlockHeader(s); ok {
			items = append(items, ModelsItem{BlockAnnotation: BlockAnnotation{Kind: ItemHeader, Level: level, Text: text}})
			continue
		}
		if text, ok := isBlockNote(s); ok {
			items = append(items, ModelsItem{BlockAnnotation: BlockAnnotation{Kind: ItemNote, Text: text}})
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
	markerLine, ok := opensRawBlock(reader, startQuestions, endQuestions)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Questions{}
	attrs, err := parseMarkerAttrs(markerLine, "questions")
	if err != nil {
		n.Err = err
	} else {
		// SPECS §5: questions unifies onto as=source|translation (default
		// source), mirroring dialog — see dialogParser.Open's comment for
		// why this is a stricter subset of the shared attr.go grammar.
		as := attrs.As
		if as == "" {
			as = "source"
		}
		if as != "source" && as != "translation" {
			n.Err = fmt.Errorf("as=%q is not valid on {start-questions}: must be source|translation", as)
		}
		n.As = as
		n.Lang = attrs.Lang
		n.Script = attrs.Script
	}
	return n, parser.NoChildren
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

// ---------------------------------------------------------------------
// Text
// ---------------------------------------------------------------------

type textParser struct{}

func newTextParser() parser.BlockParser { return &textParser{} }

func (b *textParser) Trigger() []byte { return []byte{'{'} }

// Open parses the {start-text as=... lang=... script=...} marker and returns
// a Text node. as defaults to "source" when omitted (SPECS §4.2). System is
// parsed and stored for forward-compat but not yet used by any renderer
// (OI-9). An attribute-parse error is stored on Text.Err and surfaced at
// render time, mirroring Dialog.Err.
func (b *textParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	markerLine, ok := opensRawBlock(reader, startText, endText)
	if !ok {
		return nil, parser.NoChildren
	}
	n := &Text{}
	attrs, err := parseMarkerAttrs(markerLine, "text")
	if err != nil {
		n.Err = err
	} else {
		n.As = attrs.As
		if n.As == "" {
			n.As = "source"
		}
		n.Lang = attrs.Lang
		n.Script = attrs.Script
		n.System = attrs.System
	}
	return n, parser.NoChildren
}

func (b *textParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	return continueRawBlock(node, reader, endText)
}

// Close captures the raw inner markdown into Text.Raw so renderers can
// recurse through ToHTML/ToTypst/ToMDX at render time (mirrors
// Dialog/Parallel's rawBlockText pattern, SPECS §3.2).
func (b *textParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {
	n := node.(*Text)
	n.Raw = rawBlockText(node, reader)
}

func (b *textParser) CanInterruptParagraph() bool { return true }
func (b *textParser) CanAcceptIndentedLine() bool { return false }

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

		// Header then note recognition (SPECS §3.1/§3.2/§5).
		if level, text, ok := isBlockHeader(s); ok {
			items = append(items, QuestionItem{BlockAnnotation: BlockAnnotation{Kind: ItemHeader, Level: level, Text: text}})
			continue
		}
		if text, ok := isBlockNote(s); ok {
			items = append(items, QuestionItem{BlockAnnotation: BlockAnnotation{Kind: ItemNote, Text: text}})
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
