package domain

// Visibility determines who may see entries, flights and locations.
type Visibility string

const (
	// Private is seen by the user only.
	Private Visibility = "private"
	// Public is seen by everyone, provided the user's profile is public.
	Public Visibility = "public"
)

func (v Visibility) Valid() bool {
	return v == Private || v == Public
}
