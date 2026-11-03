package extract

import "strings"

// entityReplacements are applied in this order, as in the Rust version.
var entityReplacements = []struct{ old, new string }{
	{"&#x2f;", "/"},
	{"&#x3a;", ":"},
	{"&#x3d;", "="},
	{"&#x3f;", "?"},
	{"&#x26;", "&"},
	{"&amp;", "&"},
	{"&lt;", "<"},
	{"&gt;", ">"},
	{"&quot;", "\""},
}

// DecodeEntities turns the HTML entities that the security pages use into
// their characters.
func DecodeEntities(s string) string {
	for _, r := range entityReplacements {
		s = strings.ReplaceAll(s, r.old, r.new)
	}
	return s
}
