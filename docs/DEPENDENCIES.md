# Dependencies

Single Go module (`go.mod`); all three binaries share these dependencies.

## Direct

| Module | Version | Used for |
|---|---|---|
| `github.com/go-shiori/go-epub` | v1.2.1 | EPUB generation (`pkg/ebook/epub.go`) |
| `github.com/spf13/cobra` | v1.10.1 | CLI commands, all `pkg/<tool>` packages |
| `github.com/spf13/viper` | v1.21.0 | Config loading (`pkg/config`) |
| `github.com/yuin/goldmark` | v1.8.4 | Markdown parsing, extended by `pkg/tool/markdown` |
| `golang.org/x/net` | v0.46.0 | — |
| `gopkg.in/yaml.v3` | v3.0.1 | Project/config YAML |

## Indirect (notable)

| Module | Version | Pulled in by |
|---|---|---|
| `github.com/gabriel-vasile/mimetype` | v1.4.11 | go-epub |
| `github.com/gofrs/uuid/v5` | v5.4.0 | go-epub |
| `github.com/vincent-petithory/dataurl` | v1.0.0 | go-epub |
| `go.yaml.in/yaml/v3` | v3.0.4 | viper |
| `github.com/fsnotify/fsnotify` | v1.9.0 | viper |
| `github.com/spf13/afero`, `cast`, `pflag` | — | viper/cobra |
| `github.com/pelletier/go-toml/v2` | v2.2.4 | viper |
| `golang.org/x/sys`, `golang.org/x/text` | — | transitive |

## External (non-Go) runtime dependency

- **`typst`** binary must be on `PATH` (or configured via `Typst.typst`) for
  PDF export (`ebook-cli build -f pdf`). The published container image bundles
  it; local dev must install it separately.

See `go.mod` for the authoritative, exact version list — this table summarizes
it and should be regenerated (or spot-checked) after any `go get`/`go mod
tidy`.
</content>
