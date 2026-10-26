package service

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

// TravelService manages countries and flights of users and enforces the rules
// for them. Every method only works on the data of the given user.
type TravelService struct {
	entries port.CountryEntryRepository
	flights port.FlightRepository
	users   port.UserRepository
	now     func() time.Time
}

func NewTravelService(entries port.CountryEntryRepository, flights port.FlightRepository, users port.UserRepository) *TravelService {
	return &TravelService{entries: entries, flights: flights, users: users, now: time.Now}
}

func (s *TravelService) Entries(user domain.UserID) ([]domain.CountryEntry, error) {
	return s.entries.List(user)
}

// Flights returns all flights of the user (oldest first).
func (s *TravelService) Flights(user domain.UserID) ([]domain.Flight, error) {
	return s.flights.List(user)
}

// Travel returns all entries and flights (oldest first).
func (s *TravelService) Travel(user domain.UserID) ([]domain.CountryEntry, []domain.Flight, error) {
	entries, err := s.Entries(user)
	if err != nil {
		return nil, nil, err
	}
	flights, err := s.Flights(user)
	if err != nil {
		return nil, nil, err
	}
	return entries, flights, nil
}

// Country returns the full entry of a country; ErrNotFound
// if the user has none.
func (s *TravelService) Country(user domain.UserID, iso string) (domain.CountryEntry, error) {
	e, err := s.entries.Get(user, iso)
	if err != nil {
		return domain.CountryEntry{}, err
	}
	if e == nil {
		return domain.CountryEntry{}, fmt.Errorf("%w: country %s", ErrNotFound, iso)
	}
	return *e, nil
}

// PublicTravel returns profile and public data of a user, but only if their profile is
// public; otherwise ErrNotFound, so it does not reveal whether the user exists.
func (s *TravelService) PublicTravel(username string) (domain.Profile, []domain.CountryEntry, []domain.Flight, error) {
	notFound := fmt.Errorf("%w: user %s", ErrNotFound, username)
	u, err := s.users.GetByUsername(strings.ToLower(username))
	if err != nil {
		return domain.Profile{}, nil, nil, err
	}
	if u == nil {
		return domain.Profile{}, nil, nil, notFound
	}
	settings, err := s.users.Settings(u.ID)
	if err != nil {
		return domain.Profile{}, nil, nil, err
	}
	if settings.ProfileVisibility != domain.Public {
		return domain.Profile{}, nil, nil, notFound
	}
	entries, err := s.entries.ListPublic(u.ID)
	if err != nil {
		return domain.Profile{}, nil, nil, err
	}
	flights, err := s.flights.ListPublic(u.ID)
	if err != nil {
		return domain.Profile{}, nil, nil, err
	}
	return u.Profile, entries, flights, nil
}

func (s *TravelService) defaultVisibility(user domain.UserID) (domain.Visibility, error) {
	settings, err := s.users.Settings(user)
	return settings.DefaultVisibility, err
}

// SaveCountry replaces the entry of a country. Without visibility an existing entry keeps
// its own and a new one gets the default; status none removes the entry and returns nil.
func (s *TravelService) SaveCountry(user domain.UserID, iso string, status domain.CountryStatus, visibility *domain.Visibility, d domain.CountryDetails) (*domain.CountryEntry, error) {
	if err := checkCode("country", iso); err != nil {
		return nil, err
	}
	if d.Rating < 0 || d.Rating > 5 {
		return nil, fmt.Errorf("%w: rating must be between 0 and 5", ErrInvalid)
	}
	if visibility != nil {
		if err := checkVisibility(*visibility); err != nil {
			return nil, err
		}
	}

	switch status {
	case domain.StatusNone:
		return nil, s.entries.Delete(user, iso)
	case domain.StatusVisited, domain.StatusWishlist:
	default:
		return nil, fmt.Errorf("%w: unknown status %q", ErrInvalid, status)
	}

	e := domain.CountryEntry{
		ISO:    iso,
		Status: status,
		Details: domain.CountryDetails{
			Food:        strings.TrimSpace(d.Food),
			Sightseeing: strings.TrimSpace(d.Sightseeing),
			Notes:       strings.TrimSpace(d.Notes),
			Rating:      d.Rating,
			Tags:        normalizeTags(d.Tags),
		},
	}
	if visibility != nil {
		e.Visibility = *visibility
	} else {
		existing, err := s.entries.Get(user, iso)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			e.Visibility = existing.Visibility
		} else if e.Visibility, err = s.defaultVisibility(user); err != nil {
			return nil, err
		}
	}
	if err := s.entries.Upsert(user, e); err != nil {
		return nil, err
	}
	return &e, nil
}

