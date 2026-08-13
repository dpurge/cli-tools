# Context

## Business context

Personal tooling for producing private e-book / language-learning materials. The
repository builds three command-line binaries used to turn hand-authored Markdown
project sources into distributable e-books, scanned-page references, and flashcard
decks.

## Target state

- `ebook-cli` — build an e-book project (`ebook.yml` + Markdown chapters) into
  **EPUB**, **PDF** (via Typst), and **MDX** (for a Docusaurus-style site), with
  first-class support for multi-script/RTL content (Arabic, Hebrew, Syriac) and
  per-script font roles.
- `scanbook-cli` — utilities around scanned-page PDFs (export pages, print, serve a
  web viewer).
- `flashcard-cli` — flashcard deck tooling (work in progress).

## Users

Single maintainer, run locally or via prebuilt Linux/Windows binaries and a
published container image (`ghcr.io/dpurge/cli-tools`).
</content>
