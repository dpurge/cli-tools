package scanbook

import (
	"errors"
	"log"
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Doctor Cmd short description",
	Long:  "Doctor Cmd long description",
	Run: func(cmd *cobra.Command, args []string) {

		magickConvert, err := config.GetToolPath("ImageMagick", "convert")
		if err == nil {
			log.Println("ImageMagick convert: ", magickConvert)
		} else if errors.Is(err, os.ErrNotExist) {
			log.Fatal("Missing ImageMagick convert: ", magickConvert, " (install: https://imagemagick.org/script/download.php)")
		} else {
			log.Fatal(err)
		}

		ddjvu, err := config.GetToolPath("DjVuLibre", "ddjvu")
		if err == nil {
			log.Println("DjVuLibre ddjvu: ", ddjvu)
		} else if errors.Is(err, os.ErrNotExist) {
			log.Fatal("Missing DjVuLibre ddjvu: ", ddjvu, " (install: https://sourceforge.net/projects/djvu/files/)")
		} else {
			log.Fatal(err)
		}

		k2pdfopt, err := config.GetToolPath("K2PdfOpt", "k2pdfopt")
		if err == nil {
			log.Println("K2PdfOpt command: ", k2pdfopt)
		} else if errors.Is(err, os.ErrNotExist) {
			log.Fatal("Missing K2PdfOpt command: ", k2pdfopt, " (install: https://www.willus.com/k2pdfopt/)")
		} else {
			log.Fatal(err)
		}

		pdftk, err := config.GetToolPath("PdfTkServer", "pdftk")
		if err == nil {
			log.Println("PdfTk command: ", pdftk)
		} else if errors.Is(err, os.ErrNotExist) {
			log.Fatal("Missing PdfTk command: ", pdftk, " (install: https://www.pdflabs.com/tools/pdftk-server/)")
		} else {
			log.Fatal(err)
		}

	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
