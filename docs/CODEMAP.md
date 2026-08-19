# Code map

81 Go files, ~16.5K lines. Three binaries (`cmd/*`) over shared packages
(`pkg/*`). See [ARCHITECTURE.md](ARCHITECTURE.md) for how these fit together.

## `cmd/` — binary entrypoints

- `cmd/ebook/main.go` → `pkg/ebook.Execute()`
- `cmd/scanbook/main.go` → `pkg/scanbook.Execute()`
- `cmd/flashcard/main.go` → `pkg/flashcard.Execute()`

## `pkg/ebook/` — ebook-cli (primary tool)

| File | Purpose |
|---|---|
| `main-cmd.go` | Root Cobra command, `Execute()` |
| `build-cmd.go` | `build` subcommand — load project, dispatch to exporters |
| `doctor-cmd.go` | `doctor` subcommand — environment checks |
| `vocab-cmd.go` | `vocab` subcommand — vocabulary CSV export |
| `project.go` | `EBookProject` load/model (`ebook.yml`) |
| `exporter.go` | `Exporter` interface, `ProjectItem`/`WalkTexts`, `baseOutputName` |
| `epub.go` | EPUB exporter (`go-epub`) |
| `typst.go` | PDF exporter — generates Typst source, shells out to `typst` |
| `mdx.go` | MDX exporter (Docusaurus-style chapter files + `_category_.json`) |
| `vocabulary.go` | Vocabulary block → CSV |
| `translations.go` | `as=` role resolution (source/transcription/translation/grammar) |
| `templates/book.typ` | Typst template: cover, title page, `#textblock()` |
| `*_test.go` | Table-driven tests per exporter; `typst_gate_test.go` compiles Typst to verify show-rule gating |

## `pkg/tool/markdown/` — custom Goldmark extension

Parses/renders the project's `{start-X}/{end-X}` block markers.

| File | Purpose |
|---|---|
| `extension.go` | Goldmark extension registration |
| `parser.go`, `marker.go` | Block marker parsing (`{start-vocabulary ...}` etc.) |
| `ast.go` | Custom AST node kinds — one per block type; a new block type needs a `NodeKind` registered in all 3 renderers or it panics |
| `attr.go` | Marker attribute parsing (`lang=`, `script=`, `as=`) |
| `renderer.go` | HTML (EPUB) renderer |
| `typst_render.go`, `typst_escape.go` | Typst (PDF) renderer |
| `mdx_render.go`, `mdx_escape.go` | MDX renderer |
| `interlinear.go` | Parallel-text alignment |
| `linktarget.go` | Cross-block link targets |
| `*_test.go` | One file per block type / edge case (dialog, questions, models, vocabulary, parallel, parallel-dialog, text, CRLF, idempotency, named bug regressions) |

## `pkg/config/` — shared configuration

`main.go` (Viper setup), `pdf.go` (Typst/PDF tool config), `tool.go` (external
tool path resolution), `exitCode.go` (process exit codes).

## `pkg/tool/` — cross-tool helpers

`command.go` (shell exec), `filesystem.go`, `escape.go`, `html.go`,
`scanpage.go` (shared with `pkg/scanbook`), `pdf.go`.

## `pkg/scanbook/` — scanbook-cli

`main-cmd.go`, `doctor-cmd.go`, `pdf-cmd.go`, `print-pdf-cmd.go`,
`export-page-cmd.go`, `web-cmd.go` (+ `templates/index.html.tmpl`,
`index.js.tmpl` for the local viewer).

## `pkg/flashcard/` — flashcard-cli (WIP)

`main-cmd.go`, `build-cmd.go`, `doctor-cmd.go`, `project.go`.

## `pkg/types/`, `pkg/version/`

`flashcard.go` (shared `Flashcard` type), `version.go` (build version string).
</content>
