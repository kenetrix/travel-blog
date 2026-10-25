package service_test

import (
	"errors"
	"strings"
	"testing"

	"travel-blog/backend/internal/application/service"
)

const issuer = "https://id.example.com"

func claims(sub, username, email string, verified bool) service.OIDCClaims {
	return service.OIDCClaims{Issuer: issuer, Subject: sub, PreferredUsername: username,
		Email: email, EmailVerified: verified, GivenName: "Ada", FamilyName: "Lovelace", Locale: "de_DE"}
}

func TestOIDCFirstLoginCreatesAccount(t *testing.T) {
	e := newEnv(t)
	id, token, err := e.accounts.LoginOIDC(claims("s1", "Ada.L", "Ada@Example.com", true))
	if err != nil || token == "" {
		t.Fatalf("login: %v", err)
	}
	u, _, err := e.accounts.Account(id)
	if err != nil {
		t.Fatal(err)
	}
	p := u.Profile
	if p.Username != "ada_l" || p.Email != "ada@example.com" || p.DisplayName != "Ada Lovelace" || p.Language != "de-DE" {
		t.Errorf("profile: %+v", p)
	}
	if u.MustChangePassword {
		t.Error("OIDC account must not be forced to change a password")
	}
	if user, ok, _ := e.accounts.Authenticate(token); !ok || user != id {
		t.Error("session does not authenticate")
	}
	// Returning user, even with changed claims
	id2, _, err := e.accounts.LoginOIDC(claims("s1", "other", "new@example.com", true))
	if err != nil || id2 != id {
		t.Errorf("second login: %v %v", id2, err)
	}
}

func TestOIDCAccountHasNoPasswordLogin(t *testing.T) {
	e := newEnv(t)
	if _, _, err := e.accounts.LoginOIDC(claims("s1", "ada", "", false)); err != nil {
		t.Fatal(err)
	}
	for _, pw := range []string{"", "correct horse battery"} {
		if _, _, err := e.accounts.Login("ada", pw); !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("password login %q: %v", pw, err)
		}
	}
}

func TestOIDCLinksVerifiedEmail(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.register(t, "alice")
	id, _, err := e.accounts.LoginOIDC(claims("s1", "whoever", "alice@example.com", true))
	if err != nil || id != alice {
		t.Fatalf("link: %v %v", id, err)
	}
	if id2, _, err := e.accounts.LoginOIDC(claims("s1", "x", "", false)); err != nil || id2 != alice {
		t.Errorf("relogin: %v %v", id2, err)
	}
}

func TestOIDCRejectsUnverifiedEmailOfExistingAccount(t *testing.T) {
	e := newEnv(t)
	e.register(t, "alice")
	_, _, err := e.accounts.LoginOIDC(claims("s1", "mallory", "alice@example.com", false))
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	if taken, _ := e.accounts.UsernameTaken("mallory"); taken {
		t.Error("account was created despite rejection")
	}
}

func TestOIDCUnverifiedEmailIsNotStored(t *testing.T) {
	e := newEnv(t)
	id, _, err := e.accounts.LoginOIDC(claims("s1", "ada", "ada@example.com", false))
	if err != nil {
		t.Fatal(err)
	}
	if u, _, _ := e.accounts.Account(id); u.Profile.Email != "" {
		t.Errorf("email %q stored", u.Profile.Email)
	}
}

func TestOIDCRegistrationClosed(t *testing.T) {
	e := newEnv(t)
	if _, err := e.db.Exec(`UPDATE app_settings SET value = 'false' WHERE key = 'registration_open'`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.accounts.LoginOIDC(claims("s1", "ada", "", false)); !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("want forbidden, got %v", err)
	}
	// A known identity still logs in
	e2 := newEnv(t)
	id, _, _ := e2.accounts.LoginOIDC(claims("s1", "ada", "", false))
	if _, err := e2.db.Exec(`UPDATE app_settings SET value = 'false' WHERE key = 'registration_open'`); err != nil {
		t.Fatal(err)
	}
	if id2, _, err := e2.accounts.LoginOIDC(claims("s1", "ada", "", false)); err != nil || id2 != id {
		t.Errorf("known identity: %v %v", id2, err)
	}
}

func TestOIDCUsernameCollisionAndFallbacks(t *testing.T) {
	e := newEnv(t)
	e.register(t, "ada")
	names := map[string]bool{}
	for i, c := range []service.OIDCClaims{
		claims("a", "ada", "", false),
		claims("b", "ada", "", false),
		claims("c", "", "grace.h@example.org", true),
		claims("d", "!!", "", false),
		claims("e", "admin", "", false), // reserved
	} {
		id, _, err := e.accounts.LoginOIDC(c)
		if err != nil {
			t.Fatalf("login %d: %v", i, err)
		}
		u, _, _ := e.accounts.Account(id)
		n := u.Profile.Username
		if names[n] || n == "ada" || n == "admin" {
			t.Errorf("login %d: username %q", i, n)
		}
		names[n] = true
	}
	if !names["ada_2"] || !names["ada_3"] || !names["grace_h"] {
		t.Errorf("usernames: %v", names)
	}
	for n := range names {
		if strings.ContainsAny(n, "-! ") {
			t.Errorf("invalid username %q", n)
		}
	}
}

func TestOIDCRejectsMissingSubject(t *testing.T) {
	e := newEnv(t)
	if _, _, err := e.accounts.LoginOIDC(service.OIDCClaims{Issuer: issuer}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("got %v", err)
	}
}

func TestOIDCAccountsAreListedForAdmins(t *testing.T) {
	e := newEnv(t)
	admin, _ := e.register(t, "boss")
	if _, err := e.db.Exec(`UPDATE users SET role = 'admin' WHERE id = ?`, admin); err != nil {
		t.Fatal(err)
	}
	e.accounts.LoginOIDC(claims("s1", "ada", "", false))
	users, err := e.accounts.Users(admin)
	if err != nil || len(users) != 3 { // incl. the test fixture "local"
		t.Fatalf("users: %d %v", len(users), err)
	}
}

func TestOIDCIdentityIsRemovedWithAccount(t *testing.T) {
	e := newEnv(t)
	id, _, err := e.accounts.LoginOIDC(claims("s1", "ada", "", false))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.accounts.DeleteAccount(id); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM oidc_identities`).Scan(&n); err != nil || n != 0 {
		t.Errorf("identities left: %d %v", n, err)
	}
}
