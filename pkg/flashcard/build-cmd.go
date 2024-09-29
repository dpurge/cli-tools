package flashcard

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build flashcard project",
	Long:  "Build flashard project",
	Run: func(cmd *cobra.Command, args []string) {
		// flashcardFile, err := buildAnkiPackage(_project)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println(flashcardFile)
		fmt.Println("Not implemented!")
	},
}

func init() {
	mainCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&_project, "project", "p", "flashcard.yml", "flashcard project file")
	// buildCmd.MarkFlagRequired("project")
}
