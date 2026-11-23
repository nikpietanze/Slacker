package source

import "testing"

func TestParseBadge(t *testing.T) {
	for in, want := range map[string]int{"": 0, "3": 3, "•": 0, "99+": 99, " 12 ": 12, "-1": 0} {
		if got := parseBadge(in); got != want {
			t.Errorf("parseBadge(%q) = %d, want %d", in, got, want)
		}
	}
}
