package service

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

const (
	MaxTripStops = 200
	maxTripTitle = 120
	maxStopName  = 120
	maxTripNotes = 4000
	maxStopNotes = 4000
	maxStopLinks = 10
	maxLinkURL   = 500
	maxLinkLabel = 80
)

// TripService manages the trips of users with their stops. Every method only works on
// the data of the given user.
type TripService struct {
	trips port.TripRepository
	files port.FileStorage
	users port.UserRepository
	now   func() time.Time
}

func NewTripService(trips port.TripRepository, files port.FileStorage, users port.UserRepository) *TripService {
	return &TripService{trips: trips, files: files, users: users, now: time.Now}
}

// TripInput is a whole trip as the editor sends it. A stop with an ID updates that stop
// of the trip, one without is created; the order of Stops is the visiting order.
type TripInput struct {
	Title      string
	StartsOn   string
	EndsOn     string
	Notes      string
	Visibility *domain.Visibility
	Stops      []domain.Location
}

func (s *TripService) Trips(user domain.UserID) ([]domain.TripSummary, error) {
	return s.trips.List(user)
}

func (s *TripService) Trip(user domain.UserID, id string) (domain.Trip, error) {
	t, err := s.trips.Get(user, id)
	if err != nil {
		return domain.Trip{}, err
	}
	if t == nil {
		return domain.Trip{}, tripNotFound(id)
	}
	return *t, nil
}

// CreateTrip stores a new trip; without visibility the user's default applies.
func (s *TripService) CreateTrip(user domain.UserID, in TripInput) (domain.Trip, error) {
	t, err := checkTrip(in, nil)
	if err != nil {
		return domain.Trip{}, err
	}
	if in.Visibility != nil {
		t.Visibility = *in.Visibility
	} else {
		settings, err := s.users.Settings(user)
		if err != nil {
			return domain.Trip{}, err
		}
		t.Visibility = settings.DefaultVisibility
	}
	t.ID = uuid.NewString()
	t.CreatedAt = s.now().UTC()
	t.UpdatedAt = t.CreatedAt
	if err := s.trips.Create(user, t); err != nil {
		return domain.Trip{}, err
	}
	return s.Trip(user, t.ID)
}

// ReplaceTrip overwrites a trip with its stops; stops missing from in are deleted with
// their photos. Without visibility the trip keeps its own.
func (s *TripService) ReplaceTrip(user domain.UserID, id string, in TripInput) (domain.Trip, error) {
	existing, err := s.trips.Get(user, id)
	if err != nil {
		return domain.Trip{}, err
	}
	if existing == nil {
		return domain.Trip{}, tripNotFound(id)
	}
	t, err := checkTrip(in, existing.Stops)
	if err != nil {
		return domain.Trip{}, err
	}
	t.ID = id
	t.Visibility = existing.Visibility
	if in.Visibility != nil {
		t.Visibility = *in.Visibility
	}
	t.UpdatedAt = s.now().UTC()
	found, removed, err := s.trips.Replace(user, t)
	if err != nil {
		return domain.Trip{}, err
	}
	if !found {
		return domain.Trip{}, tripNotFound(id)
	}
	s.removeFiles(removed)
	return s.Trip(user, id)
}

// DeleteTrip removes a trip with its stops and their photos, records and files.
func (s *TripService) DeleteTrip(user domain.UserID, id string) error {
	found, removed, err := s.trips.Delete(user, id)
	if err != nil {
		return err
	}
	if !found {
		return tripNotFound(id)
	}
	s.removeFiles(removed)
	return nil
}

// removeFiles runs after the records are gone; a failure only leaves an orphaned file.
func (s *TripService) removeFiles(keys []string) {
	for _, key := range keys {
		if err := s.files.Delete(key); err != nil {
			log.Printf("delete photo file %s: %v", key, err)
		}
	}
}

func tripNotFound(id string) error {
	return newError(ErrNotFound, "trip_not_found", fmt.Sprintf("trip %s", id))
}

