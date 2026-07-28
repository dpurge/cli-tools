package markdown_test

import (
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestToHTML_Idempotent guards against shared-state bugs in the
// package-level singleton converter (converter.go's var md): converting the
// same input twice, including a document that recursively re-invokes
// ToHTML for dialog/parallel cell content, must yield byte-identical
// output both times.
func TestToHTML_Idempotent(t *testing.T) {
	inputs := []string{
		"# Title\n\nSome **bold** paragraph with a [link](https://x.com) and ~~strike~~.\n",
		"{start-vocabulary}\n你好 {noun} [nǐ hǎo] = hello\n{end-vocabulary}\n",
		"{start-dialog}\n--:\n  Hello there.\n@Bob:\n  Hi!\n{end-dialog}\n",
		"{start-parallel}\nFirst para.\n\n---\n\nSecond para in main.\n---\nSecondary cell.\n{end-parallel}\n",
	}

	for _, input := range inputs {
		first, err := markdown.ToHTML([]byte(input))
		if err != nil {
			t.Fatalf("ToHTML() first call unexpected error: %v", err)
		}
		second, err := markdown.ToHTML([]byte(input))
		if err != nil {
			t.Fatalf("ToHTML() second call unexpected error: %v", err)
		}
		if string(first) != string(second) {
			t.Fatalf("ToHTML() not idempotent for input %q:\n1st: %q\n2nd: %q", input, string(first), string(second))
		}
	}
}
