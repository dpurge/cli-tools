package scanbook

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- buildSpreads: cover alone, then pairs, trailing odd alone -------------

func TestBuildSpreads(t *testing.T) {
	img := func(n int) webImage { return webImage{Width: 1, Height: 1, URI: string(rune('a' + n))} }

	// Build n images a,b,c,...
	imgs := func(n int) []webImage {
		s := make([]webImage, n)
		for i := range s {
			s[i] = img(i)
		}
		return s
	}

	tests := []struct {
		n    int
		want [][]string // URIs per spread
	}{
		{0, nil},
		{1, [][]string{{"a"}}},
		{2, [][]string{{"a"}, {"b"}}},
		{3, [][]string{{"a"}, {"b", "c"}}},
		{4, [][]string{{"a"}, {"b", "c"}, {"d"}}},
		{5, [][]string{{"a"}, {"b", "c"}, {"d", "e"}}},
		{6, [][]string{{"a"}, {"b", "c"}, {"d", "e"}, {"f"}}},
	}

	for _, tt := range tests {
		got := buildSpreads(imgs(tt.n))
		if len(got) != len(tt.want) {
			t.Errorf("n=%d: got %d spreads, want %d: %+v", tt.n, len(got), len(tt.want), got)
			continue
		}
		for i := range tt.want {
			if len(got[i]) != len(tt.want[i]) {
				t.Errorf("n=%d spread %d: got %d imgs, want %d", tt.n, i, len(got[i]), len(tt.want[i]))
				continue
			}
			for j := range tt.want[i] {
				if got[i][j].URI != tt.want[i][j] {
					t.Errorf("n=%d spread %d img %d: got %q, want %q", tt.n, i, j, got[i][j].URI, tt.want[i][j])
				}
			}
		}
	}
}

// --- jsString: single-quote JS escaping ------------------------------------

func TestJSString(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"", ""},
		{`a'b`, `a\'b`},
		{`a\b`, `a\\b`},
		{"a\nb", `a\nb`},
		{"a\rb", `a\rb`},
		{"a\tb", `a\tb`},
		{"café Датский 你好", "café Датский 你好"}, // unicode passes through unescaped
	}
	for _, tc := range cases {
		if got := jsString(tc.in); got != tc.want {
			t.Errorf("jsString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- renderWebJS: structure + escaping -------------------------------------

func TestRenderWebJS(t *testing.T) {
	model := webModel{
		Title:     "T'it", // apostrophe must be escaped
		Author:    "Auth",
		Year:      "2014",
		Info:      "",
		Thumbnail: "img/page-0000.png",
		Spreads: [][]webImage{
			{{Width: 100, Height: 200, URI: "img/page-0000.png"}},
			{{Width: 100, Height: 200, URI: "img/page-0001.png"}, {Width: 100, Height: 200, URI: "img/page-0002.png"}},
		},
	}
	out, err := renderWebJS(model)
	if err != nil {
		t.Fatalf("renderWebJS error = %v", err)
	}

	for _, want := range []string{
		"function instantiateBookReader(selector, extraOptions)",
		"{ width: 100, height: 200, uri: 'img/page-0000.png' }",
		"{ width: 100, height: 200, uri: 'img/page-0001.png' },", // comma between paired images
		"thumbnail: 'img/page-0000.png'",
		`{label: 'Title', value: 'T\'it'}`, // apostrophe escaped
		`{label: 'Author', value: 'Auth'}`,
		`{label: 'Year', value: '2014'}`,
		`{label: 'Info', value: ''}`,
		"imagesBaseURL: '../_Reader/images/'",
		"ui: 'full'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderWebJS output missing %q\n---\n%s", want, out)
		}
	}
}

// --- renderWebHTML: title escaping + verbatim static markup ----------------

func TestRenderWebHTML(t *testing.T) {
	out, err := renderWebHTML(webModel{Title: `A & B <x>`})
	if err != nil {
		t.Fatalf("renderWebHTML error = %v", err)
	}
	if !strings.Contains(out, "<title>A &amp; B &lt;x&gt;</title>") {
		t.Errorf("title not HTML-escaped as expected:\n%s", out)
	}
	// Static markup (incl. comments and the _Reader refs) must be verbatim.
	for _, want := range []string{
		"<!-- JS dependencies -->",
		`<script src="../_Reader/BookReader.js"></script>`,
		`<script>instantiateBookReader('#BookReader')</script>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderWebHTML missing verbatim %q", want)
		}
	}
}

func TestRenderWebHTMLEmptyTitle(t *testing.T) {
	out, err := renderWebHTML(webModel{Title: ""})
	if err != nil {
		t.Fatalf("renderWebHTML error = %v", err)
	}
	if !strings.Contains(out, "<title></title>") {
		t.Error("empty title should render <title></title>")
	}
}

// --- imageSize: reads real header dimensions -------------------------------

func TestImageSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.png")
	writePNG(t, path, 123, 456)

	w, h, err := imageSize(path)
	if err != nil {
		t.Fatalf("imageSize error = %v", err)
	}
	if w != 123 || h != 456 {
		t.Errorf("imageSize = %dx%d, want 123x456", w, h)
	}
}

// --- generateViewer: end to end against a temp img/ ------------------------

func TestGenerateViewer(t *testing.T) {
	dir := t.TempDir()
	imgDir := filepath.Join(dir, "img")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Three pages with distinct sizes so per-image dimensions are exercised.
	writePNG(t, filepath.Join(imgDir, "page-0000.png"), 10, 20)
	writePNG(t, filepath.Join(imgDir, "page-0001.png"), 11, 21)
	writePNG(t, filepath.Join(imgDir, "page-0002.png"), 12, 22)

	htmlPath, jsPath, err := generateViewer(dir, "My Book", "Me", "2020", "note")
	if err != nil {
		t.Fatalf("generateViewer error = %v", err)
	}
	if htmlPath != filepath.Join(dir, "index.html") || jsPath != filepath.Join(dir, "index.js") {
		t.Errorf("unexpected output paths: %q, %q", htmlPath, jsPath)
	}

	js, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatal(err)
	}
	jsStr := string(js)
	// Cover alone (page-0000, 10x20), then a pair (page-0001, page-0002).
	for _, want := range []string{
		"{ width: 10, height: 20, uri: 'img/page-0000.png' }",
		"{ width: 11, height: 21, uri: 'img/page-0001.png' },",
		"{ width: 12, height: 22, uri: 'img/page-0002.png' }",
		"thumbnail: 'img/page-0000.png'",
		`{label: 'Title', value: 'My Book'}`,
		`{label: 'Info', value: 'note'}`,
	} {
		if !strings.Contains(jsStr, want) {
			t.Errorf("index.js missing %q", want)
		}
	}

	html, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "<title>My Book</title>") {
		t.Error("index.html title mismatch")
	}
}

func TestGenerateViewerMissingImgDir(t *testing.T) {
	dir := t.TempDir() // no img/ subdir
	if _, _, err := generateViewer(dir, "", "", "", ""); err == nil {
		t.Error("expected an error when img/ is missing, got nil")
	}
}

func TestGenerateViewerEmptyImgDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := generateViewer(dir, "", "", "", ""); err == nil {
		t.Error("expected an error when img/ has no .png files, got nil")
	}
}

// writePNG writes a solid w x h PNG to path, failing the test on error.
func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	m.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}
