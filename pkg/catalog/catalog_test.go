package catalog_test

import (
	"testing"

	"github.com/dpurge/cli-tools/pkg/catalog"
)

// TestLookupLanguage_KnownAndUnknown covers a plain lookup, the
// case/whitespace normalization every existing scattered set already
// applied, and the ok=false unknown case.
func TestLookupLanguage_KnownAndUnknown(t *testing.T) {
	tests := []struct {
		iso3     string
		wantISO1 string
		wantName string
		wantOK   bool
	}{
		{"spa", "es", "Spanish", true},
		{"SPA", "es", "Spanish", true},  // case-insensitive
		{" pol ", "pl", "Polish", true}, // whitespace-tolerant
		{"heb", "he", "Hebrew", true},   // the disclosed bug fix
		{"xyz", "", "", false},
	}
	for _, tt := range tests {
		got, ok := catalog.LookupLanguage(tt.iso3)
		if ok != tt.wantOK {
			t.Fatalf("LookupLanguage(%q) ok = %v, want %v", tt.iso3, ok, tt.wantOK)
		}
		if !ok {
			continue
		}
		if got.ISO1 != tt.wantISO1 || got.Name != tt.wantName {
			t.Errorf("LookupLanguage(%q) = %+v, want iso1=%q name=%q", tt.iso3, got, tt.wantISO1, tt.wantName)
		}
	}
}

// TestLookupScript_KnownAndUnknown mirrors TestLookupLanguage_KnownAndUnknown
// for scripts, including the syrc RTL fix.
func TestLookupScript_KnownAndUnknown(t *testing.T) {
	tests := []struct {
		iso4         string
		wantEnlarged bool
		wantDir      string
		wantOK       bool
	}{
		{"latn", false, "ltr", true},
		{"cyrl", false, "ltr", true},
		{"arab", true, "rtl", true},
		{"hebr", true, "rtl", true},
		{"syrc", true, "rtl", true}, // the disclosed inconsistency fix
		{"hant", true, "ltr", true}, // enlarged but LTR — orthogonal fields
		{"XYZ", false, "", false},
	}
	for _, tt := range tests {
		got, ok := catalog.LookupScript(tt.iso4)
		if ok != tt.wantOK {
			t.Fatalf("LookupScript(%q) ok = %v, want %v", tt.iso4, ok, tt.wantOK)
		}
		if !ok {
			continue
		}
		if got.Enlarged != tt.wantEnlarged || got.Direction != tt.wantDir {
			t.Errorf("LookupScript(%q) = %+v, want enlarged=%v dir=%q", tt.iso4, got, tt.wantEnlarged, tt.wantDir)
		}
	}
}

// TestLookupTag_KnownAndUnknown covers a pos tag, a classifier tag (CJK
// character, case-sensitive-by-nature), and an unknown token.
func TestLookupTag_KnownAndUnknown(t *testing.T) {
	tests := []struct {
		value        string
		wantCategory string
		wantOK       bool
	}{
		{"N", "pos", true},
		{"f", "gender", true},
		{"sg", "number", true},
		{"个", "classifier", true},
		{"cl", "", false}, // the old spelled-out classifier form must NOT be declared
		{"ge", "", false},
		{"xyz", "", false},
	}
	for _, tt := range tests {
		got, ok := catalog.LookupTag(tt.value)
		if ok != tt.wantOK {
			t.Fatalf("LookupTag(%q) ok = %v, want %v", tt.value, ok, tt.wantOK)
		}
		if ok && got.Category != tt.wantCategory {
			t.Errorf("LookupTag(%q).Category = %q, want %q", tt.value, got.Category, tt.wantCategory)
		}
	}
}

// TestIsEnlarged_UnknownDefaultsFalse and TestDirection_UnknownDefaultsLTR
// pin the "unknown -> normal/ltr fallback" convention every existing
// scattered set already relied on (largeScript/isRtlScript).
func TestIsEnlarged_UnknownDefaultsFalse(t *testing.T) {
	if catalog.IsEnlarged("xyz") {
		t.Error("IsEnlarged(\"xyz\") = true, want false for an undeclared script")
	}
	if !catalog.IsEnlarged("hans") {
		t.Error("IsEnlarged(\"hans\") = false, want true")
	}
}

