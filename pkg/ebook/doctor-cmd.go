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
		typstUsable := false
		if err != nil {
			healthy = false
			fmt.Fprintln(os.Stderr, "ERR  typst  not found: needed for `build --format pdf` "+
				"(install https://typst.app, or set Typst.typst in the config); "+
				"EPUB and MDX export are unaffected")
		} else if out, runErr := runTypst(path, "--version"); runErr != nil {
			healthy = false
			fmt.Fprintf(os.Stderr, "ERR  typst  found at %s but not runnable: %v\n", path, runErr)
		} else {
			typstUsable = true
			fmt.Printf("OK   typst  %s (%s)\n", path, strings.TrimSpace(out))
		}

		// Verify every font family named in the optional `Pdf.font` config is
		// one Typst can actually see (`typst fonts`) — the same view the PDF
		// exporter renders against. With no fonts configured there is nothing
		// to check and the built-in default stack (book.typ) is used.
		if fonts := config.GetPdfConfig().Font; len(fonts) > 0 {
			var families map[string]bool
			var ferr error
			if !typstUsable {
				ferr = fmt.Errorf("typst unavailable")
			} else {
				families, ferr = typstFontFamilies(path)
			}
			if ferr != nil {
				healthy = false
				fmt.Fprintf(os.Stderr, "ERR  font   cannot verify configured fonts: %v\n", ferr)
			} else {
				for _, f := range fonts {
					if families[strings.ToLower(strings.TrimSpace(f))] {
						fmt.Printf("OK   font   %s\n", f)
					} else {
						healthy = false
						fmt.Fprintf(os.Stderr, "ERR  font   %q not found (not listed by `typst fonts`)\n", f)
					}
				}
			}
		}

		if !healthy {
			os.Exit(config.ExitCodeError)
		}
	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
