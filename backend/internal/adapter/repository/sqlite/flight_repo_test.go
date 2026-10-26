package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/domain"
	"travel-blog/backend/migrations/data"
)

func TestFlightDetails(t *testing.T) {
	db, err := sqlite.NewDB(filepath.Join(t.TempDir(), "test.db"), data.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO users (id, username, password_hash, created_at)
		VALUES ('u1', 'alice', 'x', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.NewFlightRepo(db)
	user := domain.UserID("u1")

	only := func() domain.Flight {
		t.Helper()
		flights, err := repo.List(user)
		if err != nil || len(flights) != 1 {
			t.Fatalf("list: %v %v", flights, err)
		}
		return flights[0]
	}
	ptr := func(s string) *string { return &s }

	created := domain.Flight{ID: "f1", From: "BER", To: "HND", Visibility: domain.Private, CreatedAt: time.Now()}
	if err := repo.Create(user, created); err != nil {
		t.Fatal(err)
	}
	if f := only(); f.Number != "" || f.Date != "" || f.DurationMinutes != 0 || f.Aircraft != "" {
		t.Errorf("new flight has details: %+v", f)
	}

	duration := 710
	found, err := repo.Update(user, "f1", domain.FlightUpdate{
		Number: ptr("NH204"), Date: ptr("2026-03-14"), DurationMinutes: &duration, Aircraft: ptr("Boeing 787-9"),
	})
	if err != nil || !found {
		t.Fatalf("update: %v %v", found, err)
	}
	if f := only(); f.Number != "NH204" || f.Date != "2026-03-14" || f.DurationMinutes != 710 || f.Aircraft != "Boeing 787-9" || f.Visibility != domain.Private {
		t.Errorf("after update: %+v", f)
	}

	// Only the given fields change; empty values clear
	public := domain.Public
	zero := 0
	if _, err := repo.Update(user, "f1", domain.FlightUpdate{Visibility: &public, Date: ptr(""), DurationMinutes: &zero}); err != nil {
		t.Fatal(err)
	}
	if f := only(); f.Number != "NH204" || f.Date != "" || f.DurationMinutes != 0 || f.Aircraft != "Boeing 787-9" || f.Visibility != domain.Public {
		t.Errorf("after partial update: %+v", f)
	}

	if found, err := repo.Update(user, "missing", domain.FlightUpdate{Number: ptr("LH400")}); err != nil || found {
		t.Errorf("update of a missing flight: %v %v", found, err)
	}
	if found, err := repo.Update("someone-else", "f1", domain.FlightUpdate{Number: ptr("LH400")}); err != nil || found {
		t.Errorf("update of another user's flight: %v %v", found, err)
	}
}
