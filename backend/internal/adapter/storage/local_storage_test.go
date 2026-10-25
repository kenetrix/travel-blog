package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoragePerUser(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	a, err := s.Save("alice", "jpg", strings.NewReader("a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Save("bob", "jpg", strings.NewReader("b"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a, "alice/") || !strings.HasPrefix(b, "bob/") {
		t.Errorf("keys not per user: %s %s", a, b)
	}
	if _, err := os.Stat(filepath.Join(dir, a)); err != nil {
		t.Errorf("file not under base dir: %v", err)
	}

	f, err := s.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if string(got) != "a" {
		t.Errorf("content = %q", got)
	}

	if err := s.DeleteAll("alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(a); err == nil {
		t.Error("alice's file survived DeleteAll")
	}
	if _, err := s.Open(b); err != nil {
		t.Errorf("bob's file was removed: %v", err)
	}
}

func TestLocalStorageRejectsEscapes(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for _, key := range []string{"../x.jpg", "alice/../../x.jpg", "/etc/passwd", "alice/x.jpg"} {
		if _, err := s.Open(key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Open(%q) = %v", key, err)
		}
		if err := s.Delete(key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Delete(%q) = %v", key, err)
		}
	}
	if _, err := s.Save("../alice", "jpg", strings.NewReader("")); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("Save with bad user: %v", err)
	}
	if err := s.DeleteAll(".."); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("DeleteAll(..): %v", err)
	}
}
