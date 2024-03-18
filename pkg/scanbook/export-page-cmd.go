package scanbook

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/config"
	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/spf13/cobra"
)

var exportPageCmd = &cobra.Command{
	Use:   "export-page",
	Short: "Export pages from scanned PDF or DjVu documents to image files",
	Long: `Export pages from scanned book to image files.
Scanned book can be a DjVu or PDF file.

Default format for the scanned pages is PNG.

Example 1:

	to-img --file=./my-book.djvu`,
	Run: exportPage,
}

func init() {
	mainCmd.AddCommand(exportPageCmd)

	exportPageCmd.Flags().StringVarP(&_input, "input", "i", "", "input file")
	exportPageCmd.MarkFlagRequired("input")

	exportPageCmd.Flags().StringVarP(&_output, "output", "o", "", "output directory (must not exist)")
	exportPageCmd.Flags().StringVarP(&_format, "format", "f", "png", "output file format (default: png)")
}

func exportPage(cmd *cobra.Command, args []string) {
	var dirname = ""
	filename, err := filepath.Abs(_input)
	if err != nil {
		log.Fatal(err)
	}

	if !tool.FileExists(filename) {
		log.Fatalf("input file does not exist: %s", filename)
	}

	var extension = filepath.Ext(filename)

	if _output != "" {
		dirname, err = filepath.Abs(_output)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		dirname = filename[0 : len(filename)-len(extension)]
	}

	if tool.DirectoryExists(dirname) {
		log.Fatalf("output directory already exist: %s", dirname)
	}

	if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	var pages []string

	switch extension {
	case ".djvu":
		pages, err = exportPageDjvu(filename, dirname)
	default:
		log.Fatalf("unsupported extension: %s", extension)
	}
	if err != nil {
		log.Fatal(err)
	}

	var page string
	for i := 0; i < len(pages); i++ {
		if page, err = tool.ConvertPageFormat(pages[i], fmt.Sprintf(".%s", _format)); err != nil {
			log.Fatal(err)
		} else {
			log.Println(page)
			err = os.Remove(pages[i])
			if err != nil {
				log.Fatalf("cannot remove file '%s': %s", pages[i], err)
			}
		}
	}
}

func exportPageDjvu(input string, output string) ([]string, error) {
	var pages []string

	ddjvu, err := config.GetToolPath("DjVuLibre", "ddjvu")
	if err != nil {
		return pages, err
	}

	cmd := exec.Command(ddjvu, "-format=tiff", "-eachpage", input, filepath.Join(output, "page-%03d.tiff"))
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return pages, err
	}
	if len(buf) > 0 {
		log.Println(string(buf[:]))
	}

	pages, err = tool.GetScanPages(output, ".tiff")
	if err != nil {
		log.Fatal(err)
	}

	return pages, nil
}

// func exportPagePdf(input string, output string) ([]string, error) {
// 	var pages []string
// 	return pages, nil
// }
