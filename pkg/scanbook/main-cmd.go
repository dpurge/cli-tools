package scanbook

import (
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var _input, _output, _format string

var mainCmd = &cobra.Command{
	Use:   "scanbook-cli",
	Short: "ScanBook CLI short description",
	Long:  "ScanBook CLI long description",
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
