package flashcard

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Doctor Cmd short description",
	Long:  "Doctor Cmd long description",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("doctor called")
	},
}

func init() {
	mainCmd.AddCommand(doctorCmd)
}
