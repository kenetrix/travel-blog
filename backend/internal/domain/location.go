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
	Rating     int    // 0 for none, else 1 to 5
	Category   string // one of StopCategories, or empty
	Links      []StopLink
	Visibility Visibility
}

type StopLink struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

var StopCategories = []string{"sight", "food", "stay", "activity", "nature", "transport", "other"}
