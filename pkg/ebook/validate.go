package ebook

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dpurge/cli-tools/pkg/catalog"
	"github.com/dpurge/cli-tools/pkg/tool/markdown"
	"gopkg.in/yaml.v3"
)

// ValidationError is one undeclared language/script/grammar-tag usage
// found while validating a project tree against pkg/catalog.
type ValidationError struct {
	File  string
	Line  int
	Kind  string // "language", "script", or "tag"
	Value string
}

// String formats e as "<file>:<line>: unknown <kind> "<value>"", the exact
// shape `ebook validate` prints per finding.
func (e ValidationError) String() string {
	return fmt.Sprintf("%s:%d: unknown %s %q", e.File, e.Line, e.Kind, e.Value)
}

// ValidateTree recursively finds every ebook.yml under root and validates
// each project's declared languages/scripts (top-level fields and every
// content block's lang=/script= attribute) and grammar tags
// ({start-vocabulary} fields) against pkg/catalog. Returns every finding,
// not just the first, so a single run surfaces the whole picture.
func ValidateTree(root string) ([]ValidationError, error) {
	var all []ValidationError
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "ebook.yml" {
			return nil
		}
		errs, err := validateProjectFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		all = append(all, errs...)
		return nil
	})
	return all, err
}

// projectYAML is a minimal, validate-only decode of ebook.yml's Text
// field — deliberately independent of readProject's full
// EBookProject/path-resolution pipeline (project.go), which resolves
// font/image/cover paths and would fail for reasons unrelated to
// language/script/tag declarations. A validator should still report
// findings even when an unrelated field in the file is broken.
type projectYAML struct {
	Text [][]string `yaml:"text"`
}

// languageFieldKind maps an ebook.yml top-level key to which catalog set
// (language or script) its value must be declared in.
var languageFieldKind = map[string]string{
	"language":             "language",
	"translation-language": "language",
	"script":               "script",
	"translation-script":   "script",
}

// validateProjectFile validates one ebook.yml: its top-level
// language/script/translation-language/translation-script fields, then
// every content file listed in its Text tree.
func validateProjectFile(path string) ([]ValidationError, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parsed via yaml.Node (not a plain struct) specifically to recover
	// each field's source Line — a plain yaml.Unmarshal into a Go struct
	// loses position information entirely, and a validator's whole value
	// is the file:line it reports.
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	var errs []ValidationError
	if len(doc.Content) > 0 {
		errs = append(errs, validateTopLevelFields(path, doc.Content[0])...)
	}

	var proj projectYAML
	if err := yaml.Unmarshal(raw, &proj); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	for _, group := range proj.Text {
		for _, file := range group {
			full := filepath.Join(dir, file)
			fileErrs, err := validateContentFile(full)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", full, err)
			}
			errs = append(errs, fileErrs...)
		}
	}

	return errs, nil
}

// validateTopLevelFields walks ebook.yml's top-level mapping node and
// checks language/script/translation-language/translation-script against
// pkg/catalog, reporting each unknown value at its own declared line.
func validateTopLevelFields(path string, mapNode *yaml.Node) []ValidationError {
	var errs []ValidationError
	if mapNode.Kind != yaml.MappingNode {
		return errs
	}
	for i := 0; i+1 < len(mapNode.Content); i += 2 {
		key := mapNode.Content[i]
		val := mapNode.Content[i+1]
		kind, ok := languageFieldKind[key.Value]
		if !ok || val.Value == "" {
			continue
		}
		if !isDeclared(kind, val.Value) {
			errs = append(errs, ValidationError{File: path, Line: val.Line, Kind: kind, Value: val.Value})
		}
	}
	return errs
}

// isDeclared reports whether value is a declared pkg/catalog entry for
// kind ("language" or "script").
func isDeclared(kind, value string) bool {
	switch kind {
	case "language":
		_, ok := catalog.LookupLanguage(value)
		return ok
	case "script":
		_, ok := catalog.LookupScript(value)
		return ok
	}
	return true // unknown kind: nothing declared to check against, don't flag
}

// startMarkerRe/endMarkerRe recognize a custom block's marker line — the
// same "{start-X ...}"/"{end-X}" shape parser.go's opensRawBlock matches,
// re-expressed as a standalone regex since validateContentFile scans lines
// directly rather than driving the full goldmark parser (see
// validateContentFile's doc comment for why).
var (
	startMarkerRe = regexp.MustCompile(`^\{start-([a-z][a-z-]*)`)
	endMarkerRe   = regexp.MustCompile(`^\{end-([a-z][a-z-]*)\}`)
)

// validateContentFile scans one markdown content file line-by-line,
// checking every custom block's lang=/script= marker attribute and every
// {start-vocabulary} grammar-field token against pkg/catalog. Errors report
// the exact 1-based source line of the offending marker/item.
//
// This deliberately does NOT drive the full AST parser (pkg/tool/markdown):
// that parser does not currently expose source line numbers for a block's
// marker or its items — teaching it to would mean threading position
// tracking through every one of the 8 custom block parsers, a much larger
// change than this validator needs. A dedicated line-by-line scan is
// simpler and sufficient here (accurate file:line attribution is what a
// validator needs, not full AST fidelity), and it still reuses the real
// parsing algorithms — markdown.ParseMarkerAttrs, markdown.ParseVocabularyItems
// — rather than duplicating them, so a marker/tag is judged exactly as the
// real parser would judge it.
func validateContentFile(path string) ([]ValidationError, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var errs []ValidationError
	var currentBlock string
	lineNo := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lineNo++
		trimmed := strings.TrimSpace(scanner.Text())

		if m := startMarkerRe.FindStringSubmatch(trimmed); m != nil {
			blockName := m[1]
			if attrs, err := markdown.ParseMarkerAttrs([]byte(trimmed), blockName); err == nil {
				if attrs.Lang != "" && !isDeclared("language", attrs.Lang) {
					errs = append(errs, ValidationError{File: path, Line: lineNo, Kind: "language", Value: attrs.Lang})
				}
				if attrs.Script != "" && !isDeclared("script", attrs.Script) {
					errs = append(errs, ValidationError{File: path, Line: lineNo, Kind: "script", Value: attrs.Script})
				}
			}
			currentBlock = blockName
			continue
		}
		if endMarkerRe.MatchString(trimmed) {
			currentBlock = ""
			continue
		}

		if currentBlock == "vocabulary" && trimmed != "" {
			errs = append(errs, checkVocabularyLineTags(path, lineNo, trimmed)...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return errs, nil
}

// checkVocabularyLineTags extracts one {start-vocabulary} line's Grammar
// field — via markdown.ParseVocabularyItems, called on this single line
// (see that function's doc comment for why that is behaviorally identical
// to calling it on the whole block, and how it gives the validator an
// accurate line number for free) — and checks each whitespace-split token
// against pkg/catalog's declared tags.
//
// Guards against ParseVocabularyItems's own documented panic (a malformed
// line whose translation split leaves an empty phrase) with recover():
// `ebook validate` is a best-effort reporting tool over arbitrary content,
// not a strict parser, so one malformed line reports nothing for itself
// rather than aborting the whole run.
func checkVocabularyLineTags(path string, lineNo int, line string) (errs []ValidationError) {
	defer func() { recover() }()

	for _, item := range markdown.ParseVocabularyItems(line) {
		if item.Grammar == "" {
			continue
		}
		for _, tok := range strings.Fields(item.Grammar) {
			if _, ok := catalog.LookupTag(tok); !ok {
				errs = append(errs, ValidationError{File: path, Line: lineNo, Kind: "tag", Value: tok})
			}
		}
	}
	return errs
}
