package scanbook

import (
	_ "embed"
	"fmt"
	"html"
	"image"
	_ "image/png" // registers the PNG decoder for image.DecodeConfig
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dpurge/cli-tools/pkg/tool"
	"github.com/spf13/cobra"
)

// webHTMLTemplate / webJSTemplate are the BookReader viewer templates, rendered
// with text/template (values escaped by the htmlstr/jsstr funcs). text/template
// is used for the HTML too so the static markup is emitted verbatim —
// html/template would strip comments and rewrite some contexts.
//
//go:embed templates/index.html.tmpl
var webHTMLTemplate string

//go:embed templates/index.js.tmpl
var webJSTemplate string

// web-command flags; all optional, defaulting to "".
var _title, _author, _year, _info string

var webCmd = &cobra.Command{
	Use:   "web [directory]",
	Short: "Generate a BookReader web viewer (index.html + index.js) from an img/ directory",
	Long: `Generate an Internet Archive BookReader web viewer for a scanned book.

Reads the img/ subdirectory of the given directory (the current directory when
no directory is passed), pairs the pages into two-page spreads, and writes
index.html and index.js next to img/. Existing index.html/index.js are
overwritten.

Example:

	web
	web ./my-book --title "My Book" --author "Jane Doe" --year 2014`,
	Args: cobra.MaximumNArgs(1),
	Run:  generateWeb,
}

func init() {
	mainCmd.AddCommand(webCmd)

	webCmd.Flags().StringVarP(&_title, "title", "t", "", "book title (page title + metadata)")
	webCmd.Flags().StringVarP(&_author, "author", "a", "", "book author (metadata)")
	webCmd.Flags().StringVarP(&_year, "year", "y", "", "publication year (metadata)")
	webCmd.Flags().StringVarP(&_info, "info", "i", "", "extra info (metadata)")
}

// webImage is one page image in the viewer: its pixel dimensions and its
// viewer-relative URI ("img/<basename>").
type webImage struct {
	Width  int
	Height int
	URI    string
}

// webModel is the data the templates render: metadata plus spreads and thumbnail.
type webModel struct {
	Title     string
	Author    string
	Year      string
	Info      string
	Thumbnail string
	Spreads   [][]webImage
}

// generateWeb is the cobra entry point, delegating to generateViewer.
func generateWeb(cmd *cobra.Command, args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	htmlPath, jsPath, err := generateViewer(dir, _title, _author, _year, _info)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(htmlPath)
	log.Println(jsPath)
}

// generateViewer reads <dir>/img/*.png and writes <dir>/index.html and
// index.js (overwriting existing), returning their paths.
func generateViewer(dir, title, author, year, info string) (htmlPath, jsPath string, err error) {
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	imgDir := filepath.Join(dir, "img")
	if !tool.DirectoryExists(imgDir) {
		return "", "", fmt.Errorf("image directory does not exist: %s", imgDir)
	}

	pages, err := tool.GetScanPages(imgDir, ".png")
	if err != nil {
		return "", "", err
	}
	if len(pages) == 0 {
		return "", "", fmt.Errorf("no .png images found in %s", imgDir)
	}

	images := make([]webImage, len(pages))
	for i, p := range pages {
		w, h, err := imageSize(p)
		if err != nil {
			return "", "", fmt.Errorf("cannot read image %q: %w", p, err)
		}
		images[i] = webImage{Width: w, Height: h, URI: "img/" + filepath.Base(p)}
	}

	model := webModel{
		Title:     title,
		Author:    author,
		Year:      year,
		Info:      info,
		Thumbnail: images[0].URI,
		Spreads:   buildSpreads(images),
	}

	htmlOut, err := renderWebHTML(model)
	if err != nil {
		return "", "", err
	}
	jsOut, err := renderWebJS(model)
	if err != nil {
		return "", "", err
	}

	htmlPath = filepath.Join(dir, "index.html")
	jsPath = filepath.Join(dir, "index.js")
	if err := os.WriteFile(htmlPath, []byte(htmlOut), 0o644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(jsPath, []byte(jsOut), 0o644); err != nil {
		return "", "", err
	}

	return htmlPath, jsPath, nil
}

// imageSize returns an image's pixel width and height, reading only its header
// (image.DecodeConfig) rather than decoding the whole image.
func imageSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// buildSpreads groups images into BookReader spreads: a lone cover first, the
// rest paired left/right, and a trailing unpaired image alone.
func buildSpreads(images []webImage) [][]webImage {
	spreads := make([][]webImage, 0, len(images)/2+1)
	if len(images) == 0 {
		return spreads
	}

	spreads = append(spreads, []webImage{images[0]})
	for i := 1; i < len(images); i += 2 {
		if i+1 < len(images) {
			spreads = append(spreads, []webImage{images[i], images[i+1]})
		} else {
			spreads = append(spreads, []webImage{images[i]})
		}
	}
	return spreads
}

// renderWebHTML renders index.html; htmlstr HTML-escapes the title.
func renderWebHTML(model webModel) (string, error) {
	t, err := template.New("index.html").Funcs(template.FuncMap{"htmlstr": html.EscapeString}).Parse(webHTMLTemplate)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, model); err != nil {
		return "", err
	}
	return b.String(), nil
}

// renderWebJS renders index.js; jsstr JS-escapes every interpolated value.
func renderWebJS(model webModel) (string, error) {
	t, err := template.New("index.js").Funcs(template.FuncMap{"jsstr": jsString}).Parse(webJSTemplate)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, model); err != nil {
		return "", err
	}
	return b.String(), nil
}

// jsString escapes s for a single-quoted JS string literal via the shared
// tool.EscapeQuoted (surrounding quotes added by the template).
func jsString(s string) string {
	return tool.EscapeQuoted(s, '\'')
}
