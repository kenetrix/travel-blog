package domain

type Location struct {
	ID         string
	CountryID  string
	Name       string
	Lat        float64
	Lng        float64
	VisitedAt  string
	Notes      string
	Visibility Visibility
}
