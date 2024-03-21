package ebook

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build ebook project",
	Long:  "Build long description",
	Run: func(cmd *cobra.Command, args []string) {
		epubFile, err := buildEPub(_project)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(epubFile)
	},
}

func init() {
	mainCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&_project, "project", "p", "ebook.yml", "eBook project file")
	// buildCmd.MarkFlagRequired("project")
}