func TestDirection_UnknownDefaultsLTR(t *testing.T) {
	if catalog.Direction("xyz") != "ltr" {
		t.Errorf("Direction(\"xyz\") = %q, want \"ltr\" for an undeclared script", catalog.Direction("xyz"))
	}
	if catalog.Direction("arab") != "rtl" {
		t.Errorf("Direction(\"arab\") = %q, want \"rtl\"", catalog.Direction("arab"))
	}
}

func TestIsRTL_MatchesDirection(t *testing.T) {
	for _, s := range []string{"arab", "hebr", "syrc", "latn", "cyrl", "xyz"} {
		want := catalog.Direction(s) == "rtl"
		if got := catalog.IsRTL(s); got != want {
			t.Errorf("IsRTL(%q) = %v, want %v (Direction=%q)", s, got, want, catalog.Direction(s))
		}
	}
}

// TestHTMLLang_ReproducesLanguageInfoExactly is the exhaustive regression
// anchor: every case pkg/ebook's old TestLanguageInfoLanguageMapping table
// asserted, reproduced here against the new catalog-backed function, PLUS
// heb (which the old table asserted as the "en" quirk — here it's fixed).
func TestHTMLLang_ReproducesLanguageInfoExactly(t *testing.T) {
	tests := []struct {
		iso3, script, want string
	}{
		{"ajp", "", "ar"},
		{"apc", "", "ar"},
		{"arb", "", "ar"},
		{"bul", "", "bg"},
		{"ces", "", "cs"},
		{"cmn", "hant", "zh-Hant"},
		{"cmn", "hans", "zh-Hans"},
		{"cmn", "", "zh-Hans"},
		{"dan", "", "da"},
		{"deu", "", "de"},
		{"ell", "", "el"},
		{"fas", "", "fa"},
		{"fra", "", "fr"},
		{"grc", "", "el"},
		{"hin", "", "hi"},
		{"ind", "", "id"},
		{"ita", "", "it"},
		{"kaz", "", "kk"},
		{"lat", "", "la"},
		{"lit", "", "lt"},
		{"mon", "", "mn"},
		{"nld", "", "nl"},
		{"ron", "", "ro"},
		{"spa", "", "es"},
		{"srp", "", "sr"},
		{"tgk", "", "tg"},
		{"tha", "", "th"},
		{"tur", "", "tr"},
		{"uig", "", "ug"},
		{"ukr", "", "uk"},
		{"uzb", "", "uz"},
		{"vie", "", "vi"},
		{"yid", "", "yi"},
		{"yue", "hans", "zh-Hans"},
		{"yue", "hant", "zh-Hant"},
		{"yue", "", "zh-Hant"},
		{"xyz", "", "en"},
		{"", "", "en"},
		// heb: the pre-existing quirk is FIXED now that the catalog is the
		// single source of truth (no reproduce-the-old-switch constraint).
		{"heb", "hebr", "he"},
	}
	for _, tt := range tests {
		if got := catalog.HTMLLang(tt.iso3, tt.script); got != tt.want {
			t.Errorf("HTMLLang(%q, %q) = %q, want %q", tt.iso3, tt.script, got, tt.want)
		}
	}
}

// TestEnlargedScripts_MatchesDeclaredSet is a sanity anchor for the
// book.typ-drift test (pkg/ebook's typst_test.go): every script this test
// expects enlarged must actually be enlarged, and Latin/Cyrillic must not.
func TestEnlargedScripts_MatchesDeclaredSet(t *testing.T) {
	got := catalog.EnlargedScripts()
	want := map[string]bool{
		"hans": true, "hant": true, "hani": true, "arab": true, "hebr": true,
		"kore": true, "hang": true, "jpan": true, "hira": true, "kana": true, "syrc": true,
	}
	if len(got) != len(want) {
		t.Fatalf("EnlargedScripts() = %v (len %d), want %d entries", got, len(got), len(want))
	}
	for _, s := range got {
		if !want[s] {
			t.Errorf("EnlargedScripts() unexpectedly contains %q", s)
		}
	}
	for s := range want {
		if !catalog.IsEnlarged(s) {
			t.Errorf("expected %q to be enlarged, IsEnlarged() = false", s)
		}
	}
}
