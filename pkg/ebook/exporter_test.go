package ebook

import "testing"

// TestLanguageInfoFR1PolMapping is the FR-1 AC-1 regression test:
// languageInfo("pol", ...) must return lang="pl" after the T1 fix.
//
// A small set of pre-existing cases are included as regression anchors so
// that a merge that accidentally shifts the switch order is caught here as
// well as in the comprehensive TestLanguageInfoLanguageMapping (typst_export_test.go).
// The table deliberately does NOT enumerate every case — only the new case,
// its alphabetic neighbours in the switch, one RTL anchor, and the two
// documented edge cases (default fallthrough, pre-existing heb quirk).
// TestBaseOutputName is the table-driven regression test for the shared
// base-name derivation helper (FR-1, FR-2 basis). Three cases cover every
// branch of baseOutputName:
//   - extensionless: no-op passthrough (FR-2 AC-1 basis)
//   - .epub-suffixed: suffix is stripped (FR-2 AC-2 basis)
//   - other-extension: stripped via generic filepath.Ext fallback (FR-1 AC-3)
func TestBaseOutputName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"extensionless", "/a/b/foo", "/a/b/foo"},
		{"epub-suffixed", "/a/b/foo.epub", "/a/b/foo"},
		{"other-extension", "/a/b/foo.txt", "/a/b/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baseOutputName(tt.input)
			if got != tt.want {
				t.Errorf("baseOutputName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLanguageInfoFR1PolMapping(t *testing.T) {
	tests := []struct {
		language string
		script   string
		wantLang string
		wantDir  string
	}{
		// FR-1 AC-1 — the fixed case (was "en" before T1):
		{"pol", "latn", "pl", "ltr"},

		// Alphabetic neighbours in the switch — guard against ordering mistakes:
		{"nld", "latn", "nl", "ltr"},
		{"ron", "latn", "ro", "ltr"},

		// One RTL case to confirm direction logic is intact:
		{"arb", "arab", "ar", "rtl"},

		// Default fallthrough — unknown language must still yield "en"/"ltr":
		{"xyz", "latn", "en", "ltr"},

		// Pre-existing heb quirk: "heb" is NOT in the switch (documented in
		// exporter.go), so it falls to "en". Direction comes from script "hebr"
		// → "rtl". This must stay as-is (out of scope, NFR-1).
		{"heb", "hebr", "en", "rtl"},
	}

	for _, tt := range tests {
		lang, dir := languageInfo(tt.language, tt.script)
		if lang != tt.wantLang {
			t.Errorf("languageInfo(%q, %q) lang = %q, want %q",
				tt.language, tt.script, lang, tt.wantLang)
		}
		if dir != tt.wantDir {
			t.Errorf("languageInfo(%q, %q) dir = %q, want %q",
				tt.language, tt.script, dir, tt.wantDir)
		}
	}
}
