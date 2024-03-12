package scanbook

import (
	"errors"
	"log"
	"os"

	"github.com/dpurge/cli-tools/pkg/cfg"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Doctor Cmd short description",
	Long:  "Doctor Cmd long description",
	Run: func(cmd *cobra.Command, args []string) {
		magickConvert, err := cfg.GetToolPath("ImageMagick", "convert")
		if err == nil {
			log.Println("ImageMagick convert: ", magickConvert)
		} else if errors.Is(err, os.ErrNotExist) {
			log.Fatal("Missing ImageMagick convert: ", magickConvert, " (install: https://imagemagick.org/script/download.php)")
		} else {
			log.Fatal(err)
		}
	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
