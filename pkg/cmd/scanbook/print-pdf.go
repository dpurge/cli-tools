package scanbook

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/cfg"
	"github.com/spf13/cobra"
)

var name string
var directory string
var extension string
var leadingBlank int

var printPdfCmd = &cobra.Command{
	Use:   "print-pdf",
	Short: "Create printable PDF signatures from a directory of scanned pages",
	Long: `Read scanned book pages from a directory, group them in signatures,
reorder pages for printing and convert to PDF.

Default format for the scanned pages is PNG.

Example 1:

	print-pdf --name=my-book --directory=./my-book
	
Example 2:

	print-pdf --name=my-book --directory=./book-pages --extension=.png`,
	Run: createPdfSignatures,
}

func init() {
	mainCmd.AddCommand(printPdfCmd)

	printPdfCmd.Flags().StringVarP(&name, "name", "n", "", "set book name")
	printPdfCmd.MarkFlagRequired("name")

	printPdfCmd.Flags().StringVarP(&directory, "directory", "d", "", "set directory with scanned pages")
	printPdfCmd.MarkFlagRequired("directory")

	printPdfCmd.Flags().StringVarP(&extension, "extension", "e", ".png", "set extension for scanned pages")
	printPdfCmd.Flags().IntVarP(&leadingBlank, "leading-blank", "b", 0, "set number of leading blank pages")
}

func createPdfSignatures(cmd *cobra.Command, args []string) {
	blank, err := createBlankPage("420x595") // A5 at 72 DPI
	if err != nil {
		log.Fatal(err)
	}

	blanks := make([]string, leadingBlank)
	for i := 0; i < leadingBlank; i++ {
		blanks[i] = blank
	}

	pages, err := getScannedPages(directory, extension)
	if err != nil {
		log.Fatal(err)
	}

	pages = append(blanks, pages...)

	lenSignature := 32 // 4 pages * 8 sheets
	lenPages := len(pages)

	signatureNr := 0
	for i := 0; i < lenPages; i += lenSignature {
		signature := make([]string, 0, lenSignature)
		signatureNr += 1

		j := i + lenSignature
		if j > lenPages {
			j = lenPages
		}
		signature = pages[i:j]
		for len(signature) < lenSignature {
			signature = append(signature, blank)
		}

		signatureName := fmt.Sprintf("%s-%02d", name, signatureNr)
		signatureFile, err := createPdfSignature(signatureName, signature)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(signatureFile)
	}

	err = os.Remove(blank)
	if err != nil {
		fmt.Println(err)
	}
}

func getScannedPages(directory string, extension string) ([]string, error) {
	items, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	pages := make([]string, 0, len(items))

	for _, item := range items {
		if !item.IsDir() && filepath.Ext(item.Name()) == extension {
			fullname, err := filepath.Abs(filepath.Join(directory, item.Name()))
			if err != nil {
				return nil, err
			}
			pages = append(pages, fullname)
		}
	}

	return pages, nil
}

func createBlankPage(size string) (string, error) {
	magickConvert, err := cfg.GetToolPath("ImageMagick", "convert")
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp(".", "*.png")
	if err != nil {
		return "", err
	}
	tmpFile.Close()

	blank, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		return "", err
	}

	cmd := exec.Command(magickConvert, "-size", size, "canvas:white", blank)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	if len(buf) > 0 {
		log.Println(string(buf[:]))
	}

	return blank, nil
}

func createPdfSignature(name string, signature []string) (string, error) {
	lenSignature := len(signature)
	if lenSignature%4 != 0 {
		log.Fatalf("Number of pages in the signature (%d) in not divisible by 4!", lenSignature)
	}

	printSignature := make([]string, 0, lenSignature)

	for i := 0; i < lenSignature/2; i += 2 {
		printSignature = append(printSignature, signature[lenSignature-i-1], signature[i], signature[i+1], signature[lenSignature-i-2])
	}

	filename, err := createPdf(name, printSignature)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func createPdf(name string, pages []string) (string, error) {
	filename, err := filepath.Abs(name + ".pdf")
	if err != nil {
		return "", err
	}

	magickConvert, err := cfg.GetToolPath("ImageMagick", "convert")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(magickConvert, append(pages, filename)...)
	buf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	if len(buf) > 0 {
		log.Println(string(buf[:]))
	}

	return filename, nil
}
