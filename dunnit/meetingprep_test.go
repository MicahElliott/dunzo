package dun

import "testing"

func TestNormalizeTag(t *testing.T) {
	cases := []struct{ in, want string }{
		{"jeff", "#jeff"},
		{"#jeff", "#jeff"},
		{"  boss  ", "#boss"},
		{"", ""},
		{"  ", ""},
	}
	for _, c := range cases {
		if got := normalizeTag(c.in); got != c.want {
			t.Errorf("normalizeTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
