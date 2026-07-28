package ebook

import (
	"fmt"
	"os"
	"strings"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check external tools required by ebook-cli",
	Long: "Check the external tools ebook-cli depends on.\n\n" +
		"Currently that is Typst, required by `build --format pdf`; EPUB and " +
		"MDX export use no external tools.",
	Run: func(cmd *cobra.Command, args []string) {
		healthy := true

		// Typst is resolved exactly as the PDF exporter resolves it (the
		// configured Typst.typst path, else PATH) and then actually run, so
		// `doctor` reports what `build --format pdf` would really find and use.
		path, err := locateTypst()
		if err != nil {
			healthy = false
			fmt.Fprintln(os.Stderr, "ERR  typst  not found: needed for `build --format pdf` "+
				"(install https://typst.app, or set Typst.typst in the config); "+
				"EPUB and MDX export are unaffected")
		} else if out, runErr := runTypst(path, "--version"); runErr != nil {
			healthy = false
			fmt.Fprintf(os.Stderr, "ERR  typst  found at %s but not runnable: %v\n", path, runErr)
		} else {
			fmt.Printf("OK   typst  %s (%s)\n", path, strings.TrimSpace(out))
		}

		if !healthy {
			os.Exit(config.ExitCodeError)
		}
	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
