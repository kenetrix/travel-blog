package domain

import (
	"slices"
	"testing"
)

func TestTripCountries(t *testing.T) {
	stops := func(countries ...string) []Location {
		out := make([]Location, len(countries))
		for i, c := range countries {
			out[i] = Location{CountryID: c}
		}
		return out
	}
	cases := []struct {
		name  string
		stops []Location
		want  []string
	}{
		{"no stops", nil, []string{}},
		{"domestic", stops("DEU", "DEU", "DEU"), []string{"DEU"}},
		{"across borders", stops("AUT", "SVK", "HUN", "AUT"), []string{"AUT", "SVK", "HUN"}},
		{"country revisited later", stops("ITA", "FRA", "ITA", "CHE"), []string{"ITA", "FRA", "CHE"}},
	}
	for _, c := range cases {
		if got := (Trip{Stops: c.stops}).Countries(); !slices.Equal(got, c.want) {
			t.Errorf("%s: Countries() = %v, want %v", c.name, got, c.want)
		}
	}
}
