package tool

import "testing"

func TestGetHtmlTitlePrefersSectionTitle(t *testing.T) {
	doc := `<h1 id="generated-from-file-heading">Generated heading</h1>
<div class="block-marker"><span class="ct-badge">S</span></div>
<div class="section" dir="ltr">
<h1 class="section-title">Title inside start-section</h1>
</div>`

	got, err := GetHtmlTitle(doc)
	if err != nil {
		t.Fatalf("GetHtmlTitle() error = %v", err)
	}
	if got != "Title inside start-section" {
		t.Fatalf("GetHtmlTitle() = %q, want section-title", got)
	}
}

func TestGetHtmlTitleFallsBackToFirstH1(t *testing.T) {
	got, err := GetHtmlTitle(`<h1>Plain chapter title</h1>`)
	if err != nil {
		t.Fatalf("GetHtmlTitle() error = %v", err)
	}
	if got != "Plain chapter title" {
		t.Fatalf("GetHtmlTitle() = %q", got)
	}
}
