package tool

import "testing"

// TestEscapeQuoted covers both quote contexts the helper serves: the double
// quote (Typst string literals) and the single quote (JS string literals).
// The opposite quote must pass through unescaped in each context.
func TestEscapeQuoted(t *testing.T) {
	doubleQuote := []struct{ in, want string }{
		{"plain", "plain"},
		{"", ""},
		{`he said "hi"`, `he said \"hi\"`},
		{`a\b`, `a\\b`},
		{"a\nb", `a\nb`},
		{"a\tb", `a\tb`},
		{"a\rb", `a\rb`},
		{"a'b", "a'b"}, // single quote not a metacharacter here
		{"café 你好", "café 你好"},
	}
	for _, tc := range doubleQuote {
		if got := EscapeQuoted(tc.in, '"'); got != tc.want {
			t.Errorf(`EscapeQuoted(%q, '"') = %q, want %q`, tc.in, got, tc.want)
		}
	}

	singleQuote := []struct{ in, want string }{
		{`a'b`, `a\'b`},
		{`a\b`, `a\\b`},
		{`a"b`, `a"b`}, // double quote not a metacharacter here
		{"a\nb", `a\nb`},
	}
	for _, tc := range singleQuote {
		if got := EscapeQuoted(tc.in, '\''); got != tc.want {
			t.Errorf("EscapeQuoted(%q, '\\'') = %q, want %q", tc.in, got, tc.want)
		}
	}
}
