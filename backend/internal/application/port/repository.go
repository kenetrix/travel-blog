package port

import (
	"errors"
	"time"

	"travel-blog/backend/internal/domain"
)

// ErrConflict is returned by a repository when a unique value (e.g. email
// or username) is already taken.
var ErrConflict = errors.New("conflict")

// Repositories for travel data only work on the data of one user;
// the user is passed explicitly on every call.

type CountryEntryRepository interface {
	List(user domain.UserID) ([]domain.CountryEntry, error)
	ListPublic(user domain.UserID) ([]domain.CountryEntry, error)
	// Get returns nil if there is no entry.
	Get(user domain.UserID, iso string) (*domain.CountryEntry, error)
	Upsert(user domain.UserID, e domain.CountryEntry) error
	Delete(user domain.UserID, iso string) error
}

// LocationRepository gives access to single stops; trips write them (TripRepository).
type LocationRepository interface {
	// Get returns nil if the user has no such location.
	Get(user domain.UserID, id string) (*domain.Location, error)
}

type TripRepository interface {
	// List returns all trips of the user, oldest first.
	List(user domain.UserID) ([]domain.TripSummary, error)
	// Get returns nil if the user has no such trip; stops are in visiting order.
	Get(user domain.UserID, id string) (*domain.Trip, error)
	// Create stores a trip with its stops.
	Create(user domain.UserID, t domain.Trip) error
	// Replace overwrites a trip and its stops in one transaction: stops with a known
	// id are updated, others created, and missing ones deleted. It returns the storage
	// keys of the photos of deleted stops, and found=false if the user has no such trip.
	Replace(user domain.UserID, t domain.Trip) (found bool, removedPhotos []string, err error)
	// Delete removes a trip with its stops and returns the storage keys of their photos.
	Delete(user domain.UserID, id string) (found bool, removedPhotos []string, err error)
}

// PhotoRepository checks ownership via the location of the photo.
type PhotoRepository interface {
	ListByLocation(user domain.UserID, locationID string) ([]domain.Photo, error)
	// Get returns nil if the photo does not belong to the user.
	Get(user domain.UserID, id string) (*domain.Photo, error)
	Create(p domain.Photo) error
	Delete(user domain.UserID, id string) (bool, error)
}

type FlightRepository interface {
	List(user domain.UserID) ([]domain.Flight, error)
	ListPublic(user domain.UserID) ([]domain.Flight, error)
	Create(user domain.UserID, f domain.Flight) error
	// Update and Delete return false if the user has no such flight.
	Update(user domain.UserID, id string, u domain.FlightUpdate) (bool, error)
	Delete(user domain.UserID, id string) (bool, error)
	RouteExists(user domain.UserID, from, to string) (bool, error)
}

type UserRepository interface {
	// Create creates user and default settings; ErrConflict if the
	// email or username is taken.
	Create(u domain.User, passwordHash string) error
	// Get returns nil if there is no such user.
	Get(id domain.UserID) (*domain.User, error)
	// GetByUsername returns nil if there is no such user.
	GetByUsername(username string) (*domain.User, error)
	// GetByEmail returns nil if no account has this (lowercase) email.
	GetByEmail(email string) (*domain.User, error)
	// GetByIdentity returns the account of an OIDC identity or nil.
	GetByIdentity(issuer, subject string) (*domain.User, error)
	// LinkIdentity assigns an OIDC identity to an account; ErrConflict
	// if it is already taken.
	LinkIdentity(issuer, subject string, id domain.UserID) error
	CreateWithIdentity(u domain.User, issuer, subject string) error
	// Credentials returns user and password hash for a username;
	// ok is false if there is no account with a password.
	Credentials(username string) (id domain.UserID, passwordHash string, ok bool, err error)
	List() ([]domain.User, error)
	CountAdmins() (int, error)
	SetRole(id domain.UserID, role domain.Role) error
	SetMustChangePassword(id domain.UserID, must bool) error
	PasswordHash(id domain.UserID) (string, error)
	SetPasswordHash(id domain.UserID, hash string) error
	// UpdateProfile writes the whole profile; ErrConflict as with Create.
	UpdateProfile(id domain.UserID, p domain.Profile) error
	Settings(id domain.UserID) (domain.Settings, error)
	SaveSettings(id domain.UserID, s domain.Settings) error
	// Delete removes the account with its settings, sessions and identities (via
	// foreign keys), not the travel data.
	Delete(id domain.UserID) error
}

// UserDataRepository holds the travel data of the users, in a database of its own.
type UserDataRepository interface {
	// DeleteAll removes the user's countries, flights, trips, stops and photo records.
	DeleteAll(user domain.UserID) error
}

type SessionRepository interface {
	Create(s domain.Session) error
	// Get returns nil if there is no such session.
	Get(tokenHash string) (*domain.Session, error)
	Delete(tokenHash string) error
	DeleteOthers(user domain.UserID, keep string) error
	DeleteExpired(now time.Time) error
}

type AppSettingsRepository interface {
	RegistrationOpen() (bool, error)
	SetRegistrationOpen(open bool) error
}

// OIDCSettingsRepository stores the admin-area OIDC configuration as key/value in the
// configuration database.
type OIDCSettingsRepository interface {
	// All returns the stored values (empty if nothing is stored).
	All() (map[string]string, error)
	Replace(values map[string]string) error
	Clear() error
}
