// book.typ — DPurge language-book Typst template
//
// Typography adapted from min-manual (src/lib.typ): 13pt body text,
// page margins top:3cm/bottom:2cm/x:2cm, heading scale x2/1.6/1.4/1.3/1.2/1.1,
// gray-header table styling, `terms` styling, justified paragraphs.
// The manual/nexus-tools chrome (page header/footer credits, `purl`/`callout`
// helpers, package-URL commands, etc.) is intentionally NOT reused — this
// template is scoped to language-book rendering: title page, `#outline` TOC,
// RTL support, and the `vocabulary`/`dialog`/`parallel` custom blocks used by
// the Typst markdown renderer (pkg/tool/markdown).
//
// Self-contained: no `#import "@preview/..."`, no nexus-tools, no other
// external package — safe to `go:embed` and `typst compile` standalone.

// #vocabulary(
//   (phrase: "..", grammar: "..", transcription: "..", translation: ".."),
//   ..
// )
// One row per item: phrase (+ optional grammar tag, + optional bracketed
// transcription) on the left, translation on the right. Missing/empty fields
// are omitted rather than rendered blank.
//
// Uses `grid`, NOT `table`: the document-wide `set table(...)` below styles
// GFM content tables with a shaded header row, and a `table` here would
// inherit it — shading/centering the first vocabulary item as a bogus header.
// `grid` is unaffected by `set table` (as `dialog`/`parallel` already are).
#let vocabulary(..items) = block(width: 100%, grid(
  columns: (auto, 1fr),
  stroke: none,
  inset: (x: 4pt, y: 3pt),
  ..items.pos().map(it => (
    {
      strong(it.at("phrase", default: ""))
      if it.at("grammar", default: "") != "" { [ ]; text(size: 0.85em, fill: gray)[#it.at("grammar")] }
      if it.at("transcription", default: "") != "" { [ ]; emph[\[#it.at("transcription")\]] }
    },
    it.at("translation", default: ""),
  )).flatten()
))

// #dialog((header: "..", content: [..]), ..)
// One row per turn: header (speaker label) then content, side by side.
#let dialog(..turns) = block(width: 100%, {
  for t in turns.pos() {
    grid(columns: (auto, 1fr), column-gutter: 0.8em,
      strong(t.at("header", default: "")), t.at("content", default: []))
    v(0.4em)
  }
})

// #parallel((main: [..], secondary: [..]), ..)
// One row per pair: main text and its secondary (e.g. translated) text side
// by side. An empty secondary renders as an empty cell, not a crash.
#let parallel(..rows) = block(width: 100%, {
  for r in rows.pos() {
    grid(columns: (1fr, 1fr), column-gutter: 1em,
      r.at("main", default: []), r.at("secondary", default: []))
    v(0.4em)
  }
})

// #show: book.with(title: .., author: .., description: .., lang: .., dir: .., cover: ..)
// Show-rule entry point: sets document metadata, typography, page layout,
// then emits a title page and a heading-driven table of contents before the
// document body.
#let book(
  title: none,
  author: none,
  description: none,
  lang: "en",
  dir: ltr,
  cover: none,
  body,
) = {
  // `set document(author: none)` errors (author must be str/array), and the
  // exporter passes "" for an absent author — normalize both to an empty
  // author list so the template is robust called with its own defaults too.
  set document(title: title, author: if author == none or author == "" { () } else { author })
  set text(
    lang: lang,
    dir: dir,
    size: 13pt,
    // Latin base (TeX Gyre / Arial) plus Noto fallbacks so non-Latin
    // scripts (Arabic, Hebrew, CJK) resolve per-glyph without a separate
    // script parameter — Typst tries each family in order per character.
    font: (
      "TeX Gyre Heros", "Arial",
      "Noto Sans Arabic", "Noto Sans Hebrew",
      "Noto Sans CJK SC", "Noto Sans SC",
      "Noto Sans",
    ),
    hyphenate: true,
  )
  set page(margin: (top: 3cm, bottom: 2cm, x: 2cm), numbering: "1")
  set par(justify: true)
  // Justify body paragraphs, but NOT headings: justified large bold chapter
  // titles produce ugly wide inter-word gaps on short lines.
  show heading: set par(justify: false)
  set terms(separator: [: ], tight: true, hanging-indent: 1em)
  set table(
    stroke: gray.lighten(60%),
    inset: 10pt,
    align: (_, y) => if y == 0 { center } else { left },
    fill: (_, y) => if y == 0 { gray.lighten(85%) } else { none },
  )
  show heading.where(level: 1): set text(size: 2em)
  show heading.where(level: 2): set text(size: 1.6em)
  show heading.where(level: 3): set text(size: 1.4em)
  show heading.where(level: 4): set text(size: 1.3em)
  show heading.where(level: 5): set text(size: 1.2em)
  show heading.where(level: 6): set text(size: 1.1em)
  show quote.where(block: true): it => pad(x: 1em, it)

  align(center + horizon, {
    if cover != none {
      // Accept either a path string (`cover: "path.svg"`, per the exporter
      // contract — project.Cover resolved to an absolute image path) or
      // pre-built content (`cover: image("path.svg")`), so a caller on
      // either side of that contract still compiles correctly.
      if type(cover) == str { image(cover, width: 50%) } else { cover }
      v(1em)
    }
    text(size: 2.2em, weight: "bold", title)
    // Treat "" like none (the exporter emits "" for absent fields) so no
    // empty author/description line or stray spacer appears on the title page.
    if author != none and author != "" { v(1em); text(size: 1.3em, author) }
    if description != none and description != "" { v(2em); text(style: "italic", description) }
  })
  pagebreak()
  outline(title: [Contents])
  pagebreak()

  body
}
