package markdown

// Unit tests for the marker.go helpers (SPECS FR-2, FR-3).
// These are in package markdown (not markdown_test) so that unexported
// functions — badgeOnlyTypst, badgeOnlyHTML — are accessible directly.

import "testing"

// TestBadgeOnlyTypst covers the standalone Typst badge block.
// Post-FR-2: the badge always renders on the LEFT regardless of the block's
// own text direction — no dir parameter, no RTL align(right) branch.
func TestBadgeOnlyTypst(t *testing.T) {
	tests := []struct {
		name   string
		letter string
		want   string
	}{
		{
			name:   "LTR emits inline bracket form",
			letter: "T",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n",
		},
		{
			name:   "RTL script: badge still emits left-side form (FR-2 always-left)",
			letter: "T",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"T\")]\n\n",
		},
		{
			name:   "RTL letter D: always left, no align(right)",
			letter: "D",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"D\")]\n\n",
		},
		{
			name:   "all five letters: M",
			letter: "M",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"M\")]\n\n",
		},
		{
			name:   "Q letter",
			letter: "Q",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"Q\")]\n\n",
		},
		{
			name:   "V letter",
			letter: "V",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"V\")]\n\n",
		},
		{
			name:   "P letter (parallel badge, FR-4)",
			letter: "P",
			want:   "#block(above: 1.2em, below: 0.5em)[#_ctbadge(\"P\")]\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := badgeOnlyTypst(tc.letter)
			if got != tc.want {
				t.Fatalf("badgeOnlyTypst(%q) = %q, want %q", tc.letter, got, tc.want)
			}
		})
	}
}

// TestBadgeOnlyHTML covers the standalone HTML badge div.
// Post-FR-2: the badge always renders on the LEFT — no dir parameter,
// no dir="rtl" attribute on the badge element itself.
func TestBadgeOnlyHTML(t *testing.T) {
	tests := []struct {
		name   string
		letter string
		want   string
	}{
		{
			name:   "LTR emits no dir attribute",
			letter: "V",
			want:   `<div class="block-marker"><span class="ct-badge">V</span></div>` + "\n",
		},
		{
			name:   "RTL script: badge still emits no dir attribute (FR-2 always-left)",
			letter: "T",
			want:   `<div class="block-marker"><span class="ct-badge">T</span></div>` + "\n",
		},
		{
			name:   "RTL letter M: no dir=rtl on badge (FR-2)",
			letter: "M",
			want:   `<div class="block-marker"><span class="ct-badge">M</span></div>` + "\n",
		},
		{
			name:   "D letter",
			letter: "D",
			want:   `<div class="block-marker"><span class="ct-badge">D</span></div>` + "\n",
		},
		{
			name:   "Q letter",
			letter: "Q",
			want:   `<div class="block-marker"><span class="ct-badge">Q</span></div>` + "\n",
		},
		{
			name:   "P letter (parallel badge, FR-4)",
			letter: "P",
			want:   `<div class="block-marker"><span class="ct-badge">P</span></div>` + "\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := badgeOnlyHTML(tc.letter)
			if got != tc.want {
				t.Fatalf("badgeOnlyHTML(%q) = %q, want %q", tc.letter, got, tc.want)
			}
		})
	}
}
