package ebook

import (
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use:   "ebook-cli",
	Short: "EBook CLI short description",
	Long:  "EBook CLI long description",
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
