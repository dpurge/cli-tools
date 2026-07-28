package config

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// TestGetPdfConfigEmpty confirms that with nothing configured every field is
// zero-valued, so an absent config file / `Pdf` section yields no overrides.
func TestGetPdfConfigEmpty(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	got := GetPdfConfig()
	if !reflect.DeepEqual(got, PdfConfig{}) {
		t.Errorf("GetPdfConfig() with no config = %+v, want zero value", got)
	}
}

// TestGetPdfConfigValues confirms each key is read, including the mixed-case
// `sizeLarge` (viper lookups are case-insensitive), nested margin keys, and
// the font sequence.
func TestGetPdfConfigValues(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	viper.Set("Pdf.paper", "a4")
	viper.Set("Pdf.size", "12pt")
	viper.Set("Pdf.sizeLarge", "16pt")
	viper.Set("Pdf.margin.top", "2cm")
	viper.Set("Pdf.margin.left", "1.5cm")
	viper.Set("Pdf.margin.inside", "1.8cm")
	viper.Set("Pdf.font", []string{"Amiri", "Noto Sans"})

	got := GetPdfConfig()
	want := PdfConfig{
		Paper:     "a4",
		Size:      "12pt",
		SizeLarge: "16pt",
		Margin:    PdfMargin{Top: "2cm", Left: "1.5cm", Inside: "1.8cm"},
		Font:      []string{"Amiri", "Noto Sans"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetPdfConfig() = %+v, want %+v", got, want)
	}
}
