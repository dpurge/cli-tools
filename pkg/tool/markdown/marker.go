package markdown

// Content-type badges (SPECS F-MARK). Each of the six content blocks gets a
// filled black-square badge with a knockout white letter that serves as the
// block's visual marker. The badge is ALWAYS emitted as a standalone element
// (never injected into a heading), at the fixed _ctbadge size, on the LEFT
// regardless of the block's own text direction (FR-2: Latin convention).
// Letters: T=text, V=vocabulary, D=dialog, M=models, Q=questions, P=parallel.
// No new AST NodeKind is introduced — these are rendering-time string
// additions only, so the 3-renderer panic-gate (ast.go) is untouched.

// contentBadgeTypst is the Typst literal invoking book.typ's `_ctbadge` helper
// for a single content-type letter.
func contentBadgeTypst(letter string) string {
	return `#_ctbadge("` + letter + `")`
}

// contentBadgeHTML is the EPUB badge span; `.ct-badge` (badge.css) styles it
// as the black square with the white letter.
func contentBadgeHTML(letter string) string {
	return `<span class="ct-badge">` + letter + `</span>`
}

// badgeOnlyTypst renders a standalone badge block (with title-line spacing).
// The badge always sits on the LEFT (FR-2: Latin convention, dir-independent).
func badgeOnlyTypst(letter string) string {
	return "#block(above: 1.2em, below: 0.5em)[" + contentBadgeTypst(letter) + "]\n\n"
}

// badgeOnlyHTML renders the standalone badge element emitted before a block's
// wrapper div. The badge always sits on the LEFT (FR-2: Latin convention,
// dir-independent) — no dir attribute is set on the badge element itself.
func badgeOnlyHTML(letter string) string {
	return `<div class="block-marker">` + contentBadgeHTML(letter) + "</div>\n"
}
