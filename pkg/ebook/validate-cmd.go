package ebook

import (
	"fmt"
	"os"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/spf13/cobra"
)

var _validateRoot string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate that only declared languages, scripts, and grammar tags are used",
	Long: "Recursively finds every ebook.yml under --root and checks:\n" +
		"  - its top-level language/script/translation-language/translation-script fields\n" +
		"  - every content block's lang=/script= marker attribute\n" +
		"  - every {start-vocabulary} grammar-field tag\n" +
		"against the built-in catalog (pkg/catalog). Every unknown value found is " +
		"reported with its file and line number; the command exits non-zero if any " +
		"were found.",
	Run: func(cmd *cobra.Command, args []string) {
		errs, err := ValidateTree(_validateRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR ", err)
			os.Exit(config.ExitCodeError)
		}

		for _, e := range errs {
			fmt.Println(e.String())
		}

		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "\n%d unknown value(s) found\n", len(errs))
			os.Exit(config.ExitCodeError)
		}
		fmt.Println("OK — no unknown languages, scripts, or grammar tags found")
	},
}

func init() {
	mainCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVarP(&_validateRoot, "root", "r", ".", "root directory to scan recursively for ebook.yml projects")
}
