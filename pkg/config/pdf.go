package config

import "github.com/spf13/viper"

// PdfConfig holds the optional PDF-render overrides from the `Pdf` config
// section. Every field is optional; empty means "not configured" and book.typ's
// defaults apply.
type PdfConfig struct {
	Paper     string    // page size, any Typst paper name (e.g. "a5", "a4")
	Size      string    // base body font size, a Typst length (e.g. "12pt")
	SizeLarge string    // enlarged body size for CJK/Arabic/Hebrew/Korean/Japanese
	Margin    PdfMargin // per-side page margins; unset sides keep the default
	Font      []string  // ordered font family list; replaces the default stack
}

// PdfMargin holds per-side page-margin overrides. inside/outside are the
// binding-relative edges (mapped to left/right by text direction); left/right
// set fixed edges. Empty sides get the exporter's fallback (typstMarginDict).
type PdfMargin struct {
	Top     string
	Bottom  string
	Left    string
	Right   string
	Inside  string
	Outside string
}

// GetPdfConfig reads the optional `Pdf` config section; missing keys yield zero
// values, so an absent section produces an all-empty struct and no overrides.
func GetPdfConfig() PdfConfig {
	return PdfConfig{
		Paper:     viper.GetString("Pdf.paper"),
		Size:      viper.GetString("Pdf.size"),
		SizeLarge: viper.GetString("Pdf.sizeLarge"),
		Margin: PdfMargin{
			Top:     viper.GetString("Pdf.margin.top"),
			Bottom:  viper.GetString("Pdf.margin.bottom"),
			Left:    viper.GetString("Pdf.margin.left"),
			Right:   viper.GetString("Pdf.margin.right"),
			Inside:  viper.GetString("Pdf.margin.inside"),
			Outside: viper.GetString("Pdf.margin.outside"),
		},
		Font: viper.GetStringSlice("Pdf.font"),
	}
}
