package ebook

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var vocabCmd = &cobra.Command{
	Use:   "vocab",
	Short: "Get vocabulary from ebook project",
	Long:  "Get vocabulary long description",
	Run: func(cmd *cobra.Command, args []string) {
		epubFile, err := buildVocabulary(_project)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(epubFile)
	},
}

func init() {
	mainCmd.AddCommand(vocabCmd)

	vocabCmd.Flags().StringVarP(&_project, "project", "p", "ebook.yml", "eBook project file")
}
