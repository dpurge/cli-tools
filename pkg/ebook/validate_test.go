package ebook

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a small test helper: creates dir if needed and writes
// content to path.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

// TestValidateTree_CleanProject_NoErrors is the happy path: every declared
// language/script/tag is known, so ValidateTree reports nothing.
func TestValidateTree_CleanProject_NoErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	writeFile(t, filepath.Join(dir, "a.md"), "{start-vocabulary lang=spa script=latn}\ncasa {N f} = house\n{end-vocabulary}\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ValidateTree() = %v, want no errors", errs)
	}
}

// TestValidateTree_UnknownTopLevelLanguage_ReportsFileAndLine checks the
// top-level `language:` field is validated, with the CORRECT source line
// (yaml.Node line tracking, not just "line 1").
func TestValidateTree_UnknownTopLevelLanguage_ReportsFileAndLine(t *testing.T) {
	dir := t.TempDir()
	ebookPath := filepath.Join(dir, "ebook.yml")
	writeFile(t, ebookPath, "title: T\nlanguage: xyz\nscript: latn\ntext: []\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("ValidateTree() = %v, want exactly 1 error", errs)
	}
	got := errs[0]
	if got.File != ebookPath || got.Kind != "language" || got.Value != "xyz" || got.Line != 2 {
		t.Errorf("ValidateTree() = %+v, want {File:%q Line:2 Kind:language Value:xyz}", got, ebookPath)
	}
}

// TestValidateTree_UnknownScriptAttribute_InContentBlock checks a
// per-block script= attribute is validated, with the marker line's own
// line number (not the file's first line).
func TestValidateTree_UnknownScriptAttribute_InContentBlock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	mdPath := filepath.Join(dir, "a.md")
	writeFile(t, mdPath, "Some prose.\n\n{start-vocabulary lang=spa script=zzzz}\ncasa {N f} = house\n{end-vocabulary}\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("ValidateTree() = %v, want exactly 1 error", errs)
	}
	got := errs[0]
	if got.File != mdPath || got.Kind != "script" || got.Value != "zzzz" || got.Line != 3 {
		t.Errorf("ValidateTree() = %+v, want {File:%q Line:3 Kind:script Value:zzzz}", got, mdPath)
	}
}

// TestValidateTree_UnknownGrammarTag_ReportsEachToken checks a
// {start-vocabulary} grammar field's tokens are validated independently —
// one undeclared token among several declared ones is still caught, with
// the vocabulary line's own line number.
func TestValidateTree_UnknownGrammarTag_ReportsEachToken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	mdPath := filepath.Join(dir, "a.md")
	writeFile(t, mdPath, "{start-vocabulary}\nfoo {N zz} = bar\n{end-vocabulary}\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("ValidateTree() = %v, want exactly 1 error", errs)
	}
	got := errs[0]
	if got.File != mdPath || got.Kind != "tag" || got.Value != "zz" || got.Line != 2 {
		t.Errorf("ValidateTree() = %+v, want {File:%q Line:2 Kind:tag Value:zz}", got, mdPath)
	}
}

// TestValidateTree_ClassifierCharacterTag_Declared confirms a Chinese
// classifier character tag (e.g. "个") is recognized, and the old
// spelled-out "cl"/"ge" form is NOT.
func TestValidateTree_ClassifierCharacterTag_Declared(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	mdPath := filepath.Join(dir, "a.md")
	writeFile(t, mdPath, "{start-vocabulary lang=cmn script=hans}\n项目 {N 个} [xiangmu] = project\n{end-vocabulary}\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ValidateTree() = %v, want no errors for the declared classifier form", errs)
	}
}

// TestValidateTree_OutsideVocabularyBlock_TagsNotChecked confirms a bare
// "{...}" that happens to appear OUTSIDE a {start-vocabulary} block (e.g.
// inside a fenced code block, mirroring the rust project's println!
// format strings) is never mistaken for a grammar tag.
func TestValidateTree_OutsideVocabularyBlock_TagsNotChecked(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	writeFile(t, filepath.Join(dir, "a.md"), "```rust\nprintln!(\"{value}\");\n```\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("ValidateTree() = %v, want no errors (code fence content is not a vocabulary block)", errs)
	}
}

// TestValidateTree_TranslationLanguageField_Validated checks
// translation-language/translation-script are validated the same way as
// language/script (spa/ebook.yml's real shape).
func TestValidateTree_TranslationLanguageField_Validated(t *testing.T) {
	dir := t.TempDir()
	ebookPath := filepath.Join(dir, "ebook.yml")
	writeFile(t, ebookPath, "language: spa\nscript: latn\ntranslation-language: qqq\ntranslation-script: latn\ntext: []\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 1 || errs[0].Kind != "language" || errs[0].Value != "qqq" {
		t.Errorf("ValidateTree() = %v, want exactly 1 language error for %q", errs, "qqq")
	}
}

// TestValidateTree_MalformedVocabularyLine_DoesNotCrash guards
// checkVocabularyLineTags's recover(): ParseVocabularyItems has a
// documented pre-existing panic for a line that reduces to an empty
// phrase after the translation split (e.g. "= foo"). The validator must
// keep running, not crash the whole tree walk.
func TestValidateTree_MalformedVocabularyLine_DoesNotCrash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ebook.yml"), "language: pol\nscript: latn\ntext:\n- - a.md\n")
	writeFile(t, filepath.Join(dir, "a.md"), "{start-vocabulary}\n= foo\n{end-vocabulary}\n")

	if _, err := ValidateTree(dir); err != nil {
		t.Fatalf("ValidateTree() error = %v, want no error (malformed line should be swallowed, not crash)", err)
	}
}

// TestValidateTree_MultipleProjects_AllScanned confirms ValidateTree finds
// every ebook.yml under root, not just the first.
func TestValidateTree_MultipleProjects_AllScanned(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "proj1", "ebook.yml"), "language: xxx\nscript: latn\ntext: []\n")
	writeFile(t, filepath.Join(dir, "proj2", "ebook.yml"), "language: yyy\nscript: latn\ntext: []\n")

	errs, err := ValidateTree(dir)
	if err != nil {
		t.Fatalf("ValidateTree() error = %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("ValidateTree() = %v, want 2 errors (one per project)", errs)
	}
}

// TestValidationError_String pins the exact printed format.
func TestValidationError_String(t *testing.T) {
	e := ValidationError{File: "a.md", Line: 7, Kind: "tag", Value: "zz"}
	want := `a.md:7: unknown tag "zz"`
	if got := e.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
