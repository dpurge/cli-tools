package ebook

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var _formats []string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build ebook project",
	Long:  "Build long description",
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve every requested format to its Exporter, and reject any
		// unknown format, BEFORE reading the project or exporting anything
		// (SPECS §9/FR-8): a typo in a later --format must not leave a
		// partially-built earlier format behind.
		exporters := make([]Exporter, 0, len(_formats))
		for _, format := range _formats {
			exporter, err := exporterFor(format)
			if err != nil {
				log.Fatal(err)
			}
			exporters = append(exporters, exporter)
		}

		project, err := readProject(_project)
		if err != nil {
			log.Fatal(err)
		}

		for _, exporter := range exporters {
			outfile, err := exporter.Export(project)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(outfile)
		}
	},
}

// exporterFor maps a --format value to its Exporter, or reports it as
// unknown (SPECS §9).
func exporterFor(format string) (Exporter, error) {
	switch format {
	case "epub":
		return epubExporter{}, nil
	case "pdf":
		return typstExporter{}, nil
	case "mdx":
		return mdxExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (want epub|pdf|mdx)", format)
	}
}

func init() {
	mainCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&_project, "project", "p", "ebook.yml", "eBook project file")
	buildCmd.Flags().StringSliceVarP(&_formats, "format", "f", []string{"epub"}, "output format(s): epub, pdf, mdx (repeatable, or comma-separated)")
}
