package sqlite_test

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/domain"
	"travel-blog/backend/migrations/data"
)

func TestTripStops(t *testing.T) {
	db, err := sqlite.NewDB(filepath.Join(t.TempDir(), "test.db"), data.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo := sqlite.NewTripRepo(db)
	user := domain.UserID("u1")

	stop := func(id, name, country string) domain.Location {
		return domain.Location{ID: id, Name: name, CountryID: country, Lat: 48.2, Lng: 16.4}
	}
	ids := func(trip *domain.Trip) []string {
		out := []string{}
		for i, s := range trip.Stops {
			if s.Position != i {
				t.Errorf("stop %s has position %d at index %d", s.ID, s.Position, i)
			}
			out = append(out, s.ID)
		}
		return out
	}
	get := func(id string) *domain.Trip {
		t.Helper()
		trip, err := repo.Get(user, id)
		if err != nil || trip == nil {
			t.Fatalf("get: %v %v", trip, err)
		}
		return trip
	}

	now := time.Now()
	trip := domain.Trip{
		ID: "t1", Title: "Danube", StartsOn: "2024-05-01", Visibility: domain.Private, CreatedAt: now, UpdatedAt: now,
		Stops: []domain.Location{stop("a", "Vienna", "AUT"), stop("b", "Bratislava", "SVK"), stop("c", "Budapest", "HUN")},
	}
	if err := repo.Create(user, trip); err != nil {
		t.Fatal(err)
	}
	if got := get("t1"); !slices.Equal(ids(got), []string{"a", "b", "c"}) || got.StartsOn != "2024-05-01" || got.EndsOn != "" {
		t.Errorf("created = %+v", got)
	}

	// Reverse the order: positions are swapped without hitting the unique constraint
	trip.Stops = []domain.Location{trip.Stops[2], trip.Stops[1], trip.Stops[0]}
	if found, _, err := repo.Replace(user, trip); err != nil || !found {
		t.Fatalf("reorder: %v %v", found, err)
	}
	if got := ids(get("t1")); !slices.Equal(got, []string{"c", "b", "a"}) {
		t.Errorf("after reorder = %v", got)
	}

	// Remove b with its photo, add d
	if _, err := db.Exec(`INSERT INTO photos (id, location_id, storage_key, content_type, created_at)
		VALUES ('p1', 'b', 'u1/p1.jpg', 'image/jpeg', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	trip.Stops = []domain.Location{trip.Stops[0], stop("d", "Graz", "AUT"), trip.Stops[2]}
	found, removed, err := repo.Replace(user, trip)
	if err != nil || !found || !slices.Equal(removed, []string{"u1/p1.jpg"}) {
		t.Fatalf("remove: %v %v %v", found, removed, err)
	}
	if got := ids(get("t1")); !slices.Equal(got, []string{"c", "d", "a"}) {
		t.Errorf("after remove = %v", got)
	}
	var photos int
	db.QueryRow(`SELECT COUNT(*) FROM photos`).Scan(&photos)
	if photos != 0 {
		t.Errorf("photo of the removed stop left: %d", photos)
	}

	summaries, err := repo.List(user)
	if err != nil || len(summaries) != 1 || summaries[0].StopCount != 3 || !slices.Equal(summaries[0].Countries, []string{"HUN", "AUT"}) {
		t.Errorf("summaries = %+v, %v", summaries, err)
	}

	// A failing stop rolls back the whole replacement
	other := domain.Trip{ID: "t2", Title: "Other", Visibility: domain.Private, CreatedAt: now, UpdatedAt: now,
		Stops: []domain.Location{stop("x", "Rome", "ITA")}}
	if err := repo.Create(user, other); err != nil {
		t.Fatal(err)
	}
	broken := trip
	broken.Title = "Changed"
	broken.Stops = []domain.Location{trip.Stops[0], stop("x", "Clash", "ITA")}
	if _, _, err := repo.Replace(user, broken); err == nil {
		t.Fatal("replace with a clashing stop id succeeded")
	}
	if got := get("t1"); got.Title != "Danube" || !slices.Equal(ids(got), []string{"c", "d", "a"}) {
		t.Errorf("after failed replace = %+v", got)
	}

	// Another user sees and changes nothing
	if got, err := repo.Get("u2", "t1"); err != nil || got != nil {
		t.Errorf("bob got alice's trip: %v %v", got, err)
	}
	if list, _ := repo.List("u2"); len(list) != 0 {
		t.Errorf("bob lists %v", list)
	}
	if found, _, err := repo.Replace("u2", trip); err != nil || found {
		t.Errorf("bob replaced alice's trip: %v %v", found, err)
	}
	if found, _, err := repo.Delete("u2", "t1"); err != nil || found {
		t.Errorf("bob deleted alice's trip: %v %v", found, err)
	}

	if found, _, err := repo.Delete(user, "t1"); err != nil || !found {
		t.Fatalf("delete: %v %v", found, err)
	}
	var stops int
	db.QueryRow(`SELECT COUNT(*) FROM locations WHERE trip_id = 't1'`).Scan(&stops)
	if stops != 0 {
		t.Errorf("stops left after deleting the trip: %d", stops)
	}
}
