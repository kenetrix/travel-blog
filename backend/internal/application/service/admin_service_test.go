package service_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

// readInitialAdmin reads username and password from the credentials file.
func readInitialAdmin(t *testing.T, file string) (username, password string) {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "username: "); ok {
			username = v
		}
		if v, ok := strings.CutPrefix(line, "password: "); ok {
			password = v
		}
	}
	return username, password
}

func TestEnsureAdmin(t *testing.T) {
	e := newEnv(t)
	a := e.accounts
	file := filepath.Join(t.TempDir(), "initial-admin.txt")

	created, err := a.EnsureAdmin(file)
	if err != nil || !created {
		t.Fatalf("first start: %v %v", created, err)
	}
	username, password := readInitialAdmin(t, file)
	if username != "admin" || len(password) < 20 {
		t.Fatalf("credentials: %q %q", username, password)
	}
	if created, err := a.EnsureAdmin(file); err != nil || created {
		t.Errorf("second start: %v %v", created, err)
	}

	id, token, err := a.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}
	u, _, err := a.Account(id)
	if err != nil || u.Role != domain.RoleAdmin || !u.MustChangePassword || !a.MustChangePassword(id) {
		t.Fatalf("account: %+v %v", u, err)
	}

	if err := a.ChangePassword(id, token, password, "a brand new password"); err != nil {
		t.Fatal(err)
	}
	if a.MustChangePassword(id) {
		t.Error("still must change the password")
	}
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("credentials file stays: %v", err)
	}
}

func TestEnsureAdminKeepsExistingAccount(t *testing.T) {
	e := newEnv(t)
	// A regular account already uses the name "admin" (before it was reserved)
	if _, err := e.db.Exec(`INSERT INTO users (id, username, password_hash, created_at)
		VALUES ('legacy', 'admin', 'x', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "initial-admin.txt")

	if created, err := e.accounts.EnsureAdmin(file); err != nil || !created {
		t.Fatalf("ensure: %v %v", created, err)
	}
	username, _ := readInitialAdmin(t, file)
	if username == "admin" || !strings.HasPrefix(username, "admin_") {
		t.Errorf("username %q takes over the existing account", username)
	}
}

func TestAdminOperations(t *testing.T) {
	e := newEnv(t)
	a := e.accounts
	file := filepath.Join(t.TempDir(), "initial-admin.txt")
	if _, err := a.EnsureAdmin(file); err != nil {
		t.Fatal(err)
	}
	name, pass := readInitialAdmin(t, file)
	admin, _, err := a.Login(name, pass)
	if err != nil {
		t.Fatal(err)
	}
	alice, _ := e.register(t, "alice")

	// Only administrators
	if _, err := a.Users(alice); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("Users as user: %v", err)
	}
	if _, err := a.SetRole(alice, alice, domain.RoleAdmin); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("SetRole as user: %v", err)
	}
	if err := a.SetRegistrationOpen(alice, false); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("SetRegistrationOpen as user: %v", err)
	}

	users, err := a.Users(admin)
	if err != nil || len(users) != 3 { // incl. the test fixture "local"
		t.Fatalf("Users: %d %v", len(users), err)
	}

	// The last administrator stays one; a second one can step down
	if _, err := a.SetRole(admin, admin, domain.RoleUser); !errors.Is(err, service.ErrConflict) {
		t.Errorf("demote last admin: %v", err)
	}
	if _, err := a.SetRole(admin, alice, "owner"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("unknown role: %v", err)
	}
	if u, err := a.SetRole(admin, alice, domain.RoleAdmin); err != nil || u.Role != domain.RoleAdmin {
		t.Fatalf("promote: %+v %v", u, err)
	}
	if _, err := a.SetRole(alice, admin, domain.RoleUser); err != nil {
		t.Errorf("demote with two admins: %v", err)
	}
	if _, err := a.SetRole(admin, alice, domain.RoleUser); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("demoted admin acts as admin: %v", err)
	}
	if _, err := a.SetRole(alice, domain.UserID("nobody"), domain.RoleUser); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}

	// Password reset: new password, old sessions gone, change required
	bob, bobToken := e.register(t, "bob")
	if err := a.ResetPassword(alice, alice, "another new password"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("reset own password: %v", err)
	}
	if err := a.ResetPassword(alice, bob, "short"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("weak password: %v", err)
	}
	if err := a.ResetPassword(alice, bob, "temporary password"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := a.Authenticate(bobToken); ok {
		t.Error("session survives the reset")
	}
	if _, _, err := a.Login("bob", "correct horse battery"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("old password: %v", err)
	}
	if _, _, err := a.Login("bob", "temporary password"); err != nil || !a.MustChangePassword(bob) {
		t.Errorf("temporary password: %v", err)
	}

	// Registration switch
	if err := a.SetRegistrationOpen(alice, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Register(service.Registration{Username: "carol", Password: "correct horse battery"}); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("register while closed: %v", err)
	}
	if open, err := a.AdminSettings(alice); err != nil || open {
		t.Errorf("AdminSettings: %v %v", open, err)
	}
	if err := a.SetRegistrationOpen(alice, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Register(service.Registration{Username: "carol", Password: "correct horse battery"}); err != nil {
		t.Errorf("register while open: %v", err)
	}

	// Deleting
	if err := a.DeleteUser(alice, alice); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("delete self: %v", err)
	}
	if err := a.DeleteUser(alice, bob); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.Account(bob); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("deleted user: %v", err)
	}
}

func TestEnsureDevAdmin(t *testing.T) {
	e := newEnv(t)
	id, err := e.accounts.EnsureDevAdmin()
	if err != nil {
		t.Fatal(err)
	}
	if again, err := e.accounts.EnsureDevAdmin(); err != nil || again != id {
		t.Fatalf("second call: %v %v", again, err)
	}
	if _, _, err := e.accounts.Login("admin", ""); err == nil {
		t.Error("the dev admin can log in without a password")
	}
	if _, err := e.accounts.Users(id); err != nil {
		t.Errorf("dev admin is no administrator: %v", err)
	}
}
