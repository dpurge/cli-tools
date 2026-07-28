package scanbook

import (
	"log"
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

// scanbookTools are the external tools scanbook-cli depends on, each with a
// download URL surfaced when missing.
var scanbookTools = []struct {
	group   string
	bin     string
	install string
}{
	{"ImageMagick", "convert", "https://imagemagick.org/script/download.php"},
	{"DjVuLibre", "ddjvu", "https://sourceforge.net/projects/djvu/files/"},
	{"K2PdfOpt", "k2pdfopt", "https://www.willus.com/k2pdfopt/"},
	{"PdfTkServer", "pdftk", "https://www.pdflabs.com/tools/pdftk-server/"},
	{"CPDF", "cpdf", "https://github.com/coherentgraphics/cpdf-binaries/releases/tag/v2.7"},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check external tools required by scanbook-cli",
	Long:  "Check the external tools scanbook-cli depends on, reporting every one that is missing.",
	Run: func(cmd *cobra.Command, args []string) {
		// Report all missing tools in one run rather than aborting at the first.
		healthy := true
		for _, t := range scanbookTools {
			path, err := config.GetToolPath(t.group, t.bin)
			if err == nil {
				log.Printf("OK       %s %s: %s", t.group, t.bin, path)
				continue
			}
			healthy = false
			log.Printf("MISSING  %s %s (install: %s): %v", t.group, t.bin, t.install, err)
		}

		if !healthy {
			os.Exit(config.ExitCodeError)
		}
	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
