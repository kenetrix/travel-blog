package domain

import "testing"

func TestNormalizeFlightNumber(t *testing.T) {
	valid := map[string]string{
		"LH400":    "LH400",
		"lh0400":   "LH400",
		"LH 400":   "LH400",
		"lh-400":   "LH400",
		" nh 204 ": "NH204",
		"dlh 400":  "DLH400",
		"U2 1234":  "U21234",
		"4U 9":     "4U9",
		"BA 117A":  "BA117A",
	}
	for in, want := range valid {
		got, ok := NormalizeFlightNumber(in)
		if !ok || got != want {
			t.Errorf("NormalizeFlightNumber(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}

	for _, in := range []string{"", "400", "L 400", "LH", "LH 40000", "12 345", "LH 0", "LH4OO", "LH/400"} {
		if got, ok := NormalizeFlightNumber(in); ok {
			t.Errorf("NormalizeFlightNumber(%q) = %q; want invalid", in, got)
		}
	}
}
