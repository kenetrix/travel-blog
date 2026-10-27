package domain

// Location is a stop of a trip; Position is its place in the visiting order.
type Location struct {
	ID         string
	TripID     string
	Position   int
	CountryID  string
	Name       string
	Lat        float64
	Lng        float64
	VisitedAt  string // YYYY-MM-DD, arrival
	Notes      string
	Visibility Visibility
}
