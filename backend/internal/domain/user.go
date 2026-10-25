package domain

import "time"

// UserID identifies a user; all travel data belongs to exactly one.
type UserID string

type Profile struct {
	Username    string // public name, lowercase
	Email       string // empty if none is set
	FirstName   string
	LastName    string
	DisplayName string
	Bio         string
	Language    string // BCP 47, e.g. "de" or "en-US"
	Timezone    string // IANA, e.g. "Europe/Berlin"
	HomeCountry string // ISO 3166 alpha-3 or empty
	Color       string // #rrggbb or empty
	CreatedAt   time.Time
}

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID      UserID
	Profile Profile
	Role    Role
	// MustChangePassword: the password is generated or set by an
	// administrator and must be changed
	MustChangePassword bool
}

type ProfileUpdate struct {
	Username    *string
	Email       *string
	FirstName   *string
	LastName    *string
	DisplayName *string
	Bio         *string
	Language    *string
	Timezone    *string
	HomeCountry *string
	Color       *string
}

type DistanceUnit string

const (
	Kilometers DistanceUnit = "km"
	Miles      DistanceUnit = "mi"
)

type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

type View string

const (
	ViewCountries View = "countries"
	ViewFlights   View = "flights"
)

type Settings struct {
	DistanceUnit DistanceUnit
	Theme        Theme
	DefaultView  View
	// DefaultVisibility applies to new entries, flights and locations
	DefaultVisibility Visibility
	// ProfileVisibility public makes profile and public data readable by everyone
	ProfileVisibility Visibility
}

// DefaultSettings apply to new users (like the column defaults).
func DefaultSettings() Settings {
	return Settings{
		DistanceUnit:      Kilometers,
		Theme:             ThemeSystem,
		DefaultView:       ViewCountries,
		DefaultVisibility: Private,
		ProfileVisibility: Private,
	}
}

// Session is a login; only the hash of the token is stored.
type Session struct {
	TokenHash string
	UserID    UserID
	CreatedAt time.Time
	ExpiresAt time.Time
}
