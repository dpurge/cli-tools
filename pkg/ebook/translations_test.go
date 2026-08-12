package ebook

// Unit tests for resolveContentsTitle (FR-7 AC-4).
// The function is unexported; this file is in package ebook (not ebook_test)
// so it has direct access.
//
// Coverage:
//   - AC-4a: explicit override wins over everything (catalog, empty lang)
//   - AC-4b: catalog hit for a catalogued language ("en")
//   - AC-4c: catalog miss for an unsupported language returns ""
//   - AC-4d: case-insensitivity of the language key (BCP-47 variants and
//     uppercase inputs resolve to the same catalog entry via typstLang +
//     strings.ToLower)

import "testing"

func TestResolveContentsTitle(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		lang     string
		want     string
	}{
		// AC-4a: explicit override always wins, regardless of lang or catalog.
		{
			name:     "explicit override wins over catalog hit",
			explicit: "Spis treści",
			lang:     "en",
			want:     "Spis treści",
		},
		{
			name:     "explicit override wins over catalog miss",
			explicit: "Table des matières",
			lang:     "xx-invented",
			want:     "Table des matières",
		},
		{
			name:     "explicit override wins even when empty-ish lang",
			explicit: "Inhaltsverzeichnis",
			lang:     "",
			want:     "Inhaltsverzeichnis",
		},

		// AC-4b: catalog hit — a supported language resolves its Contents string.
		{
			name:     "catalog hit: en → Contents",
			explicit: "",
			lang:     "en",
			want:     "Contents",
		},

		// AC-4c: catalog miss — an unsupported/unknown language returns "".
		{
			name:     "catalog miss: unknown lang returns empty string",
			explicit: "",
			lang:     "xx-invented",
			want:     "",
		},
		{
			name:     "catalog miss: empty lang returns empty string",
			explicit: "",
			lang:     "",
			want:     "",
		},

		// AC-4d: case-insensitivity — BCP-47 variants and uppercase all resolve.
		// typstLang("en-US") → "en"; strings.ToLower("en") → "en" → catalog hit.
		{
			name:     "case-insensitive: en-US resolves to en catalog entry",
			explicit: "",
			lang:     "en-US",
			want:     "Contents",
		},
		{
			name:     "case-insensitive: EN (uppercase) resolves to en catalog entry",
			explicit: "",
			lang:     "EN",
			want:     "Contents",
		},
		{
			name:     "case-insensitive: En mixed case resolves to en catalog entry",
			explicit: "",
			lang:     "En",
			want:     "Contents",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveContentsTitle(tc.explicit, tc.lang)
			if got != tc.want {
				t.Fatalf("resolveContentsTitle(%q, %q) = %q, want %q",
					tc.explicit, tc.lang, got, tc.want)
			}
		})
	}
}
