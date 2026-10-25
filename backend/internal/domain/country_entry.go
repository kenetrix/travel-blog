package domain

type CountryStatus string

const (
	StatusVisited  CountryStatus = "visited"
	StatusWishlist CountryStatus = "wishlist"
	// StatusNone is only valid as input and removes the entry.
	StatusNone CountryStatus = "none"
)

type CountryDetails struct {
	Food        string
	Sightseeing string
	Notes       string
	Rating      int // 0 = unrated
	Tags        []string
}

type CountryEntry struct {
	ISO        string // ISO 3166 alpha-3 code, e.g. "DEU"
	Status     CountryStatus
	Visibility Visibility
	Details    CountryDetails
}
