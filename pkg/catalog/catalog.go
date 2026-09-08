// Package catalog is the single declared source of truth for every
// language, script, and vocabulary grammar tag this tool recognizes.
//
// Before this package existed, that knowledge was scattered across four
// independently-maintained closed sets with inconsistent membership:
// pkg/ebook/exporter.go's languageInfo (language -> BCP-47 tag, plus its
// own separate RTL check), pkg/ebook/typst.go's largeScriptCodes (mirrored
// by hand in pkg/ebook/templates/book.typ's _largeScripts), and
// pkg/tool/markdown/attr.go's isRtlScript. All four now consult this
// package instead of their own hardcoded switches/maps (book.typ's
// _largeScripts stays a static literal for Typst-runtime-risk reasons —
// see its own doc comment — but a test asserts it never drifts from this
// catalog's Enlarged set).
//
// Data lives in the embedded catalog.yml (go:embed), not runtime-loaded
// config, so the catalog is always in sync with the binary — the same
// reasoning pkg/ebook/translations.go's bookStrings documents for the
// identical reason.
package catalog

import (
	_ "embed"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed catalog.yml
var catalogYAML []byte

// Language is one declared ISO 639-3 language entry.
type Language struct {
	ISO3 string `yaml:"iso3"`
	ISO1 string `yaml:"iso1"`
	Name string `yaml:"name"`
}

// Script is one declared ISO 15924 script entry. Enlarged mirrors the
// pre-existing largeScriptCodes semantics (book.typ substitutes a distinct
// role font at normal weight/style, and the book-level base size grows,
// for a script whose synthetic bold/italic renders poorly). Direction is
// "ltr" or "rtl".
type Script struct {
	ISO4      string `yaml:"iso4"`
	Name      string `yaml:"name"`
	Enlarged  bool   `yaml:"enlarged"`
	Direction string `yaml:"direction"`
}

// Tag is one declared {start-vocabulary} grammar-field token (SPECS: the
// Grammar field is whitespace-split; each token is independently declared
// and composed at authoring time, e.g. "N f sg" = noun + feminine +
// singular).
type Tag struct {
	Value    string `yaml:"value"`
	Category string `yaml:"category"`
	Meaning  string `yaml:"meaning"`
}

// data is the parsed shape of catalog.yml.
type data struct {
	Languages []Language `yaml:"languages"`
	Scripts   []Script   `yaml:"scripts"`
	Tags      []Tag      `yaml:"tags"`
}

var (
	languages map[string]Language // keyed by lowercase ISO3
	scripts   map[string]Script   // keyed by lowercase ISO4
	tags      map[string]Tag      // keyed by tag value verbatim (case-sensitive: classifiers are CJK characters, "N"/"n" are distinct grammar tags)
)

// init parses the embedded catalog.yml once at program start. A malformed
// catalog.yml is a build-time programmer error (it ships inside the
// binary, never user-supplied), so it panics rather than returning an
// error every caller would have to thread through — mirrors how a bad
// //go:embed template would already fail impossible to recover from at
// runtime.
func init() {
	var d data
	if err := yaml.Unmarshal(catalogYAML, &d); err != nil {
		panic("catalog: malformed embedded catalog.yml: " + err.Error())
	}

	languages = make(map[string]Language, len(d.Languages))
	for _, l := range d.Languages {
		languages[strings.ToLower(l.ISO3)] = l
	}

	scripts = make(map[string]Script, len(d.Scripts))
	for _, s := range d.Scripts {
		scripts[strings.ToLower(s.ISO4)] = s
	}

	tags = make(map[string]Tag, len(d.Tags))
	for _, t := range d.Tags {
		tags[t.Value] = t
	}
}

// normalize lowercases and trims an ISO code before lookup, matching the
// defensive normalization every existing scattered set already applied
// (largeScript's strings.ToLower(strings.TrimSpace(...)), book.typ's
// _resolveFont doing the same Typst-side).
func normalize(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

// LookupLanguage returns the declared entry for an ISO 639-3 code, or
// ok=false when the code is not declared.
func LookupLanguage(iso3 string) (Language, bool) {
	l, ok := languages[normalize(iso3)]
	return l, ok
}

// LookupScript returns the declared entry for an ISO 15924 code, or
// ok=false when the code is not declared.
func LookupScript(iso4 string) (Script, bool) {
	s, ok := scripts[normalize(iso4)]
	return s, ok
}

// LookupTag returns the declared entry for a grammar-field token, or
// ok=false when the token is not declared. Tag values are matched
// verbatim (case-sensitive): grammar tags distinguish e.g. "N" (noun) from
// a hypothetical lowercase variant, and classifier tags are CJK
// characters with no meaningful case-folding.
func LookupTag(value string) (Tag, bool) {
	t, ok := tags[value]
	return t, ok
}

// IsEnlarged reports whether an ISO 15924 script code selects the
// enlarged-body-size / large-script font substitution treatment. An
// undeclared script returns false (the same "unknown -> normal size"
// fallback largeScript already used).
func IsEnlarged(iso4 string) bool {
	s, ok := LookupScript(iso4)
	return ok && s.Enlarged
}

// Direction returns "rtl" for a declared RTL script and "ltr" for every
// other case, including an undeclared script (OI-6 LTR-fallback rule,
// matching isRtlScript/languageInfo's existing convention).
func Direction(iso4 string) string {
	if s, ok := LookupScript(iso4); ok && s.Direction == "rtl" {
		return "rtl"
	}
	return "ltr"
}

// IsRTL reports whether an ISO 15924 script code is declared
// right-to-left. Equivalent to Direction(iso4) == "rtl", provided as the
// boolean form pkg/tool/markdown/attr.go's isRtlScript callers expect.
func IsRTL(iso4 string) bool {
	return Direction(iso4) == "rtl"
}

// HTMLLang returns the BCP-47-ish tag languageInfo's callers expect for an
// ISO 639-3 language code, given its ISO 15924 script (only consulted for
// cmn/yue's Hans/Hant qualification). An undeclared language returns "en"
// — languageInfo's pre-existing default fallback.
//
// cmn/yue are special-cased here, not declared as separate catalog rows,
// because their resolved tag depends on BOTH language and script — this
// mirrors setLanguage/languageInfo's exact pre-existing logic (SPECS AC7):
// cmn defaults to zh-Hans unless script is "hant"; yue defaults to
// zh-Hant unless script is "hans" (the two are NOT symmetric — this
// asymmetry is preserved verbatim from the code being replaced).
func HTMLLang(iso3, scriptISO4 string) string {
	lang, ok := LookupLanguage(iso3)
	if !ok {
		return "en"
	}

	script := normalize(scriptISO4)
	switch normalize(iso3) {
	case "cmn":
		if script == "hant" {
			return "zh-Hant"
		}
		return "zh-Hans"
	case "yue":
		if script == "hans" {
			return "zh-Hans"
		}
		return "zh-Hant"
	default:
		return lang.ISO1
	}
}

// EnlargedScripts returns every declared ISO 15924 code whose Enlarged
// flag is true, sorted for deterministic output — used by the
// book.typ-drift test (typst_test.go) to assert the template's own static
// _largeScripts literal never falls out of sync with this catalog.
func EnlargedScripts() []string {
	out := make([]string, 0, len(scripts))
	for iso4, s := range scripts {
		if s.Enlarged {
			out = append(out, iso4)
		}
	}
	sort.Strings(out)
	return out
}
