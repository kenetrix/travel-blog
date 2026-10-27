package service_test

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func stop(name, country string) domain.Location {
	return domain.Location{Name: name, CountryID: country, Lat: 47.5, Lng: 16.4}
}

func stopNames(t domain.Trip) []string {
	names := make([]string, len(t.Stops))
	for i, s := range t.Stops {
		names[i] = s.Name
	}
	return names
}

func TestTripLifecycle(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.register(t, "alice")
	bob, _ := e.register(t, "bob")

	trip, err := e.trips.CreateTrip(alice, service.TripInput{
		Title:    "  Danube  ",
		StartsOn: "2024-05-01",
		EndsOn:   "2024-05-14",
		Stops:    []domain.Location{stop(" Vienna ", "AUT"), stop("Bratislava", "SVK"), stop("Budapest", "HUN"), stop("Vienna", "AUT")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if trip.Title != "Danube" || trip.Visibility != domain.Private || !slices.Equal(stopNames(trip), []string{"Vienna", "Bratislava", "Budapest", "Vienna"}) {
		t.Errorf("created trip = %+v", trip)
	}
	if got := trip.Countries(); !slices.Equal(got, []string{"AUT", "SVK", "HUN"}) {
		t.Errorf("countries = %v", got)
	}

	summaries, err := e.trips.Trips(alice)
	if err != nil || len(summaries) != 1 || summaries[0].StopCount != 4 || !slices.Equal(summaries[0].Countries, []string{"AUT", "SVK", "HUN"}) {
		t.Errorf("summaries = %+v, %v", summaries, err)
	}
	if others, _ := e.trips.Trips(bob); len(others) != 0 {
		t.Errorf("bob sees alice's trips: %+v", others)
	}

	// Rename Vienna, drop Bratislava, move Budapest first, add Graz; ids of kept stops stay
	vienna, budapest := trip.Stops[0], trip.Stops[2]
	vienna.Name = "Wien"
	public := domain.Public
	trip, err = e.trips.ReplaceTrip(alice, trip.ID, service.TripInput{
		Title:      "Danube",
		Visibility: &public,
		Stops:      []domain.Location{budapest, vienna, stop("Graz", "AUT")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(stopNames(trip), []string{"Budapest", "Wien", "Graz"}) || trip.Stops[0].ID != budapest.ID || trip.Stops[1].ID != vienna.ID {
		t.Errorf("replaced stops = %+v", trip.Stops)
	}
	for _, s := range trip.Stops {
		if s.Visibility != domain.Public {
			t.Errorf("stop %s has visibility %s, want the trip's", s.Name, s.Visibility)
		}
	}
	if trip.StartsOn != "" || trip.Visibility != domain.Public {
		t.Errorf("trip fields after replace = %+v", trip)
	}

	// Without visibility the trip keeps its own
	if trip, err = e.trips.ReplaceTrip(alice, trip.ID, service.TripInput{Title: "Danube"}); err != nil || trip.Visibility != domain.Public || len(trip.Stops) != 0 {
		t.Errorf("replace without visibility: %+v, %v", trip, err)
	}

	if _, err := e.trips.Trip(bob, trip.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob read alice's trip: %v", err)
	}
	if _, err := e.trips.ReplaceTrip(bob, trip.ID, service.TripInput{Title: "x"}); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob replaced alice's trip: %v", err)
	}
	if err := e.trips.DeleteTrip(bob, trip.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob deleted alice's trip: %v", err)
	}
	if err := e.trips.DeleteTrip(alice, trip.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.trips.Trip(alice, trip.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("trip left after delete: %v", err)
	}
}

func TestTripValidation(t *testing.T) {
	e := newEnv(t)
	valid := func() service.TripInput {
		return service.TripInput{Title: "Trip", Stops: []domain.Location{stop("Vienna", "AUT")}}
	}
	other := e.addStop(t, local, "Kyoto")
	unknown := domain.Visibility("friends")

	cases := map[string]func(*service.TripInput){
		"title_required":      func(in *service.TripInput) { in.Title = "   " },
		"text_too_long":       func(in *service.TripInput) { in.Title = strings.Repeat("a", 121) },
		"invalid_date":        func(in *service.TripInput) { in.StartsOn = "2024-02-30" },
		"invalid_date_range":  func(in *service.TripInput) { in.StartsOn, in.EndsOn = "2024-05-14", "2024-05-01" },
		"invalid_visibility":  func(in *service.TripInput) { in.Visibility = &unknown },
		"too_many_stops":      func(in *service.TripInput) { in.Stops = slices.Repeat(in.Stops, service.MaxTripStops+1) },
		"name_required":       func(in *service.TripInput) { in.Stops[0].Name = "" },
		"invalid_code":        func(in *service.TripInput) { in.Stops[0].CountryID = "at" },
		"invalid_coordinates": func(in *service.TripInput) { in.Stops[0].Lat = 91 },
		"unknown_stop":        func(in *service.TripInput) { in.Stops[0].ID = other.ID },
	}
	for code, change := range cases {
		in := valid()
		change(&in)
		_, err := e.trips.CreateTrip(local, in)
		var se *service.Error
		if !errors.As(err, &se) || !errors.Is(err, service.ErrInvalid) || se.Code != code {
			t.Errorf("%s: got %v", code, err)
		}
	}

	in := valid()
	in.Stops[0].Lng = math.NaN()
	if _, err := e.trips.CreateTrip(local, in); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("NaN longitude accepted: %v", err)
	}

	// A stop id used twice in one trip is unknown the second time
	trip, err := e.trips.CreateTrip(local, valid())
	if err != nil {
		t.Fatal(err)
	}
	twice := service.TripInput{Title: "Trip", Stops: []domain.Location{trip.Stops[0], trip.Stops[0]}}
	if _, err := e.trips.ReplaceTrip(local, trip.ID, twice); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("duplicate stop id accepted: %v", err)
	}
	if got, _ := e.trips.Trip(local, trip.ID); len(got.Stops) != 1 {
		t.Errorf("failed replace changed the trip: %+v", got.Stops)
	}

	if _, err := e.trips.ReplaceTrip(local, "missing", valid()); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("replace of a missing trip: %v", err)
	}
}

func TestRemovedStopsLoseTheirPhotos(t *testing.T) {
	e := newEnv(t)
	trip, err := e.trips.CreateTrip(local, service.TripInput{
		Title: "Japan",
		Stops: []domain.Location{stop("Tokyo", "JPN"), stop("Kyoto", "JPN")},
	})
	if err != nil {
		t.Fatal(err)
	}
	tokyo, kyoto := trip.Stops[0], trip.Stops[1]
	keep, err := e.photos.Upload(local, tokyo.ID, "image/jpeg", strings.NewReader("a"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	drop, err := e.photos.Upload(local, kyoto.ID, "image/jpeg", strings.NewReader("b"), "", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.trips.ReplaceTrip(local, trip.ID, service.TripInput{Title: "Japan", Stops: []domain.Location{tokyo}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.photoDir, drop.StorageKey)); !os.IsNotExist(err) {
		t.Errorf("photo file of the removed stop left: %v", err)
	}
	if _, _, err := e.photos.Open(local, drop.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("photo record of the removed stop left: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.photoDir, keep.StorageKey)); err != nil {
		t.Errorf("photo file of the kept stop: %v", err)
	}
}
