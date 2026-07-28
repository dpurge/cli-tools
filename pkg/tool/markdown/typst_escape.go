package markdown

import (
	"strings"

	"github.com/yuin/goldmark/util"
)

// typstMarkupMetachars are the characters that must be neutralized when
// emitted into Typst markup/content context (paragraph text, headings,
// list items, table cells, link labels, ...). Per SPECS §5.2, `.` IS
// included: an escaped ordinal such as `1\. text`, once unescaped back to
// plain `1. text` (see unescapeMarkdownBackslash below) and emitted at the
// start of a Typst line, is parsed by Typst as an enum marker
// (empirically reproduced against Typst 0.15.1: see the line-start golden
// test in typst_test.go). Escaping `.` — and `-`, `/`, `=`, `+` likewise,
// for the same line-start-marker reason — is harmless everywhere else
// (over-escaping proven safe: `one\. two` renders as "one. two", `a\-b`
// renders as "a-b"), so it is applied unconditionally rather than only at
// detected line starts.
const typstMarkupMetachars = "\\#[]*_`$<>@~=+-/."

// unescapeMarkdownBackslash removes the backslash from a CommonMark
// backslash-escape sequence (`\` followed by ASCII punctuation) in s.
//
// GOLDMARK API DISCREPANCY vs SPECS §5.2 (verified against the installed
// github.com/yuin/goldmark@v1.8.4): SPECS assumed the AST's ast.Text.Value
// already has the escaping backslash removed for an escaped ordinal like
// `1\. text` ("decodes in the AST to the literal Text `1. text` with no
// backslash"). Direct inspection of the parsed AST
// (parser/parser.go:1246, the parseBlock scan) shows this is NOT so: the
// Text segment's byte range covers the SOURCE bytes verbatim, backslash
// included (confirmed empirically: Value(source) for "1\. text" is the
// 8-byte string "1\. text", not "1. text"). goldmark's own HTML path only
// removes the backslash at RENDER time, inside
// renderer/html/html.go:845 defaultWriter.Write (tracks an `escaped`
// bool per byte; on a punctuation byte following a backslash, it writes
// up to but excluding the backslash, then resumes writing from the
// punctuation byte itself — otherwise the backslash is literal text,
// e.g. `\a` keeps its backslash since 'a' is not ASCII punctuation).
// Since the Typst renderer's Text handling (typst_render.go) never
// touches goldmark's HTML writer, it must replicate that exact unescape
// pass itself before applying escapeTypstMarkup, or a source escape like
// `1\. text` would be double-handled (the source's own backslash PLUS
// Typst's metachar-escape backslash for the period), producing a
// visibly wrong extra backslash in the rendered PDF. util.IsPunct is the
// same ASCII-punctuation table goldmark's own writer consults, so the
// escapable character set matches exactly.
func unescapeMarkdownBackslash(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' && i+1 < len(s) && util.IsPunct(s[i+1]) {
			escaped = true
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// escapeTypstMarkup escapes s for emission into Typst markup/content
// context. This is a single left-to-right scan of the INPUT, so the
// "escape backslash first" rule from SPECS §5.2 is satisfied structurally:
// each input byte is inspected exactly once, so a backslash this function
// inserts for one metachar is never re-scanned and re-escaped when a later
// metachar is processed (unlike a naive sequence of strings.ReplaceAll
// calls, where processing order would matter and get this wrong).
func escapeTypstMarkup(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(typstMarkupMetachars, c) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// escapeTypstString escapes s for emission inside a Typst `"..."` string
// literal (vocabulary fields, dialog header, link/image destinations,
// #raw content, code/font paths). Per SPECS §5.1.3, string context is NOT
// markup context: only `\` and `"` are meaningful escapes there, plus the
// control characters that cannot appear literally inside a one-line Typst
// string literal (newline, tab, carriage return). Markup metachars such as
// `/` or `.` must NOT be escaped here — a stray `\/` inside a Typst string
// prints a literal backslash (empirically observed), which is exactly the
// class of bug this separate function exists to avoid (SPECS §5.1.3).
func escapeTypstString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// typographerEntities maps the HTML entity strings that the Typographer
// extension's inline parser can produce (extension/typographer.go's
// newDefaultSubstitutions, the only source of ast.String nodes under this
// package's parser configuration) to their Unicode codepoints. There are
// 9 unique values: Apostrophe and RightSingleQuote both default to
// "&rsquo;" (SPECS §4/§5.2), so the 10 TypographicPunctuation constants
// collapse to 9 map entries. None of these runes are Typst metachars, so
// they are emitted directly without escaping.
var typographerEntities = map[string]string{
	"&lsquo;":  "‘", // ‘
	"&rsquo;":  "’", // ’ (also Apostrophe's default)
	"&ldquo;":  "“", // “
	"&rdquo;":  "”", // ”
	"&ndash;":  "–", // –
	"&mdash;":  "—", // —
	"&hellip;": "…", // …
	"&laquo;":  "«", // «
	"&raquo;":  "»", // »
}
