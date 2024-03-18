package tool

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dpurge/cli-tools/pkg/config"
)

func GetScanPages(directory string, extension string) ([]string, error) {
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

func ConvertPageFormat(input string, extension string) (string, error) {
	if filepath.Ext(input) == extension {
		return input, nil
	}

	magickConvert, err := config.GetToolPath("ImageMagick", "convert")
	if err != nil {
		return "", err
	}

	var ext = filepath.Ext(input)
	var output = input[0:len(input)-len(ext)] + extension
	if FileExists(output) {
		log.Printf("scanpage already exists: %s", output)
	} else {
		cmd := exec.Command(magickConvert, input, output)
		buf, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		if len(buf) > 0 {
			log.Println(string(buf[:]))
		}
	}

	return output, nil
}
