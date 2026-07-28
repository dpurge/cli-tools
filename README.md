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

Chapters are CommonMark/GFM markdown plus custom blocks — `{start-vocabulary}`, `{start-dialog}`, `{start-parallel}` — rendered natively into each output format.

## Configuration

Optional. `~/.config/cli-tools/config.yml` maps external tool names to executables (used by `scanbook-cli`, and optionally to locate `typst`):

```yml
Typst:
  typst: /usr/bin/typst
PdfTkServer:
  pdftk: /usr/bin/pdftk
```

If the file is absent the tools still run; a command that needs a specific tool reports a clear error only at that point.

## Development

```sh
task build     # cross-compile all commands (linux + windows)
task test      # run unit tests for each command
task package   # build the Docker image
```

## Releases

Merging to `main` runs [`.github/workflows/release.yml`](.github/workflows/release.yml): it computes a **CalVer** tag `vYYYY.M.MICRO` (micro auto-increments within the month), builds the Linux/Windows binaries + a GitHub release via GoReleaser, and pushes `ghcr.io/dpurge/cli-tools:<version>` (and `:latest`). The same version is stamped into `--version`.
