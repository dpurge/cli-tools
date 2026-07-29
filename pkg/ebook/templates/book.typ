#let _baseFont = (
  "Gentium", "Charis SIL", "Noto Serif",
  "Amiri", "Scheherazade", "Noto Naskh Arabic", "Noto Sans Arabic",
  "Ezra SIL", "Frank Ruehl CLM", "Noto Serif Hebrew", "Noto Sans Hebrew",
  "AR PL UMing", "AR PL UKai",
  "Baekmuk Batang",
  "Noto Sans",
)

#let _vocabFonts = state("vocab-fonts", (
  header: _baseFont, transcription: _baseFont, translation: _baseFont,
))

#let vocabulary(..items) = block(width: 100%, grid(
  columns: (auto, 1fr),
  column-gutter: 1em,
  stroke: (y: 0.5pt + luma(220)),
  align: (start + top, start + top),
  inset: (x: 2pt, y: 4pt),
  ..items.pos().map(it => (
    {
      strong(it.at("phrase", default: ""))
      if it.at("grammar", default: "") != "" {
        [ ]; context text(font: _vocabFonts.get().header, size: 0.85em, fill: gray)[#it.at("grammar")]
      }
      if it.at("transcription", default: "") != "" {
        [ ]; emph[#context text(font: _vocabFonts.get().transcription)[\[#it.at("transcription")\]]]
      }
    },
    context text(font: _vocabFonts.get().translation)[#it.at("translation", default: "")],
  )).flatten()
))

#let models(..items) = block(width: 100%, grid(
  columns: (auto, 1fr),
  column-gutter: 1em,
  align: (start + top, start + top),
  inset: (x: 2pt, y: 4pt),
  ..items.pos().map(it => {
    let phrase = it.at("phrase", default: "")
    let transcription = it.at("transcription", default: "")
    let translation = it.at("translation", default: "")
    if transcription == "" and translation == "" {
      (grid.cell(colspan: 2, strong(phrase)),)
    } else {
      (
        {
          strong(phrase)
          if transcription != "" and translation != "" {
            linebreak()
            emph[#context text(font: _vocabFonts.get().transcription)[\[#transcription\]]]
          }
        },
        if translation != "" {
          context text(font: _vocabFonts.get().translation)[#translation]
        } else if transcription != "" {
          emph[#context text(font: _vocabFonts.get().transcription)[\[#transcription\]]]
        } else { [] },
      )
    }
  }).flatten()
))

#let questions(..items) = {
  let run = ()
  for it in items.pos() {
    let question = it.at("question", default: "")
    let answer = it.at("answer", default: "")
    if answer != "" {
      run += (question, answer)
    } else {
      if run.len() > 0 {
        grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
        run = ()
      }
      question
      parbreak()
    }
  }
  if run.len() > 0 {
    grid(columns: (auto, 1fr), column-gutter: 1em, row-gutter: 0.5em, align: (start + top, start + top), ..run)
  }
}

#let dialog(..turns) = block(width: 100%, grid(
  columns: (auto, 1fr), column-gutter: 0.8em, row-gutter: 0.5em,
  ..turns.pos().map(t => (strong(t.at("header", default: "")), t.at("content", default: []))).flatten()
))

#let parallel(..rows) = block(width: 100%, grid(
  columns: (1fr, 1fr), column-gutter: 1.2em, row-gutter: 0.5em,
  stroke: (x: 0.5pt + luma(230)),
  ..rows.pos().map(r => (r.at("main", default: []), r.at("secondary", default: []))).flatten()
))

#let book(
  title: none,
  author: none,
  description: none,
  lang: "en",
  dir: ltr,
  cover: none,
  paper: "a5",
  size: 12pt,
  size-large: 16pt,
  large-script: false,
  margin: (inside: 1.8cm, outside: 1.4cm, top: 1.7cm, bottom: 2cm),
  font: _baseFont,
  font-body: (),
  font-header: (),
  font-transcription: (),
  font-translation: (),
  font-strong: (),
  font-emph: (),
  body,
) = {
  let headerFont = font-header + font
  let strongFont = font-strong + font
  let emphFont = font-emph + font
  _vocabFonts.update((
    header: headerFont,
    transcription: font-transcription + font,
    translation: font-translation + font,
  ))

  set document(title: title, author: if author == none or author == "" { () } else { author })
  set text(
    lang: lang,
    dir: dir,
    size: if large-script { size-large } else { size },
    font: font-body + font,
    hyphenate: true,
  )
  set page(paper: paper, margin: margin, numbering: "1")
  set par(justify: true, leading: 0.7em, spacing: 0.7em, first-line-indent: (amount: 1.2em, all: false))
  show heading: set par(justify: false, first-line-indent: 0pt)
  set terms(separator: [: ], tight: true, hanging-indent: 1em)
  set table(
    stroke: (x: none, y: 0.7pt + luma(180)),
    inset: (x: 8pt, y: 5pt),
    align: (_, y) => if y == 0 { center } else { left },
    fill: (_, y) => if y == 0 { luma(235) } else { none },
  )
  show heading: set block(above: 1.5em, below: 0.75em)
  show heading: set text(font: headerFont)
  show heading.where(level: 1): set text(size: 1.7em)
  show heading.where(level: 2): set text(size: 1.4em)
  show heading.where(level: 3): set text(size: 1.2em)
  show heading.where(level: 4): set text(size: 1.1em)
  show heading.where(level: 5): set text(size: 1.05em)
  show heading.where(level: 6): set text(size: 1em)
  show strong: it => if large-script { text(font: strongFont, weight: "regular", it.body) } else { it }
  show emph: it => if large-script { text(font: emphFont, style: "normal", it.body) } else { it }
  show heading: it => if large-script { set text(weight: "regular"); it } else { it }
  show quote.where(block: true): it => block(inset: (left: 1em, y: 0.3em), stroke: (left: 2pt + luma(180)))[#emph(it.body)]

  set page(numbering: none)

  if cover != none {
    page(margin: 0pt, numbering: none, {
      if type(cover) == str {
        image(cover, width: 100%, height: 100%, fit: "cover")
      } else { cover }
    })
  }

  {
    set par(justify: false)
    set text(hyphenate: false)
    align(center + horizon, {
      if large-script {
        text(size: 2.4em, font: strongFont, title)
      } else {
        text(size: 2.4em, weight: "bold", title)
      }
      if author != none and author != "" { v(1.2em); text(size: 1.3em, author) }
      if description != none and description != "" {
        v(1.6em)
        if large-script {
          text(font: emphFont, fill: luma(90%), description)
        } else {
          text(style: "italic", fill: luma(90%), description)
        }
      }
    })
  }
  pagebreak()

  set page(numbering: "i")
  counter(page).update(1)
  show outline.entry.where(level: 1): strong
  outline(title: [Contents], indent: auto)
  pagebreak()

  set page(numbering: "1")
  counter(page).update(1)

  body
}
