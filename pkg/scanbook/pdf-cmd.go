package scanbook

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/spf13/cobra"
)

var pdfCmd = &cobra.Command{
	Use:   "pdf",
	Short: "Combine a directory of scanned pages into a single PDF",
	Long: `Read scanned book pages from a directory and combine them, in sorted
order, into a single PDF.

Default format for the scanned pages is PNG.

Example 1:

	pdf --input ./book-pages --output my-book

Example 2:

	pdf --input ./book-pages --output my-book --format png`,
	Run: convertToPdf,
}

func init() {
	mainCmd.AddCommand(pdfCmd)

	pdfCmd.Flags().StringVarP(&_input, "input", "i", "", "input directory with scanned pages")
	pdfCmd.MarkFlagRequired("input")

	pdfCmd.Flags().StringVarP(&_output, "output", "o", "", "output book name")
	pdfCmd.MarkFlagRequired("output")

	pdfCmd.Flags().StringVarP(&_format, "format", "f", "png", "format of scanned pages")
}

func convertToPdf(cmd *cobra.Command, args []string) {
	out, err := convertPagesToPdf(_input, _output, _format)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(out)
}

// convertPagesToPdf combines <input>/*.<format> (sorted) into a single PDF
// derived from output, returning the output PDF path.
func convertPagesToPdf(input, output, format string) (string, error) {
	if !tool.DirectoryExists(input) {
		return "", fmt.Errorf("input directory does not exist: %s", input)
	}

	pages, err := tool.GetScanPages(input, "."+format)
	if err != nil {
		return "", err
	}
	if len(pages) == 0 {
		return "", fmt.Errorf("no .%s pages found in %s", format, input)
	}

	return tool.ConvertImagesToPdf(pdfOutputPath(output), pages)
}

// pdfOutputPath appends .pdf to output unless it already ends in .pdf (any
// case), so an explicit extension is never doubled.
func pdfOutputPath(output string) string {
	if strings.EqualFold(filepath.Ext(output), ".pdf") {
		return output
	}
	return output + ".pdf"
}