// checkTrip validates and normalises in. Stop IDs must belong to existing (the stored
// stops of the trip); stops without one get a new ID.
func checkTrip(in TripInput, existing []domain.Location) (domain.Trip, error) {
	title, err := trimText("title", in.Title, maxTripTitle)
	if err != nil {
		return domain.Trip{}, err
	}
	if title == "" {
		return domain.Trip{}, newError(ErrInvalid, "title_required", "title is required")
	}
	notes, err := trimText("notes", in.Notes, maxTripNotes)
	if err != nil {
		return domain.Trip{}, err
	}
	for _, d := range []string{in.StartsOn, in.EndsOn} {
		if err := checkDate(d); err != nil {
			return domain.Trip{}, err
		}
	}
	// Dates are YYYY-MM-DD, so they compare as strings
	if in.StartsOn != "" && in.EndsOn != "" && in.EndsOn < in.StartsOn {
		return domain.Trip{}, newError(ErrInvalid, "invalid_date_range", "the trip ends before it starts")
	}
	if in.Visibility != nil {
		if err := checkVisibility(*in.Visibility); err != nil {
			return domain.Trip{}, err
		}
	}
	if len(in.Stops) > MaxTripStops {
		return domain.Trip{}, newError(ErrInvalid, "too_many_stops", fmt.Sprintf("a trip has at most %d stops", MaxTripStops), "max", strconv.Itoa(MaxTripStops))
	}

	known := map[string]bool{}
	for _, l := range existing {
		known[l.ID] = true
	}
	stops := make([]domain.Location, len(in.Stops))
	for i, l := range in.Stops {
		if l.ID != "" {
			if !known[l.ID] {
				return domain.Trip{}, newError(ErrInvalid, "unknown_stop", fmt.Sprintf("stop %s is not part of this trip", l.ID), "value", l.ID)
			}
			// A second use of the same ID is unknown too
			delete(known, l.ID)
		} else {
			l.ID = uuid.NewString()
		}
		if l.Name, err = trimText("name", l.Name, maxStopName); err != nil {
			return domain.Trip{}, err
		}
		if l.Name == "" {
			return domain.Trip{}, newError(ErrInvalid, "name_required", "every stop needs a name")
		}
		if err := checkCode("country", l.CountryID); err != nil {
			return domain.Trip{}, err
		}
		if math.IsNaN(l.Lat) || math.IsNaN(l.Lng) || l.Lat < -90 || l.Lat > 90 || l.Lng < -180 || l.Lng > 180 {
			return domain.Trip{}, newError(ErrInvalid, "invalid_coordinates", "latitude must be -90 to 90 and longitude -180 to 180")
		}
		if err := checkDate(l.VisitedAt); err != nil {
			return domain.Trip{}, err
		}
		if l.Notes, err = trimText("notes", l.Notes, maxStopNotes); err != nil {
			return domain.Trip{}, err
		}
		if l.Rating < 0 || l.Rating > 5 {
			return domain.Trip{}, newError(ErrInvalid, "invalid_rating", "rating must be between 0 and 5")
		}
		if l.Category != "" && !slices.Contains(domain.StopCategories, l.Category) {
			return domain.Trip{}, newError(ErrInvalid, "invalid_category", "unknown stop category", "value", l.Category)
		}
		if l.Links, err = checkLinks(l.Links); err != nil {
			return domain.Trip{}, err
		}
		l.TripID, l.Position = "", i
		stops[i] = l
	}
	return domain.Trip{
		Title:    title,
		StartsOn: in.StartsOn,
		EndsOn:   in.EndsOn,
		Notes:    notes,
		Stops:    stops,
	}, nil
}

// checkLinks keeps only http(s) links, so a stop never carries a javascript: URL.
func checkLinks(in []domain.StopLink) ([]domain.StopLink, error) {
	if len(in) > maxStopLinks {
		return nil, newError(ErrInvalid, "too_many_links", fmt.Sprintf("a stop has at most %d links", maxStopLinks), "max", strconv.Itoa(maxStopLinks))
	}
	out := make([]domain.StopLink, 0, len(in))
	for _, l := range in {
		u, err := url.Parse(strings.TrimSpace(l.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(l.URL) > maxLinkURL {
			return nil, newError(ErrInvalid, "invalid_link", "a link must be a valid http or https URL", "value", l.URL)
		}
		label, err := trimText("label", l.Label, maxLinkLabel)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.StopLink{URL: u.String(), Label: label})
	}
	return out, nil
}
