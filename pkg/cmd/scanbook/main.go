package scanbook

import (
	"os"

	"github.com/dpurge/cli-tools/pkg/cfg"
	"github.com/dpurge/cli-tools/pkg/cmd"
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use:   "scanbook-cli",
	Short: "ScanBook CLI short description",
	Long:  "ScanBook CLI long description",
}

func Execute() {
	cfg.ReadConfig()
	err := mainCmd.Execute()
	if err != nil {
		os.Exit(cmd.ExitCodeError)
	}
}

func init() {
}
