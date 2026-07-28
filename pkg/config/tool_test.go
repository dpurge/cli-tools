package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// writeExe creates an executable stub file and returns its path.
func writeExe(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	return p
}

// TestGetToolPathAbsolute: a configured path to an existing file is returned
// verbatim.
func TestGetToolPathAbsolute(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	exe := writeExe(t, t.TempDir(), "mytool")
	viper.Set("App.tool", exe)

	got, err := GetToolPath("App", "tool")
	if err != nil {
		t.Fatalf("GetToolPath error = %v", err)
	}
	if got != exe {
		t.Errorf("GetToolPath = %q, want %q", got, exe)
	}
}

// TestGetToolPathBareNameOnPath: a configured bare command name that is not a
// file resolves via PATH (the fix). This is the cfg/linux.yml case.
func TestGetToolPathBareNameOnPath(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	dir := t.TempDir()
	exe := writeExe(t, dir, "mytool")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	viper.Set("App.tool", "mytool") // bare name, not a stat-able path

	got, err := GetToolPath("App", "tool")
	if err != nil {
		t.Fatalf("GetToolPath error = %v", err)
	}
	if got != exe {
		t.Errorf("GetToolPath = %q, want resolved %q", got, exe)
	}
}

// TestGetToolPathMissing: a bare name that is neither a file nor on PATH errors.
func TestGetToolPathMissing(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	// An empty PATH so the bogus name cannot resolve anywhere.
	t.Setenv("PATH", t.TempDir())
	viper.Set("App.tool", "definitely-not-a-real-binary-xyz")

	if _, err := GetToolPath("App", "tool"); err == nil {
		t.Error("expected an error for a missing tool, got nil")
	}
}

// TestGetToolPathNotConfigured: an unset key errors before any lookup, so the
// no-config path (relied on by ebook's locateTypst fallback) is preserved.
func TestGetToolPathNotConfigured(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	if _, err := GetToolPath("App", "tool"); err == nil {
		t.Error("expected an error when the tool is not configured, got nil")
	}
}
