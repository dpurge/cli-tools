package tool

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp creates an empty file at path, failing the test on error.
func writeTemp(t *testing.T, path string) error {
	t.Helper()
	return os.WriteFile(path, []byte("x"), 0o644)
}

// TestResolvePathEmptyStaysEmpty guards the fix for the empty-path bug: an
// unset optional path ("") must resolve to "" (not to the directory), so the
// callers' `if path != ""` guards keep treating it as unset. Before the fix,
// filepath.Join(dir, "") == dir made ResolvePath return the directory, which
// then read as a bogus present cover/stylesheet.
func TestResolvePathEmptyStaysEmpty(t *testing.T) {
	dir := t.TempDir() // a real, existing directory — the trap the bug fell into

	// checkExists true: the old code returned dir (it exists); the fix returns "".
	got, err := ResolvePath(dir, "", true)
	if err != nil {
		t.Fatalf("ResolvePath(%q, \"\", true) error = %v", dir, err)
	}
	if got != "" {
		t.Errorf("ResolvePath(%q, \"\", true) = %q, want \"\"", dir, got)
	}

	// checkExists false must behave the same.
	if got, err := ResolvePath(dir, "", false); err != nil || got != "" {
		t.Errorf("ResolvePath(%q, \"\", false) = (%q, %v), want (\"\", nil)", dir, got, err)
	}
}

// TestResolvePathNonEmptyUnchanged confirms the empty-path guard does not
// disturb normal resolution: a real file resolves to its absolute path.
func TestResolvePathNonEmptyUnchanged(t *testing.T) {
	dir := t.TempDir()
	name := "cover.svg"
	if err := writeTemp(t, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}

	got, err := ResolvePath(dir, name, true)
	if err != nil {
		t.Fatalf("ResolvePath error = %v", err)
	}
	if want := filepath.Join(dir, name); got != want {
		t.Errorf("ResolvePath = %q, want %q", got, want)
	}
}

// TestResolvePathsSkipsEmpty confirms the list wrapper inherits the guard:
// an empty entry stays empty, a real entry resolves.
func TestResolvePathsSkipsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := writeTemp(t, filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	paths := []string{"", "a.md"}
	if err := ResolvePaths(dir, paths, true); err != nil {
		t.Fatalf("ResolvePaths error = %v", err)
	}
	if paths[0] != "" {
		t.Errorf("empty entry = %q, want \"\"", paths[0])
	}
	if want := filepath.Join(dir, "a.md"); paths[1] != want {
		t.Errorf("resolved entry = %q, want %q", paths[1], want)
	}
}
