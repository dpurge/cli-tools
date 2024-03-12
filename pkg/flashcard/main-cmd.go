package flashcard

import (
	"os"

	"github.com/dpurge/cli-tools/pkg/cfg"
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use:   "flashcard-cli",
	Short: "FlashCard CLI short description",
	Long:  "FlashCard CLI long description",
}

func Execute() {
	cfg.ReadConfig()
	err := mainCmd.Execute()
	if err != nil {
		os.Exit(cfg.ExitCodeError)
	}
}

func init() {
}
