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

	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
