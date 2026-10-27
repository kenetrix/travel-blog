package domain

import "time"

// Trip is an ordered list of stops, inside one country or across several.
type Trip struct {
	ID         string
	Title      string
	StartsOn   string // YYYY-MM-DD
	EndsOn     string // YYYY-MM-DD
	Notes      string
	Visibility Visibility
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Stops      []Location
}

// TripSummary is a trip without notes and stops, with the countries of its stops.
type TripSummary struct {
	ID         string
	Title      string
	StartsOn   string
	EndsOn     string
	Visibility Visibility
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Countries  []string
	StopCount  int
}

// Countries returns the distinct countries of the stops in the order they are first reached.
func (t Trip) Countries() []string {
	return TripCountries(t.Stops)
}

func TripCountries(stops []Location) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range stops {
		if !seen[s.CountryID] {
			seen[s.CountryID] = true
			out = append(out, s.CountryID)
		}
	}
	return out
}
