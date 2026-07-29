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

Chapters are CommonMark/GFM markdown plus custom blocks — `{start-vocabulary}`, `{start-dialog}`, `{start-parallel}`, `{start-models}`, `{start-questions}` — rendered natively into each output format.

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

#### Font roles (`font.css`)

`font.css` declares each font role as an `@font-face` whose `font-family` is `Font <Role>` and whose `src: local(...)` names the real installed family for that role — `Font Header`, `Font Body`, `Font Transcription`, `Font Translation`, and (for large scripts) `Font Strong`/`Font Emphasis`. On the EPUB side that name is used directly in CSS (e.g. `font-family: "Font Header", sans-serif`); on the PDF side the same `font.css` (read from `stylesheet.common`) is parsed and each role's `local()` name is prepended to the Typst font stack, so PDF and EPUB pick up the same fonts per role. A role with no matching `@font-face` falls back to a recommended installed family (`Noto Sans` for header, `Gentium` for body/translation/strong/emphasis, `DejaVu Sans` for transcription) so an incomplete `font.css` still renders sensibly.

`Font Strong`/`Font Emphasis` auto-activate only for "large scripts" — Arabic, Hebrew, CJK, Korean, Japanese, the same script set that triggers the enlarged PDF body size. Synthetic bold/italic renders poorly in these scripts, so bold (`**...**` / `<strong>`/`<b>`) and italic (`*...*` / `<em>`/`<i>`) switch to a distinct substitute font at *normal* weight/style instead of a synthetically thickened or slanted glyph; headings likewise keep `Font Header` but drop synthetic bold. Latin and other non-large scripts are unaffected — real bold/italic stays as-is. When `Font Strong`/`Font Emphasis` are undeclared, both fall back to the body font (`Gentium`), which simply drops the bold/italic distinction rather than inventing an unrelated one.

The ready-made large-script EPUB bundles (`epub-public`'s `src/css/main/{arab,hebr,cjk}/`) declare `Font Strong`/`Font Emphasis` and route `strong`/`b`/`em`/`i` (and `h1`–`h3`) to them at normal weight/style; `cjk` is the one shared bundle for Chinese/Japanese/Korean. The `latn` bundle keeps real bold/italic and declares no `Font Strong`/`Font Emphasis`.

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
