package ebook

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var projectFile string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build short description",
	Long:  "Build long description",
	Run: func(cmd *cobra.Command, args []string) {
		epubFile, err := buildEPub(projectFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(epubFile)
	},
}

func init() {
	mainCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&projectFile, "filepath", "f", "./ebook.yml", "path to the eBook project")
	buildCmd.MarkFlagRequired("filepath")
}