// CreateFlight creates a flight. The countries of both airports (resolved by the client) then
// count as visited; the result lists the changed countries. Without visibility the default applies.
func (s *TravelService) CreateFlight(user domain.UserID, from, to string, countries []string, visibility *domain.Visibility) (domain.Flight, []string, error) {
	for _, iata := range []string{from, to} {
		if err := checkCode("airport", iata); err != nil {
			return domain.Flight{}, nil, err
		}
	}
	if len(countries) > 2 {
		return domain.Flight{}, nil, fmt.Errorf("%w: a flight touches at most two countries", ErrInvalid)
	}
	for _, iso := range countries {
		if err := checkCode("country", iso); err != nil {
			return domain.Flight{}, nil, err
		}
	}
	countries = slices.Compact(slices.Clone(countries))
	if visibility != nil {
		if err := checkVisibility(*visibility); err != nil {
			return domain.Flight{}, nil, err
		}
	}
	if from == to {
		return domain.Flight{}, nil, ErrSameAirport
	}
	exists, err := s.flights.RouteExists(user, from, to)
	if err != nil {
		return domain.Flight{}, nil, err
	}
	if exists {
		return domain.Flight{}, nil, fmt.Errorf("%w: flight %s → %s", ErrDuplicate, from, to)
	}

	def, err := s.defaultVisibility(user)
	if err != nil {
		return domain.Flight{}, nil, err
	}
	f := domain.Flight{ID: uuid.NewString(), From: from, To: to, Visibility: def, CreatedAt: s.now().UTC()}
	if visibility != nil {
		f.Visibility = *visibility
	}
	if err := s.flights.Create(user, f); err != nil {
		return domain.Flight{}, nil, err
	}

	updated := []string{}
	for _, iso := range countries {
		changed, err := s.markVisited(user, iso, def)
		if err != nil {
			return domain.Flight{}, nil, err
		}
		if changed {
			updated = append(updated, iso)
		}
	}
	return f, updated, nil
}

// markVisited marks a country as visited and keeps existing details
// and visibility; new entries get def.
func (s *TravelService) markVisited(user domain.UserID, iso string, def domain.Visibility) (bool, error) {
	e, err := s.entries.Get(user, iso)
	if err != nil {
		return false, err
	}
	if e == nil {
		e = &domain.CountryEntry{ISO: iso, Visibility: def, Details: domain.CountryDetails{Tags: []string{}}}
	} else if e.Status == domain.StatusVisited {
		return false, nil
	}
	e.Status = domain.StatusVisited
	return true, s.entries.Upsert(user, *e)
}

// UpdateFlight changes the given fields of a flight; empty values clear them.
func (s *TravelService) UpdateFlight(user domain.UserID, id string, u domain.FlightUpdate) error {
	if u.Empty() {
		return fmt.Errorf("%w: nothing to change", ErrInvalid)
	}
	if err := checkFlightUpdate(&u); err != nil {
		return err
	}
	found, err := s.flights.Update(user, id, u)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: flight %s", ErrNotFound, id)
	}
	return nil
}

// DeleteFlight removes a flight; visited countries stay visited.
func (s *TravelService) DeleteFlight(user domain.UserID, id string) error {
	found, err := s.flights.Delete(user, id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: flight %s", ErrNotFound, id)
	}
	return nil
}

func normalizeTags(tags []string) []string {
	out := []string{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}
