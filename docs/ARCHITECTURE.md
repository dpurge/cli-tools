# Architecture

## Workspace structure

Go module `github.com/dpurge/cli-tools`, three independent binaries sharing common
packages:

```
cmd/ebook/main.go       → pkg/ebook.Execute()      (ebook-cli)
cmd/scanbook/main.go    → pkg/scanbook.Execute()   (scanbook-cli)
cmd/flashcard/main.go   → pkg/flashcard.Execute()  (flashcard-cli)
```

Each `cmd/<tool>/main.go` is a thin entrypoint; all logic lives in the matching
`pkg/<tool>` package. Every tool package follows the same Cobra layout:
`main-cmd.go` (root command + `Execute()`), `<verb>-cmd.go` per subcommand
(`build-cmd.go`, `doctor-cmd.go`, `vocab-cmd.go`, ...).

## Modules

- **`pkg/ebook`** — the primary tool. Loads an `ebook.yml` project
  (`project.go`), then exports it via format-specific exporters implementing
  the shared `Exporter` interface (`exporter.go`): `epub.go` (EPUB via
  `go-epub`), `typst.go` (PDF via generated Typst source + `typst` binary,
  template in `templates/book.typ`), `mdx.go` (MDX for Docusaurus-style
  sites). `vocabulary.go` exports vocabulary blocks to CSV.
  `translations.go` handles the `as=` role system (source/transcription/
  translation/grammar).
- **`pkg/tool/markdown`** — custom Goldmark (CommonMark/GFM) extension. Parses
  the project's `{start-X}/{end-X}` block markers (vocabulary, models,
  questions, dialog, parallel, text) into AST nodes (`ast.go`, `marker.go`,
  `parser.go`) and renders each to HTML (EPUB), Typst (PDF), and MDX via
  dedicated renderers (`renderer.go`, `typst_render.go`, `mdx_render.go`).
  `interlinear.go` and `linktarget.go` support parallel-text and cross-block
  linking. Escaping is format-specific (`mdx_escape.go`, `typst_escape.go`).
- **`pkg/config`** — shared Viper-based config loading (`main.go`), PDF tool
  config (`pdf.go`), external tool resolution (`tool.go`), and process exit
  codes (`exitCode.go`).
- **`pkg/tool`** — cross-tool helpers: shell command execution
  (`command.go`), filesystem helpers, string escaping, HTML helpers, and a
  scanned-page PDF helper shared with `pkg/scanbook`.
- **`pkg/scanbook`** — scanned-page PDF utilities: export/print pages,
  serve a local web viewer (`web-cmd.go`, `templates/index.html.tmpl`).
- **`pkg/flashcard`** — flashcard project build (work in progress), shares
  the `pkg/types.Flashcard` type.
- **`pkg/types`**, **`pkg/version`** — small shared types and build version.

## Testing architecture

Standard Go `testing` package, table-driven tests colocated with source
(`<name>_test.go`). No mocking framework in use. Notable patterns:

- **`typst_gate_test.go`** — asserts Typst show-rule gating (strong/emph per
  script) rather than just string presence; see
  [[typst-strong-emph-gate]] memory.
- Golden/compile-style checks in `typst_test.go` / `typst_export_test.go`
  compile emitted Typst source rather than only asserting on strings — see
  [[typst-template-verify-by-compile]].
- `pkg/tool/markdown` has the heaviest test surface (one `_test.go` per
  block type / edge case: CRLF handling, idempotency, bug regressions).

## Build & release

- **Taskfile.yml** (go-task) — `task build` → `dist/{ebook,flashcard,scanbook}-cli`
  for linux + windows. PDF export shells out to a `typst` binary resolved via
  the shared config (`config.Typst.typst`), falling back to `PATH`.
- **`.goreleaser.yml`** — release build + container image
  (`ghcr.io/dpurge/cli-tools`), which bundles the `typst` binary since PDF
  export shells out to it.
- **`.devcontainer/`** — Go + markdownlint devcontainer for local dev.
</content>
