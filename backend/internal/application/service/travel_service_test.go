package service_test

import (
	"errors"
	"slices"
	"testing"

	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func vis(v domain.Visibility) *domain.Visibility { return &v }

func TestSaveCountry(t *testing.T) {
	s := newEnv(t).travel

	e, err := s.SaveCountry(local, "JPN", domain.StatusWishlist, nil, domain.CountryDetails{
		Notes: "  sushi  ",
		Tags:  []string{" City Trip", "city trip", "", "Food"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Details.Notes != "sushi" || !slices.Equal(e.Details.Tags, []string{"city trip", "food"}) {
		t.Errorf("not normalized: %+v", e.Details)
	}

	if _, err := s.SaveCountry(local, "deu", domain.StatusVisited, nil, domain.CountryDetails{}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("malformed country: %v", err)
	}
	if _, err := s.SaveCountry(local, "JPN", domain.StatusVisited, nil, domain.CountryDetails{Rating: 6}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("rating 6: %v", err)
	}
	if _, err := s.SaveCountry(local, "JPN", domain.StatusVisited, vis("friends"), domain.CountryDetails{}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("unknown visibility: %v", err)
	}

	if e, err := s.SaveCountry(local, "JPN", domain.StatusNone, nil, domain.CountryDetails{}); err != nil || e != nil {
		t.Fatalf("remove: %v, %v", e, err)
	}
	entries, _, _ := s.Travel(local)
	if len(entries) != 0 {
		t.Errorf("entry not removed: %+v", entries)
	}
}

func TestVisibilityDefaults(t *testing.T) {
	e := newEnv(t)
	s := e.travel

	// New users default to private
	jpn, _ := s.SaveCountry(local, "JPN", domain.StatusVisited, nil, domain.CountryDetails{})
	if jpn.Visibility != domain.Private {
		t.Errorf("default = %s", jpn.Visibility)
	}
	// Explicit visibility wins and sticks when later saves omit it
	s.SaveCountry(local, "JPN", domain.StatusVisited, vis(domain.Public), domain.CountryDetails{})
	jpn, _ = s.SaveCountry(local, "JPN", domain.StatusVisited, nil, domain.CountryDetails{Notes: "x"})
	if jpn.Visibility != domain.Public {
		t.Errorf("existing visibility lost: %s", jpn.Visibility)
	}

	settings := domain.DefaultSettings()
	settings.DefaultVisibility = domain.Public
	if err := e.accounts.SaveSettings(local, settings); err != nil {
		t.Fatal(err)
	}
	f, _, err := s.CreateFlight(local, "BER", "CDG", []string{"DEU", "FRA"}, nil)
	if err != nil || f.Visibility != domain.Public {
		t.Fatalf("flight: %+v %v", f, err)
	}
	entries, _, _ := s.Travel(local)
	for _, en := range entries {
		if en.Visibility != domain.Public {
			t.Errorf("%s promoted by flight is %s", en.ISO, en.Visibility)
		}
	}

	if err := s.SetFlightVisibility(local, f.ID, domain.Private); err != nil {
		t.Fatal(err)
	}
	if _, flights, _ := s.Travel(local); flights[0].Visibility != domain.Private {
		t.Error("SetFlightVisibility had no effect")
	}
	if err := s.SetFlightVisibility(local, "00000000-0000-0000-0000-000000000000", domain.Public); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown flight: %v", err)
	}
}

func TestPublicTravel(t *testing.T) {
	e := newEnv(t)
	s := e.travel
	alice, _ := e.register(t, "alice")

	s.SaveCountry(alice, "JPN", domain.StatusVisited, vis(domain.Public), domain.CountryDetails{})
	s.SaveCountry(alice, "FRA", domain.StatusVisited, vis(domain.Private), domain.CountryDetails{})
	s.CreateFlight(alice, "BER", "HND", nil, vis(domain.Public))
	s.CreateFlight(alice, "BER", "CDG", nil, vis(domain.Private))

	if _, _, _, err := s.PublicTravel("alice"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("private profile must look missing: %v", err)
	}

	settings := domain.DefaultSettings()
	settings.ProfileVisibility = domain.Public
	e.accounts.SaveSettings(alice, settings)

	p, entries, flights, err := s.PublicTravel("Alice")
	if err != nil {
		t.Fatal(err)
	}
	if p.Username != "alice" || len(entries) != 1 || entries[0].ISO != "JPN" || len(flights) != 1 || flights[0].To != "HND" {
		t.Errorf("public view: %+v %+v %+v", p, entries, flights)
	}
	if _, _, _, err := s.PublicTravel("nobody"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}
}

func TestCreateFlight(t *testing.T) {
	s := newEnv(t).travel
	if _, err := s.SaveCountry(local, "JPN", domain.StatusWishlist, nil, domain.CountryDetails{Notes: "keep"}); err != nil {
		t.Fatal(err)
	}

	f, updated, err := s.CreateFlight(local, "BER", "HND", []string{"DEU", "JPN"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(updated, []string{"DEU", "JPN"}) {
		t.Errorf("countriesUpdated = %v", updated)
	}
	entries, flights, _ := s.Travel(local)
	if len(flights) != 1 || flights[0].ID != f.ID {
		t.Errorf("flights = %+v", flights)
	}
	for _, e := range entries {
		if e.Status != domain.StatusVisited {
			t.Errorf("%s not visited", e.ISO)
		}
		if e.ISO == "JPN" && e.Details.Notes != "keep" {
			t.Error("wishlist details lost on promotion")
		}
	}

	// Countries already visited are not reported again
	if _, updated, err := s.CreateFlight(local, "HND", "BER", []string{"JPN", "DEU"}, nil); err != nil || len(updated) != 0 {
		t.Errorf("return flight: %v, %v", updated, err)
	}

	cases := map[string]struct {
		from, to string
		want     error
	}{
		"duplicate":    {"BER", "HND", service.ErrDuplicate},
		"same airport": {"BER", "BER", service.ErrSameAirport},
		"malformed":    {"BER", "???", service.ErrInvalid},
	}
	for name, c := range cases {
		if _, _, err := s.CreateFlight(local, c.from, c.to, nil, nil); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}

	if err := s.DeleteFlight(local, f.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteFlight(local, f.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if entries, _, _ := s.Travel(local); len(entries) != 2 {
		t.Errorf("countries must stay visited, got %+v", entries)
	}
}

func TestUsersAreIsolated(t *testing.T) {
	e := newEnv(t)
	s := e.travel
	alice, _ := e.register(t, "alice")

	if _, err := s.SaveCountry(local, "JPN", domain.StatusWishlist, nil, domain.CountryDetails{}); err != nil {
		t.Fatal(err)
	}
	f, _, err := s.CreateFlight(local, "BER", "HND", []string{"DEU", "JPN"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	entries, flights, err := s.Travel(alice)
	if err != nil || len(entries) != 0 || len(flights) != 0 {
		t.Fatalf("alice sees local data: %v %v %v", entries, flights, err)
	}
	// The same route is no duplicate for another user
	if _, updated, err := s.CreateFlight(alice, "BER", "HND", []string{"DEU", "JPN"}, nil); err != nil || len(updated) != 2 {
		t.Errorf("alice's flight: %v, %v", updated, err)
	}
	if err := s.DeleteFlight(alice, f.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("alice deleted local's flight: %v", err)
	}
	if err := s.SetFlightVisibility(alice, f.ID, domain.Public); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("alice changed local's flight: %v", err)
	}

	// Deleting an account removes all of its data
	if err := e.accounts.DeleteAccount(alice); err != nil {
		t.Fatal(err)
	}
	entries, flights, _ = s.Travel(alice)
	if len(entries) != 0 || len(flights) != 0 {
		t.Errorf("data left after deleting the user: %v %v", entries, flights)
	}
	if entries, flights, _ := s.Travel(local); len(entries) != 2 || len(flights) != 1 {
		t.Errorf("local data changed: %v %v", entries, flights)
	}
}
