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

var _blank int

var printPdfCmd = &cobra.Command{
	Use:   "print-pdf",
	Short: "Create printable PDF signatures from a directory of scanned pages",
	Long: `Read scanned book pages from a directory, group them in signatures,
reorder pages for printing and convert to PDF.

Default format for the scanned pages is PNG.

Example 1:

	print-pdf --input ./book-pages --output my-book
	
Example 2:

	print-pdf --input ./book-pages --output my-book --format png`,
	Run: createPdfSignatures,
}

func init() {
	mainCmd.AddCommand(printPdfCmd)

	printPdfCmd.Flags().StringVarP(&_output, "output", "o", "", "output book name")
	printPdfCmd.MarkFlagRequired("output")

	printPdfCmd.Flags().StringVarP(&_input, "input", "i", "", "input directory with scanned pages")
	printPdfCmd.MarkFlagRequired("input")

	printPdfCmd.Flags().StringVarP(&_format, "format", "f", "png", "format of scanned pages")
	printPdfCmd.Flags().IntVarP(&_blank, "blank", "b", 0, "number of leading blank pages")
}

func createPdfSignatures(cmd *cobra.Command, args []string) {
	blank, err := createBlankPage("420x595") // A5 at 72 DPI
	if err != nil {
		log.Fatal(err)
	}

	blanks := make([]string, _blank)
	for i := 0; i < _blank; i++ {
		blanks[i] = blank
	}

	pages, err := tool.GetScanPages(_input, fmt.Sprintf(".%s", _format))
	if err != nil {
		log.Fatal(err)
	}

	pages = append(blanks, pages...)

	lenSignature := 32 // 4 pages * 8 sheets
	lenPages := len(pages)

	signatureNr := 0
	for i := 0; i < lenPages; i += lenSignature {
		var signature []string
		signatureNr += 1

		j := i + lenSignature
		if j > lenPages {
			j = lenPages
		}
		signature = pages[i:j]
		for len(signature) < lenSignature {
			signature = append(signature, blank)
		}

		signatureName := fmt.Sprintf("%s-%02d", _output, signatureNr)
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

func createBlankPage(size string) (string, error) {
	magickConvert, err := config.GetToolPath("ImageMagick", "convert")
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

	magickConvert, err := config.GetToolPath("ImageMagick", "convert")
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
