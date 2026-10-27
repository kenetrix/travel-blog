package service_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func TestPhotosBelongToTheirUser(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.register(t, "alice")
	bob, _ := e.register(t, "bob")

	loc := e.addStop(t, alice, "Kyoto")
	if loc.Visibility != domain.Private {
		t.Errorf("location visibility = %s", loc.Visibility)
	}

	if _, err := e.photos.Upload(bob, loc.ID, "image/jpeg", strings.NewReader("x"), "", ""); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob uploaded to alice's location: %v", err)
	}
	if _, err := e.photos.Upload(alice, loc.ID, "text/html", strings.NewReader("x"), "", ""); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("html accepted: %v", err)
	}

	p, err := e.photos.Upload(alice, loc.ID, "image/jpeg", strings.NewReader("temple"), " Kinkaku-ji ", "2025-04-01")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.StorageKey, string(alice)+"/") || !strings.HasSuffix(p.StorageKey, ".jpg") || p.Caption != "Kinkaku-ji" {
		t.Errorf("photo = %+v", p)
	}

	_, f, err := e.photos.Open(alice, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(f)
	f.Close()
	if string(content) != "temple" {
		t.Errorf("content = %q", content)
	}

	if _, _, err := e.photos.Open(bob, p.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob opened alice's photo: %v", err)
	}
	if err := e.photos.Delete(bob, p.ID); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("bob deleted alice's photo: %v", err)
	}

	if err := e.photos.Delete(alice, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.photoDir, p.StorageKey)); !os.IsNotExist(err) {
		t.Errorf("file left after delete: %v", err)
	}
}

func TestPhotoFilesFollowTheirRecords(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.register(t, "alice")
	loc := e.addStop(t, alice, "Kyoto")
	p, err := e.photos.Upload(alice, loc.ID, "image/png", strings.NewReader("a"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(e.photoDir, p.StorageKey)

	// Deleting the trip deletes the photo files of its stops
	if err := e.trips.DeleteTrip(alice, loc.TripID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("file left after deleting the trip: %v", err)
	}

	// Deleting the account deletes all of its files
	loc = e.addStop(t, alice, "Nara")
	if _, err := e.photos.Upload(alice, loc.ID, "image/png", strings.NewReader("b"), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := e.accounts.DeleteAccount(alice); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.photoDir, string(alice))); !os.IsNotExist(err) {
		t.Errorf("user folder left after account deletion: %v", err)
	}
}

func TestPhotoSizeLimit(t *testing.T) {
	e := newEnv(t)
	loc := e.addStop(t, local, "Kyoto")
	big := io.LimitReader(zeros{}, service.MaxPhotoBytes+1)
	if _, err := e.photos.Upload(local, loc.ID, "image/jpeg", big, "", ""); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("oversized photo: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(e.photoDir, string(local))); len(entries) != 0 {
		t.Errorf("oversized file kept: %v", entries)
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
