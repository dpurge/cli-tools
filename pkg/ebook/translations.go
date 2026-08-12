package ebook

import "strings"

// translations.go — compiled-in catalog mapping a book language to its
// rendering strings (FR-7). Using embedded Go source (not a runtime file)
// means the catalog is always in sync with the binary; updating it requires
// only adding one map entry and rebuilding.
//
// How to add a language
// ---------------------
// 1. Add one entry to bookStrings: key = bare lowercase ISO 639 subtag
//    (the same form typstLang() produces, e.g. "ar", "zh", "fr").
// 2. Add only a translation you can verify. If unsure, omit the entry so
//    book.typ's [Contents] default continues to stand for that language.
// 3. Add a comment "// please verify" next to any entry you are not certain of.
// 4. Rebuild. No other change required.

// bookStringSet holds the UI strings for one language. Designed to be
// extended: add a new field here and populate it in bookStrings below.
// (Named bookStringSet, not bookStrings, to avoid the identifier collision
// with the package-level var bookStrings below — Go forbids a type and a
// package-level variable sharing one identifier.)
type bookStringSet struct {
	// Contents is the localised title of the table of contents, used as the
	// `contents-title:` argument to book.typ's book() function.
	Contents string
}

// bookStrings maps bare lowercase ISO 639 subtags to their UI string sets.
// Keyed by the same form typstLang() produces (e.g. "en", not "en-US").
// Add entries only with verified translations (see header comment).
var bookStrings = map[string]bookStringSet{
	"en": {Contents: "Contents"},
	// "ar": {Contents: "المحتويات"}, // please verify before enabling
	// "fr": {Contents: "Sommaire"},  // please verify before enabling
	// "de": {Contents: "Inhalt"},    // please verify before enabling
}

// resolveContentsTitle returns the string to emit as contents-title in the
// assembled Typst document (FR-7):
//   - explicit (the book's per-file ContentsTitle override) wins when non-empty;
//   - otherwise the catalog entry for lang is used;
//   - otherwise "" is returned and the caller omits the argument, leaving
//     book.typ's built-in [Contents] default in place.
//
// lang is passed as-is from assembleTypstDocument; it goes through typstLang()
// (strips BCP-47 subtags, lowercases) before the catalog lookup so
// "en-US", "EN", and "en" all resolve to the same entry.
func resolveContentsTitle(explicit, lang string) string {
	if explicit != "" {
		return explicit
	}
	return bookStrings[strings.ToLower(typstLang(lang))].Contents
}
