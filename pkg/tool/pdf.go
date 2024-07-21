package tool

import (
	"bufio"
	"strconv"

	// "log"
	// "os"
	// "path/filepath"
	"strings"
)

type pdfPageRect struct {
	StartX float64
	StartY float64
	EndX   float64
	EndY   float64
}

type pdfPageDimensions struct {
	Width  float64
	Height float64
}

type PdfPage struct {
	Number     int
	Rotation   int
	Rectangle  pdfPageRect
	Dimensions pdfPageDimensions
}

func getValue(line string, key string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, key))
}

func GetPdfPages(document string) ([]PdfPage, error) {

	var pages []PdfPage
	i := -1

	output, err := RunCmd("PdfTkServer", "pdftk", document, "dump_data")
	if err != nil {
		return nil, err
	}

	if len(output) > 0 {
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "NumberOfPages:") {
				v := getValue(line, "NumberOfPages:")

				n, err := strconv.Atoi(v)
				if err != nil {
					return nil, err
				}

				pages = make([]PdfPage, n)
			}

			if line == "PageMediaBegin" {
				i++
			}

			if strings.HasPrefix(line, "PageMediaNumber:") {
				v := getValue(line, "PageMediaNumber:")

				n, err := strconv.Atoi(v)
				if err != nil {
					return nil, err
				}

				pages[i].Number = n
			}

			if strings.HasPrefix(line, "PageMediaRotation:") {
				v := getValue(line, "PageMediaRotation:")

				n, err := strconv.Atoi(v)
				if err != nil {
					return nil, err
				}

				pages[i].Rotation = n
			}

			if strings.HasPrefix(line, "PageMediaRect:") {
				v := getValue(line, "PageMediaRect:")
				var f []string = strings.Fields(v)
				var rect pdfPageRect

				rect.StartX, err = strconv.ParseFloat(f[0], 64)
				if err != nil {
					return nil, err
				}

				rect.StartY, err = strconv.ParseFloat(f[1], 64)
				if err != nil {
					return nil, err
				}

				rect.EndX, err = strconv.ParseFloat(f[2], 64)
				if err != nil {
					return nil, err
				}

				rect.EndY, err = strconv.ParseFloat(f[3], 64)
				if err != nil {
					return nil, err
				}

				pages[i].Rectangle = rect
			}

			if strings.HasPrefix(line, "PageMediaDimensions:") {
				v := getValue(line, "PageMediaDimensions:")
				var f []string = strings.Fields(v)
				var dimentions pdfPageDimensions

				dimentions.Width, err = strconv.ParseFloat(f[0], 64)
				if err != nil {
					return nil, err
				}

				dimentions.Height, err = strconv.ParseFloat(f[1], 64)
				if err != nil {
					return nil, err
				}

				pages[i].Dimensions = dimentions
			}
		}
	}

	return pages, nil
}
