package markdown

import "strings"

// mdxTextMetachars are the characters that must be neutralized when
// emitted into MDX prose/markdown context (heading text, paragraph text,
// list items, table cells, link labels, blockquote content, ...). Per
// SPECS §5.2:
//
//	\   {   }   <   `   *   _   [   ]
//
// `\ { } <` are build-breaking (a stray `{`/`<` can open an unterminated
// JSX expression/element and fail the whole MDX build; `\` must be escaped
// first so an escape this function inserts is never itself re-interpreted
// as escaping the NEXT metachar). The remaining five (backtick, *, _, [,
// ]) are CommonMark inline delimiters that could accidentally pair/open a
// span in re-parsed output;
// they are escaped unconditionally (over-escaping is harmless: `\*` still
// renders as a literal `*`). Deliberately NOT escaped here (SPECS §5.2):
// `. , ! ? ( ) : ; # + - = ~ > /` — safe mid-line; their only hazard is at
// the very start of a line (handled separately by escapeMdxLineStart) or
// link/image delimiter pairing (handled by only escaping `[`/`]`).
const mdxTextMetachars = "\\{}<`*_[]"

// escapeMdxText escapes s for emission into MDX prose (a Text/String
// node's rendered value, a link Title, an Image alt string, ...). This is
// a single left-to-right scan of the INPUT (mirrors escapeTypstMarkup,
// typst_escape.go): each input byte is inspected exactly once, so a
// backslash this function inserts for one metachar is never re-scanned
// and re-escaped when a later metachar is processed.
func escapeMdxText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte(mdxTextMetachars, c) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// escapeMdxLineStart escapes a leading block-marker run at the very start
// of s (SPECS §5.3): a leading `#`, `>`, `-`, `+`, `=`, `~`, or a run of
// ASCII digits immediately followed by `.` or `)` (an ordinal list
// marker). s is assumed to already have escapeMdxText applied: none of
// escapeMdxText's own escaped characters overlap this function's trigger
// set, so applying this afterward can never double-escape a backslash
// escapeMdxText inserted, and the backslash THIS function inserts is
// never re-scanned by escapeMdxText (mdx_render.go calls them in that
// fixed order).
//
// This exists because a Soft/Hard line break inside a Text node is
// re-emitted as a literal newline (D2, mdx_render.go's renderText), so
// authored prose that merely starts a NEW physical source line with, say,
// a literal "-" would otherwise land at the start of an emitted MDX line
// too and be misread as a list item / blockquote / heading / ordinal list
// marker when the emitted file is re-parsed downstream.
func escapeMdxLineStart(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '#', '>', '-', '+', '=', '~':
		return "\\" + s
	}
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) && (s[i] == '.' || s[i] == ')') {
		return s[:i] + "\\" + s[i:]
	}
	return s
}

// escapeMdxUrl escapes dest for emission as a Link/Image destination
// (SPECS §5.5, review-added defense-in-depth): a destination is
// structural, not prose, but a stray space or an unbalanced `(`/`)` would
// still break the enclosing `](...)` syntax. If dest contains a space,
// `<`, `>`, `(`, or `)`, it is wrapped in CommonMark's angle-bracket
// destination form `<...>`, with any `<`/`>` INSIDE the wrapped value
// backslash-escaped so it cannot prematurely close the bracket; otherwise
// dest is emitted as-is (the common case: no metachars, no wrapping).
func escapeMdxUrl(dest string) string {
	if !strings.ContainsAny(dest, " <>()") {
		return dest
	}
	var b strings.Builder
	b.Grow(len(dest) + 2)
	b.WriteByte('<')
	for i := 0; i < len(dest); i++ {
		c := dest[i]
		if c == '<' || c == '>' {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte('>')
	return b.String()
}

// escapeMdxAttr escapes s for emission inside a double-quoted MDX JSX
// attribute value (`<Text lang="..." script="...">`, SPECS §7.5) — the
// same minimal two-character escape SPECS §5.4 specifies for YAML
// frontmatter double-quoted scalars (mirrors typstStringLiteral,
// pkg/ebook/typst.go): only `\` and `"` are meaningful to escape in a
// double-quoted string/attribute context.
func escapeMdxAttr(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// longestBacktickRun returns the length of the longest consecutive run of
// backtick characters in s (0 if s contains no backtick). Shared by
// mdxFence (block fences, SPECS §5.1.3) and renderCodeSpan's inline
// delimiter widening (mdx_render.go) — both need the same "one more
// backtick than the longest run already present" rule so the delimiter
// can never be prematurely closed by the content it wraps.
func longestBacktickRun(s string) int {
	longest, current := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}

// mdxFence returns the backtick fence delimiter to wrap content in a
// fenced code block (a standard FencedCodeBlock re-emission, or one of
// the vocabulary/dialog/parallel custom-block fences, SPECS §5.1.3): its
// length is max(3, longestBacktickRun(content)+1), so a content line
// consisting of N backticks can never be misread as closing the fence.
func mdxFence(content string) string {
	n := longestBacktickRun(content) + 1
	if n < 3 {
		n = 3
	}
	return strings.Repeat("`", n)
}
