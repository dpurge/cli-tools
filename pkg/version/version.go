// Package version holds the build-time version string shared by every CLI.
package version

// Version is the CLI version reported by `<cmd> --version`. It defaults to
// "dev" for local/un-stamped builds and is overridden at build time via the
// linker:
//
//	go build -ldflags "-X github.com/dpurge/cli-tools/pkg/version.Version=<v>"
//
// The release pipeline stamps the CalVer version here — see .goreleaser.yml,
// Taskfile.yml, and .github/workflows/release.yml.
var Version = "dev"
