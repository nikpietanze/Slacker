package source

import "testing"

func TestParseBadge(t *testing.T) {
	for in, want := range map[string]int{"": 0, "3": 3, "•": 0, "99+": 99, " 12 ": 12, "-1": 0} {
		if got := parseBadge(in); got != want {
			t.Errorf("parseBadge(%q) = %d, want %d", in, got, want)
		}
	}
}

// Real output from macOS 27.2 (lsappinfo -all info -only StatusLabel Slack)
// and the NULL record the plain form returns there.
func TestParseStatusLabel(t *testing.T) {
	cases := map[string]struct {
		label string
		ok    bool
	}{
		`{ "LSASN"=ASN:0x0-0x18bfdbe5:, "StatusLabel"={ "label"="•" } }`: {"•", true},
		`{ "LSASN"=ASN:0x0-0x18bfdbe5:, "StatusLabel"={ "label"="3" } }`: {"3", true},
		`"StatusLabel"={ "label"="" }`:                                   {"", true},
		"[ NULL ]  ASN:0x0-0x18bfdbe5: \n    bundleID=[ NULL ] \n":       {"", false},
	}
	for in, want := range cases {
		got, ok := parseStatusLabel([]byte(in))
		if got != want.label || ok != want.ok {
			t.Errorf("parseStatusLabel(%q) = %q,%v; want %q,%v", in, got, ok, want.label, want.ok)
		}
	}
}
