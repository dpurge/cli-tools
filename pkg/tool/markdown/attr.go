package markdown

import (
	"bytes"
	"fmt"
	"strings"
)

// blockAttrs holds the parsed attributes from a {start-X ...} marker line.
// All fields are empty strings when the corresponding attribute is absent.
type blockAttrs struct {
	Lang, Script, As, System string
}

// parseMarkerAttrs parses the attribute string from a {start-blockName ...}
// marker line. markerLine is the raw line bytes as returned by the goldmark
// reader (may include a trailing \r or \n; stripped automatically — ASR-2
// belt-and-suspenders). blockName is the expected block name ("vocabulary",
// "text", etc.) and is used in error messages.
//
// Grammar per SPECS §4.1:
//
//	marker   = "{start-" name { WS } [ attrlist ] { WS } "}" [ CR ] LF
//	attrlist = attr { WS attr }
//	attr     = key "=" value
//	key      = "lang" | "script" | "as" | "system"
//	value    = '"' {any-except-dquote} '"' | {any-except-WS-and-"}"}+
//
// Error cases (ASR-7):
//   - unknown key                                    → error
//   - malformed attr (no '=', unterminated quote)    → error
//   - unknown as value (not source/transcription/translation/grammar) → error
//   - unknown script                                 → NOT an error (LTR
//     fallback; isRtlScript handles it in M2)
func parseMarkerAttrs(markerLine []byte, blockName string) (blockAttrs, error) {
	// Tolerate trailing \r or \n (CRLF belt-and-suspenders, ASR-2).
	markerLine = bytes.TrimRight(markerLine, "\r\n")

	prefix := []byte("{start-" + blockName)
	if !bytes.HasPrefix(markerLine, prefix) {
		return blockAttrs{}, fmt.Errorf("not a {start-%s} marker: %q", blockName, markerLine)
	}

	// Extract the attribute section: everything after the prefix, stripping
	// trailing whitespace then the closing '}'.
	rest := markerLine[len(prefix):]
	rest = bytes.TrimRight(rest, " \t")
	if len(rest) > 0 && rest[len(rest)-1] == '}' {
		rest = rest[:len(rest)-1]
	}
	attrStr := strings.TrimSpace(string(rest))
	if attrStr == "" {
		return blockAttrs{}, nil // bare marker — no attributes
	}

	var attrs blockAttrs
	s := attrStr
	for s != "" {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			break
		}

		// Find the '=' separator for this attr.
		eq := strings.IndexByte(s, '=')
		if eq == -1 {
			return blockAttrs{}, fmt.Errorf("malformed attribute (no '=') near %q in {start-%s}", s, blockName)
		}

		key := strings.TrimRight(s[:eq], " \t")
		s = strings.TrimLeft(s[eq+1:], " \t")

		// Parse value: quoted (may contain spaces) or unquoted.
		var value string
		if len(s) > 0 && s[0] == '"' {
			end := strings.IndexByte(s[1:], '"')
			if end == -1 {
				return blockAttrs{}, fmt.Errorf("unterminated quoted value in {start-%s}: %q", blockName, attrStr)
			}
			value = s[1 : end+1]
			s = s[end+2:]
		} else {
			i := strings.IndexAny(s, " \t")
			if i == -1 {
				value = s
				s = ""
			} else {
				value = s[:i]
				s = s[i:]
			}
		}

		switch key {
		case "lang":
			attrs.Lang = value
		case "script":
			// Unknown script is NOT an error (OI-6 LTR-fallback rule).
			attrs.Script = value
		case "as":
			switch value {
			case "source", "transcription", "translation", "grammar":
				attrs.As = value
			default:
				return blockAttrs{}, fmt.Errorf("unknown as=%q in {start-%s}: must be source|transcription|translation|grammar", value, blockName)
			}
		case "system":
			attrs.System = value
		default:
			return blockAttrs{}, fmt.Errorf("unknown attribute %q on {start-%s}", key, blockName)
		}
	}
	return attrs, nil
}

// isRtlScript reports whether script (a lowercase ISO-15924 code) belongs to
// the RTL set {arab, hebr, syrc} per SPECS §5 and D9. Unknown scripts return
// false (LTR fallback), consistent with phraseforge's normalizeScript +
// getBodyDirection behavior (shared.tsx:151-161,192-215).
func isRtlScript(script string) bool {
	switch script {
	case "arab", "hebr", "syrc":
		return true
	}
	return false
}

// blockDirection returns "rtl" for RTL scripts and "ltr" for all others
// (including unknown scripts; OI-6 LTR-fallback rule). Defined here in M1;
// used by M2+ renderers when wiring per-block direction.
func blockDirection(script string) string {
	if isRtlScript(script) {
		return "rtl"
	}
	return "ltr"
}
