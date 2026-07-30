package markdown

import "testing"

// TestParseMarkerAttrs covers the attribute grammar per SPECS §4 / §11.3.
// Table categories: happy-path (valid attrs), edge (boundary/CRLF/bare),
// error (malformed/unknown). The test is white-box (package markdown) so it
// can access the unexported parseMarkerAttrs and blockAttrs directly.
func TestParseMarkerAttrs(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		blockName string
		wantErr   bool
		wantAttrs blockAttrs
	}{
		// ---------- happy-path ----------
		{
			name:      "script unquoted",
			line:      "{start-text script=arab}",
			blockName: "text",
			wantAttrs: blockAttrs{Script: "arab"},
		},
		{
			name:      "script quoted",
			line:      `{start-text script="arab"}`,
			blockName: "text",
			wantAttrs: blockAttrs{Script: "arab"},
		},
		{
			name:      "lang and script",
			line:      "{start-text lang=arb script=arab}",
			blockName: "text",
			wantAttrs: blockAttrs{Lang: "arb", Script: "arab"},
		},
		{
			name:      "as and system with spaces",
			line:      `{start-text as=transcription system="DIN 31635"}`,
			blockName: "text",
			wantAttrs: blockAttrs{As: "transcription", System: "DIN 31635"},
		},
		{
			name:      "all four valid as values - source",
			line:      "{start-text as=source}",
			blockName: "text",
			wantAttrs: blockAttrs{As: "source"},
		},
		{
			name:      "all four valid as values - translation",
			line:      "{start-text as=translation}",
			blockName: "text",
			wantAttrs: blockAttrs{As: "translation"},
		},
		{
			name:      "all four valid as values - grammar",
			line:      "{start-text as=grammar}",
			blockName: "text",
			wantAttrs: blockAttrs{As: "grammar"},
		},
		{
			name:      "non-text block with lang and script",
			line:      "{start-vocabulary lang=arb script=arab}",
			blockName: "vocabulary",
			wantAttrs: blockAttrs{Lang: "arb", Script: "arab"},
		},
		// ---------- edge ----------
		{
			name:      "bare marker — no attrs",
			line:      "{start-text}",
			blockName: "text",
			wantAttrs: blockAttrs{},
		},
		{
			name:      "trailing \\r stripped (CRLF belt-and-suspenders)",
			line:      "{start-text script=arab}\r",
			blockName: "text",
			wantAttrs: blockAttrs{Script: "arab"},
		},
		{
			name:      "trailing \\r\\n stripped",
			line:      "{start-text script=arab}\r\n",
			blockName: "text",
			wantAttrs: blockAttrs{Script: "arab"},
		},
		{
			name:      "unknown script is NOT an error (LTR fallback, OI-6)",
			line:      "{start-text script=khmr}",
			blockName: "text",
			wantErr:   false,
			wantAttrs: blockAttrs{Script: "khmr"},
		},
		// ---------- error ----------
		{
			name:      "unknown key errors",
			line:      "{start-text foo=bar}",
			blockName: "text",
			wantErr:   true,
		},
		{
			name:      "malformed attr — no equals sign",
			line:      "{start-text noequalssign}",
			blockName: "text",
			wantErr:   true,
		},
		{
			name:      "unterminated quoted value",
			line:      `{start-text system="DIN 31635}`,
			blockName: "text",
			wantErr:   true,
		},
		{
			name:      "unknown as value errors",
			line:      "{start-text as=bogus}",
			blockName: "text",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			attrs, err := parseMarkerAttrs([]byte(tc.line), tc.blockName)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.line)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.line, err)
			}
			if attrs != tc.wantAttrs {
				t.Fatalf("parseMarkerAttrs(%q, %q)\n got: %+v\nwant: %+v", tc.line, tc.blockName, attrs, tc.wantAttrs)
			}
		})
	}
}

// TestIsRtlScript and TestBlockDirection verify the RTL-set helpers.
func TestIsRtlScript(t *testing.T) {
	rtl := []string{"arab", "hebr", "syrc"}
	ltr := []string{"latn", "cyrl", "hans", "hant", "kore", "jpan", "armn", "geor", "mong", "khmr", ""}

	for _, s := range rtl {
		if !isRtlScript(s) {
			t.Errorf("isRtlScript(%q) = false, want true", s)
		}
		if blockDirection(s) != "rtl" {
			t.Errorf("blockDirection(%q) = %q, want rtl", s, blockDirection(s))
		}
	}
	for _, s := range ltr {
		if isRtlScript(s) {
			t.Errorf("isRtlScript(%q) = true, want false", s)
		}
		if blockDirection(s) != "ltr" {
			t.Errorf("blockDirection(%q) = %q, want ltr", s, blockDirection(s))
		}
	}
}
