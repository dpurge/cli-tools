# Tech stack

- **Language**: Go 1.25.
- **CLI framework**: `spf13/cobra` (commands), `spf13/viper` (config loading,
  `pkg/config`).
- **Markdown**: `yuin/goldmark` (CommonMark/GFM) + a custom in-repo extension
  (`pkg/tool/markdown`) for the project's block-marker syntax.
- **EPUB generation**: `go-shiori/go-epub`.
- **PDF generation**: external `typst` binary (not a Go dependency) — this
  repo generates `.typ` source (`pkg/ebook/templates/book.typ` +
  `typst_render.go`) and shells out to compile it.
- **YAML**: `gopkg.in/yaml.v3` / `go.yaml.in/yaml/v3` (project files, config).
- **Build**: `go-task/task` (`Taskfile.yml`); release/container packaging via
  `goreleaser` (`.goreleaser.yml`), publishing to `ghcr.io/dpurge/cli-tools`.
- **Testing**: standard library `testing`, no external test framework.
- **Dev environment**: VS Code devcontainer (`golang.go`,
  `DavidAnson.vscode-markdownlint` extensions).
</content>
