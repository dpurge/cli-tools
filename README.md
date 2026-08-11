# CLI tools

Command-line tools for private e-book / language-learning projects. Three binaries are built from this repo:

- **`ebook-cli`** — build an e-book project into EPUB, PDF, or MDX.
- **`scanbook-cli`** — scanned-page / PDF utilities.
- **`flashcard-cli`** — flashcard tooling (work in progress).

## Install

Prebuilt Linux and Windows binaries are attached to each [release](../../releases), and a container image is published to `ghcr.io/dpurge/cli-tools`.

Build locally (needs [Go](https://go.dev) 1.25+ and [go-task](https://taskfile.dev)):

```sh
task build   # → dist/{ebook,flashcard,scanbook}-cli(.exe)  (linux + windows)
```

Or with [GoReleaser](https://goreleaser.com): `goreleaser build --snapshot --clean`.

## `ebook-cli`

Build an e-book from a project file into one or more formats:

```sh
ebook-cli build -p ebook.yml                  # EPUB (default)
ebook-cli build -p ebook.yml -f pdf           # PDF (via Typst)
ebook-cli build -p ebook.yml -f epub,pdf,mdx  # all three
```

- `-f, --format` — `epub` (default), `pdf`, `mdx`; repeatable or comma-separated. An unknown format is rejected before anything is written.
- `-p, --project` — project file (default `ebook.yml`).
- Output: EPUB and PDF are written next to the project's `filename`; MDX is written to a `<name>-mdx/` directory (one `.mdx` per chapter + a `_category_.json`).

**PDF** export generates [Typst](https://typst.app) source and compiles it, so a `typst` binary must be on `PATH` (or set `Typst.typst` in the config). The container image ships Typst.

Other subcommands:

```sh
ebook-cli vocab -p ebook.yml   # export the vocabulary blocks to CSV
ebook-cli --version            # print the version
```

### Project file (`ebook.yml`)

Book metadata plus an ordered list of markdown sources:

```yml
identifier: urn:uuid:...
filename: my-book.epub
title: My Book
author: Jane Doe
language: tur          # ISO 639-3
script: latn           # ISO 15924
cover: cover.svg
description: ...
stylesheet:
  common: [base.css]
  section: section.css
  chapter: chapter.css
text:
  - [section.md, 01.md, 02.md]   # [section, chapter, chapter, ...]
```

Chapters are CommonMark/GFM markdown plus custom blocks rendered natively into each output format. Block markers take `lang` (ISO 639-3) and `script` (ISO 15924) attributes; the unified `{start-text as=...}` block also takes an `as=` role. **`script` — not the book's `language` — now determines each block's text direction and font role.** This is a behavior change for existing content: a marker with no `script` renders left-to-right regardless of the book language, so right-to-left projects must set `script=` (e.g. `arab`) on their block markers.

```
{start-vocabulary lang=arb script=arab}
كتاب {N m} [kitāb] = book
{end-vocabulary}

{start-text as=source lang=arb script=arab}
## Heading

Paragraph in the source language.
{end-text}
```

**`script` drives direction**: `arab`, `hebr`, and `syrc` scripts → RTL; all others → LTR. The `as=transcription` role is always pinned LTR (romanization). Fonts come from the `font.css` roles declared in your stylesheet — either the plain book-wide roles or, for finer control, per-`script`/`extension`/`field` roles (see [Font configuration](#font-configuration-fontcss) below).

**`{start-text as=source|transcription|translation|grammar lang=... script=...}` ... `{end-text}`**

Unified text block with four roles:

- `as=source` — body font, direction from `script`.
- `as=transcription` — transcription font, pinned LTR.
- `as=translation` — translation font, direction from `script`.
- `as=grammar` — translation font, direction from `script` for prose; tables inside render in the source language's direction and font at full text-block width.

Headings (`h1`–`h3`) inside any text block are centered, and markdown tables span the full text-block width in both outputs. For PDF the direction and font are resolved via `book.typ`'s `#textblock(role:, dir:, ...)` function; for EPUB the class (`text`, `transcription`, `translation`, `grammar`) and `dir` attribute on the wrapper `<div>` drive the matching CSS rules in your stylesheet bundle.

**Block set**: `{start-vocabulary}`, `{start-models}`, `{start-questions}`, `{start-dialog}`, `{start-parallel}`, and `{start-text}`. **`as=` roles** are unified across the blocks that carry a source/translation distinction: `{start-text}` takes `as=source|transcription|translation|grammar`; `{start-dialog}` and `{start-questions}` take `as=source|translation` (an `as=translation` block is in the reader's own language — comprehension questions, a translated dialog — and uses the Translation font); `{start-vocabulary}`, `{start-models}`, and `{start-parallel}` reject `as=` because their field languages are fixed. Validation: an unrecognized `script` value falls back to LTR (no error); an **unknown attribute key**, a **malformed attribute** (missing `=` or unterminated quote), or an **`as=` value not accepted by that block** fails the build with a message naming the offending marker.

**Headers and notes**: `vocabulary`, `models`, `questions`, and `dialog` blocks accept a line starting with `#` through `######` anywhere inside them as a heading (renders as `h1`–`h6`), interleaved in place among the block's data lines — it's a visual heading local to the block, not a table-of-contents entry. `dialog`, `questions`, and `models` (not `vocabulary`) additionally accept a note — a sentence or phrase alone on a line inside `(...)` — rendered as a centered paragraph (see **Notes** under [Font configuration](#font-configuration-fontcss)). Vocabulary export to CSV skips header lines entirely (no row emitted); phraseforge/MDX export keeps them as literal text.

**`contents-title` project key**: set in `ebook.yml` to override the PDF outline title (default "Contents"):

```yml
contents-title: Spis treści
```

The EPUB nav title is not configurable (go-epub does not expose a setter).

**Block types**:

#### `{start-models}` ... `{end-models}`

Like `{start-vocabulary}` but without a grammar tag or notes. Each line is:

```
phrase [transcription] = translation
```

`[transcription]` and ` = translation` are each optional, giving four render variants:

```
phrase only:                    run
phrase + transcription:         run [rʌn]
phrase + translation:           run = biec
phrase + transcription + trans: run [rʌn] = biec
```

- **phrase only** renders as a plain line (no table/grid row).
- **phrase + transcription** or **phrase + translation** render as a top-aligned two-column row (phrase | transcription-or-translation).
- **phrase + transcription + translation** renders as a two-column row whose first column stacks the transcription below the phrase, top-aligned with the translation in the second column.

Fonts match `{start-vocabulary}`'s roles: phrase is bold (body font), transcription is italic (`Font Transcription`), translation uses `Font Translation`.

#### `{start-questions}` ... `{end-questions}`

Each line is:

```
question = answer
```

` = answer` is optional — a line with no answer is a question-only line. Example:

```
{start-questions}
What did you notice about the ending?
Who is the narrator? = An unnamed neighbor.
What year is it set in? = 1920s.
{end-questions}
```

A question-only line renders as a normal paragraph (body font). Consecutive question+answer lines are grouped into one aligned two-column block (question | answer, top-aligned, body font); a question-only line flushes the current group, so a block may contain several such runs.

#### `{start-parallel}` ... `{end-parallel}`

Two-column parallel text (e.g. source + translation side-by-side). Rows are separated by a lone `===` line; within a row, every lone `---` line splits the record into up to **three** fields — **source**, **translation**, **transcription** (the last two optional). `as=` is not accepted.

```
{start-parallel lang=lat script=latn}
Et rex David senuerat habebatque aetatis plurimos dies.
---
Now king David was old, and advanced in years.
===
Dixerunt ergo ei servi sui.
---
His servants therefore, said to him.
---
Dixerunt ergo ei servi sui.
{end-parallel}
```

The second row above shows the optional third field (transcription).

> **Behavior change:** this reverses the column rule from earlier releases (primary column used to inherit the book's language with no marker override; the marker's `lang=`/`script=` used to drive the *secondary* column only). The rule below is the current, correct one.

**Column rule**: the **primary column is the source** — its text direction, font, and (for large scripts) size all come from the marker's `lang=`/`script=` attributes (falling back to the book's own language/script when the marker omits them, same as every other block). If present, the **transcription** stacks directly below the source, inside the same primary column — always rendered as a Latin-script, left-to-right romanization (matching `{start-vocabulary}`'s transcription field), regardless of the marker's or book's own script. The **secondary column is the translation** — always in the book's own language, script, and font, with no marker override. Column *position* (left/right) still follows the **book's** reading direction (RTL book → primary on the right, LTR book → primary on the left); only each column's internal direction/font follows the rule above.

#### EPUB stylesheets

The models/questions markup needs a matching CSS bundle listed under the project's `stylesheet.common` (`epub-public`'s `src/css/main/{latn,arab,hebr,cjk}/models.css` and `questions.css` are ready-made examples — copy them into your own stylesheet set, or use them directly if your project already pulls from that repo). `cjk` is one shared bundle for Chinese/Japanese/Korean (no per-script `hans`/`hant`/`kore`/`jpan` variants). Example `models.css` (`latn`):

```css
div.models > div.models-item.paired {
    display: table;
    width: 100%;
}
div.models > div.models-item.paired > div.models-col1,
div.models > div.models-item.paired > div.models-col2 {
    display: table-cell;
    vertical-align: top;
}
div.models span.models-phrase { font-weight: bold; }
div.models span.models-transcription { font-family: "Font Transcription", sans-serif; font-style: italic; }
div.models span.models-translation { font-family: "Font Translation", serif; }
```

Example `questions.css` (`latn`):

```css
div.questions { font-family: "Font Body", serif; }
div.questions > div.questions-group {
    display: table;
    width: 100%;
}
div.questions > div.questions-group > div.questions-item.paired {
    display: table-row;
}
div.questions > div.questions-group > div.questions-item.paired > div.questions-col1,
div.questions > div.questions-group > div.questions-item.paired > div.questions-col2 {
    display: table-cell;
    vertical-align: top;
}
```

The `arab`/`hebr` variants keep the same column order (no reordering under RTL) and right-align the leading column's text (`models-col1`/`questions-col1`, and the question-only paragraph) instead.

#### Font configuration (`font.css`)

`font.css` is the single source of truth for fonts: the EPUB uses its `@font-face` names directly and the PDF (Typst) reads the **same** file (from `stylesheet.common`), so both outputs pick the same font for every slot. Each entry maps a role name to a real installed family via `src: local(...)`. A role with no matching `@font-face` falls back to a recommended installed family (`Noto Sans` header, `Gentium` body/translation/strong/emphasis, `DejaVu Sans` transcription), so an incomplete `font.css` still renders.

**Book-wide roles** — always available, fully backward compatible — name the whole book's fonts:

```css
@font-face { font-family: "Font Header";        src: local("Noto Sans");  }
@font-face { font-family: "Font Body";          src: local("Noto Serif"); }
@font-face { font-family: "Font Transcription"; src: local("Noto Sans");  }
@font-face { font-family: "Font Translation";   src: local("Noto Serif"); }
```

**Per-slot fonts.** To give a particular script / block / field its own font, *qualify* the role name. Segments run general → specific, and any may be omitted (an omitted segment applies to all values of that axis):

```
Font <Script> <Extension> <Field> [Strong|Emphasis]
```

| Axis | Values |
|---|---|
| Script | ISO-15924, Titlecase — `Arab` `Hebr` `Latn` … |
| Extension | `Text` `Dialog` `Questions` `Vocabulary` `Models` `Parallel` |
| Field | `Source` `Question` `Answer` `Transcription` `Translation` `Grammar` `Phrase` … |
| Style | `Strong` or `Emphasis` (omitted = regular) |

The six book-wide roles are just the zero-qualifier form of this grammar. Example — a distinct Arabic font for the question vs the answer, and distinct transcription fonts for a text block vs a vocabulary list:

```css
@font-face { font-family: "Font Arab Questions Question";       src: local("Noto Naskh Arabic"); }
@font-face { font-family: "Font Arab Questions Answer";         src: local("Amiri");             }
@font-face { font-family: "Font Latn Text Transcription";       src: local("DejaVu Sans");       }
@font-face { font-family: "Font Latn Vocabulary Transcription"; src: local("Noto Sans");         }
```

**Resolution** picks the most specific declared slot, dropping one axis at a time:

```
Font <S> <E> <F>  →  Font <S> <E>  →  Font <S>  →  Font <base role>  →  generic (serif/sans)
```

Declare only the slots you want to override — undeclared names are skipped. A **field** always needs its **extension** to resolve: `Font Hebr Vocabulary Phrase` works, `Font Hebr Phrase` never matches.

**How each output applies it.** On EPUB each qualified name becomes a `font-family` fallback list on the field's CSS selector. Fields that follow the block's own script are scoped by an `s-<script>` class the renderer puts on the block wrapper (`<div class="questions s-arab">`), so one book can mix scripts; fields whose language is fixed regardless of the block — transcription (romanization → `latn`) and translation / grammar (the base language) — are matched unscoped. The PDF resolver reproduces the same order from the same `font.css`. Which base role a block's main text plays follows its `as=` role (`source`→Body, `translation`→Translation, `transcription`→Transcription); a `dialog`/`questions` block marked `as=translation` also emits an `as-translation` class so EPUB matches the PDF.

**Strong / Emphasis.** For large scripts (Arabic, Hebrew, CJK, Korean, Japanese) synthetic bold/italic renders poorly, so bold (`**…**` / `<strong>`/`<b>`) and italic (`*…*` / `<em>`/`<i>`) switch to the resolved `Strong`/`Emphasis` slot at *normal* weight/style; Latin and other scripts keep real bold/italic. The switch is decided **per field, by that field's resolved script** — so a Latin translation inside an Arabic block still gets ordinary Latin bold, not an Arabic strong face. Declare `Font <Script> Strong`/`Emphasis` (or plain `Font Strong`/`Font Emphasis`) to choose the substitute; when undeclared, bold/italic simply falls back to the regular resolved font.

**Notes.** A comment/note line — a sentence or phrase alone on a line, wrapped in `(...)` — inside a `dialog`, `questions`, or `models` block (not `vocabulary`) renders as a centered paragraph in the `Notes` role. Declare `Font Notes` to choose its font; when undeclared, notes fall back to the `Emphasis` font.

The ready-made bundles under `epub-public`'s `src/css/main/{arab,hebr,latn}/` show the full pattern — a `font.css` palette plus the per-component CSS chains that spell out these fallback lists.

## `scanbook-cli`

Utilities for scanned-book pages. These call external tools (ImageMagick, Poppler, DjVuLibre, …) resolved via the [config](#configuration); the container image ships them.

Combine a directory of scanned pages into a single PDF:

```sh
scanbook-cli pdf --input ./book-pages --output my-book              # → my-book.pdf
scanbook-cli pdf --input ./book-pages --output my-book --format png
```

- `-i, --input` — directory of scanned pages (required).
- `-o, --output` — output book name; `.pdf` is appended unless already present (required).
- `-f, --format` — page-image extension to read (default `png`).
- Pages are combined in filename order (via ImageMagick `convert`).

Generate an [Internet Archive BookReader](https://github.com/internetarchive/bookreader) web viewer from an `img/` directory:

```sh
scanbook-cli web                                                      # current directory
scanbook-cli web ./my-book --title "My Book" --author "Jane Doe" --year 2014
```

- `[directory]` — optional positional argument (default: current directory).
- `-t/--title`, `-a/--author`, `-y/--year`, `-i/--info` — optional metadata (default empty).
- Reads `<dir>/img/*.png` in filename order, reads each image's real pixel dimensions, and writes `index.html` + `index.js` next to `img/`, overwriting any existing ones.
- The viewer references a sibling `../_Reader/` library and a parent `../index.html`, so the output directory is meant to sit inside that layout.

Export the pages of a scanned PDF or DjVu document to image files:

```sh
scanbook-cli export-page --input ./my-book.djvu                        # → ./my-book/page-*.png
scanbook-cli export-page --input ./my-book.pdf --output ./pages --format tiff
```

- `-i, --input` — source `.pdf` or `.djvu` file (required).
- `-o, --output` — output directory; **must not already exist** (default: the input name without its extension).
- `-f, --format` — image format to write (default `png`; PDF pages are rendered at 300 DPI).

Impose a directory of scanned pages into printable booklet-signature PDFs:

```sh
scanbook-cli print-pdf --input ./book-pages --output my-book           # → my-book-01.pdf, my-book-02.pdf, …
scanbook-cli print-pdf --input ./book-pages --output my-book --blank 2
```

- `-i, --input` — directory of scanned pages (required).
- `-o, --output` — output book name; one PDF per 32-page signature is written as `<name>-NN.pdf` (required).
- `-f, --format` — page-image extension to read (default `png`).
- `-b, --blank` — number of leading blank pages to insert before the first page (default `0`).
- Pages are grouped into 32-page signatures and reordered for folding into saddle-stitched booklets; a short final signature is padded with blank A5 pages.

Check that the external tools are installed and resolvable:

```sh
scanbook-cli doctor
```

- Takes no flags. Reports `OK`/`MISSING` for each tool — ImageMagick `convert`, DjVuLibre `ddjvu`, K2PdfOpt `k2pdfopt`, PdfTkServer `pdftk`, CPDF `cpdf` — and exits non-zero if any is missing.

## Configuration

Optional. `~/.config/cli-tools/config.yml` maps external tool names to executables (used by `scanbook-cli`, and optionally to locate `typst`):

```yml
Typst:
  typst: /usr/bin/typst
PdfTkServer:
  pdftk: /usr/bin/pdftk
```

If the file is absent the tools still run; a command that needs a specific tool reports a clear error only at that point.

### PDF rendering (`Pdf:` section)

Optional overrides for `build --format pdf`. Every key is optional; anything you omit keeps the built-in default (A5 page, 12pt body, 16pt for Chinese/Arabic/Hebrew/Korean/Japanese, binding-aware A5 margins, and the bundled font stack). A whole missing `Pdf:` section changes nothing.

```yml
Pdf:
  paper: a5            # any Typst paper name (a4, us-letter, ...)
  size: 12pt           # base body font size
  sizeLarge: 16pt      # body size for Chinese/Arabic/Hebrew/Korean/Japanese scripts
  margin:              # any of top/bottom/inside/outside/left/right; unset sides → 1.5cm
    inside: 1.8cm      # binding-relative edges (follow text direction), recommended
    outside: 1.4cm
    top: 1.7cm
    bottom: 2cm
  font:                # ordered family list; replaces the default stack entirely
    - Gentium
    - Amiri
    - Ezra SIL
    - AR PL UMing
    - Baekmuk Batang
    - Noto Sans
```

Sizes and margins are Typst lengths (`pt`, `mm`, `cm`, `in`, `em`); a malformed value fails the build with a clear message. `ebook-cli doctor` verifies every family listed under `font:` is one Typst can actually see.

## Development

```sh
task build     # cross-compile all commands (linux + windows)
task test      # run unit tests for each command
task package   # build the Docker image
```

## Releases

Merging to `main` runs [`.github/workflows/release.yml`](.github/workflows/release.yml): it computes a **CalVer** tag `vYYYY.M.MICRO` (micro auto-increments within the month), builds the Linux/Windows binaries + a GitHub release via GoReleaser, and pushes `ghcr.io/dpurge/cli-tools:<version>` (and `:latest`). The same version is stamped into `--version`.
