package flashcard

import (
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use:   "flashcard-cli",
	Short: "FlashCard CLI short description",
	Long:  "FlashCard CLI long description",
}

func Execute() {
	config.ReadConfig()
	err := mainCmd.Execute()
	if err != nil {
		os.Exit(config.ExitCodeError)
	}
}

func init() {
}
