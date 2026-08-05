package markdown

import "strings"

// Content-type badges (SPECS F-MARK). Each of the five content blocks gets a
// filled black-square badge with a knockout white letter that also serves as
// the block's title line: when the block has a first heading the badge is
// injected INTO that heading (so it reads "[badge] Title" and stays in the
// PDF outline / EPUB nav), otherwise a standalone badge-only line is emitted
// to supply the separation that a missing title would otherwise leave. No new
// AST NodeKind is introduced — these are rendering-time string additions only,
// so the 3-renderer panic-gate (ast.go) is untouched. Letters: T=text,
// V=vocabulary, D=dialog, M=models, Q=questions (parallel/interlinear: none).

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

// badgeOnlyTypst renders a standalone badge line (its own block, with title-
// line spacing) for a block that has no heading title. dir-aware: an RTL badge
// (arab/hebr/syrc) is right-aligned so it sits at the visual start (NFR-4).
func badgeOnlyTypst(letter, dir string) string {
	inner := contentBadgeTypst(letter)
	if dir == "rtl" {
		return "#block(above: 1.2em, below: 0.5em, align(right)[" + inner + "])\n\n"
	}
	return "#block(above: 1.2em, below: 0.5em)[" + inner + "]\n\n"
}

// badgeOnlyHTML renders the standalone badge element emitted before a block's
// wrapper div. dir-aware (NFR-4): the block-marker sits outside the block's
// own dir-bearing wrapper, so RTL must be set explicitly here.
func badgeOnlyHTML(letter, dir string) string {
	d := ""
	if dir == "rtl" {
		d = ` dir="rtl"`
	}
	return `<div class="block-marker"` + d + `>` + contentBadgeHTML(letter) + "</div>\n"
}

// injectBadgeIntoFirstHeadingTypst inserts the badge just after the `=`-run
// prefix of the FIRST Typst heading line in rendered, returning (out, true)
// when a heading was found. A heading line is one whose first non-space run is
// one-or-more '=' immediately followed by a space — exactly what
// renderHeadingTypst emits — which never matches a '=' inside a #raw(...) code
// argument (that line starts with '#', not '='). The heading is left otherwise
// intact (level, children), so it still renders as a real heading and stays in
// the outline; the heading is never duplicated (SPECS FR-7).
func injectBadgeIntoFirstHeadingTypst(rendered, letter string) (string, bool) {
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lead := len(line) - len(strings.TrimLeft(line, " "))
		j := lead
		for j < len(line) && line[j] == '=' {
			j++
		}
		if j > lead && j < len(line) && line[j] == ' ' {
			// insert after the "= " prefix (the '=' run plus its single space)
			at := j + 1
			lines[i] = line[:at] + contentBadgeTypst(letter) + " " + line[at:]
			return strings.Join(lines, "\n"), true
		}
	}
	return rendered, false
}

// injectBadgeIntoFirstHeadingHTML adds the `block-title` class to the first
// <h1>/<h2>/<h3> open tag in rendered and inserts the badge span immediately
// after its '>', returning (out, true) when a heading was found. goldmark's
// auto `id` on the tag is preserved (we edit the open tag, never re-slug), and
// the heading is never duplicated (SPECS FR-7).
func injectBadgeIntoFirstHeadingHTML(rendered, letter string) (string, bool) {
	idx := -1
	for _, tag := range []string{"<h1", "<h2", "<h3"} {
		k := strings.Index(rendered, tag)
		// Require a tag-name boundary (space or '>') right after "<hN" so a
		// bare substring like "<h10" or "<h1foo" can't false-match; goldmark
		// only ever emits "<h1 …>" or "<h1>". This assumption also underwrites
		// the raw '>' scan below (goldmark with Unsafe=false never emits an
		// unescaped '>' inside an attribute value), so the open tag ends at the
		// first '>'. This helper is only fed goldmark-rendered HTML.
		if k >= 0 && k+3 < len(rendered) && (rendered[k+3] == ' ' || rendered[k+3] == '>') && (idx < 0 || k < idx) {
			idx = k
		}
	}
	if idx < 0 {
		return rendered, false
	}
	rel := strings.IndexByte(rendered[idx:], '>')
	if rel < 0 {
		return rendered, false
	}
	gt := idx + rel
	openTag := addClassBlockTitle(rendered[idx:gt])
	return rendered[:idx] + openTag + ">" + contentBadgeHTML(letter) + rendered[gt+1:], true
}

// addClassBlockTitle prepends `block-title` to an existing class attribute on
// the open tag, or adds a new class attribute when none is present. goldmark's
// heading tags carry only an `id`, so the else-branch is the usual path.
func addClassBlockTitle(openTag string) string {
	if strings.Contains(openTag, `class="`) {
		return strings.Replace(openTag, `class="`, `class="block-title `, 1)
	}
	return openTag + ` class="block-title"`
}
