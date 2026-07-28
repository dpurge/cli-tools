package markdown_test

import (
	"strings"
	"testing"

	"github.com/dpurge/cli-tools/pkg/tool/markdown"
)

// TestCRLFDialog guards the CRLF fix: a dialog in a CRLF-encoded source must
// parse identically to its LF form. Before the fix the trailing "\r" left on
// each split line made the "--:" header fail to match, erroring the build.
func TestCRLFDialog(t *testing.T) {
	lf := "{start-dialog}\n--:\n  Hello there.\n{end-dialog}\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	convs := []struct {
		name string
		fn   func([]byte) ([]byte, error)
	}{
		{"ToHTML", markdown.ToHTML},
		{"ToTypst", markdown.ToTypst},
	}
	for _, c := range convs {
		lfOut, err := c.fn([]byte(lf))
		if err != nil {
			t.Fatalf("%s LF error: %v", c.name, err)
		}
		crlfOut, err := c.fn([]byte(crlf))
		if err != nil {
			t.Fatalf("%s CRLF error: %v", c.name, err)
		}
		if string(lfOut) != string(crlfOut) {
			t.Errorf("%s CRLF output differs from LF:\n LF=%q\n CRLF=%q", c.name, lfOut, crlfOut)
		}
	}
}
