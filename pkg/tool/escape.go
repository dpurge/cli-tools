package tool

import "strings"

// EscapeQuoted escapes s for embedding inside a string literal delimited by the
// given quote character (backslash, the quote, and newline/CR/tab); surrounding
// quotes are the caller's job. Shared by the Typst (pkg/ebook) and JS
// (pkg/scanbook) string writers, which differ only in quote character.
func EscapeQuoted(s string, quote rune) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case quote:
			b.WriteByte('\\')
			b.WriteRune(quote)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
