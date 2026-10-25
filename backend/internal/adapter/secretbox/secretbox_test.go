package secretbox_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"travel-blog/backend/internal/adapter/secretbox"
)

func TestRoundTripAndKeyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	box := secretbox.New(path)
	if _, err := os.Stat(path); err == nil {
		t.Fatal("key created before first use")
	}
	sealed, err := box.Encrypt("s3cret")
	if err != nil || strings.Contains(sealed, "s3cret") {
		t.Fatalf("encrypt: %q %v", sealed, err)
	}
	other, _ := box.Encrypt("s3cret")
	if other == sealed {
		t.Error("nonce reused")
	}
	// A fresh box on the same key file reads it back
	plain, err := secretbox.New(path).Decrypt(sealed)
	if err != nil || plain != "s3cret" {
		t.Errorf("decrypt: %q %v", plain, err)
	}
}

func TestDecryptFailures(t *testing.T) {
	dir := t.TempDir()
	sealed, err := secretbox.New(filepath.Join(dir, "a.key")).Encrypt("x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := secretbox.New(filepath.Join(dir, "missing.key")).Decrypt(sealed); err == nil {
		t.Error("decrypt without key file succeeded")
	}
	if _, err := secretbox.New(filepath.Join(dir, "b.key")).Encrypt("y"); err != nil {
		t.Fatal(err)
	}
	if _, err := secretbox.New(filepath.Join(dir, "b.key")).Decrypt(sealed); err == nil {
		t.Error("decrypt with wrong key succeeded")
	}
	if _, err := secretbox.New(filepath.Join(dir, "a.key")).Decrypt("not base64!"); err == nil {
		t.Error("malformed input accepted")
	}
}
