package service_test

import (
	"errors"
	"testing"

	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

func ptr(s string) *string { return &s }

func TestRegisterAndLogin(t *testing.T) {
	e := newEnv(t)
	a := e.accounts

	u, token, err := a.Register(service.Registration{
		Email:     " Alice@Example.com ",
		Username:  "Alice_1",
		Password:  "correct horse battery",
		FirstName: "Alice",
		LastName:  "Liddell",
		Language:  "de",
	})
	if err != nil {
		t.Fatal(err)
	}
	p := u.Profile
	if p.Email != "alice@example.com" || p.Username != "alice_1" || p.DisplayName != "Alice Liddell" || p.Language != "de" || p.Timezone != "UTC" {
		t.Errorf("profile = %+v", p)
	}
	if id, ok, err := a.Authenticate(token); err != nil || !ok || id != u.ID {
		t.Errorf("register session: %v %v %v", id, ok, err)
	}
	if _, settings, _ := a.Account(u.ID); settings != domain.DefaultSettings() {
		t.Errorf("settings = %+v", settings)
	}

	invalid := []service.Registration{
		{Email: "no-at", Username: "bob", Password: "correct horse battery"},
		{Email: "bob@example.com", Username: "b", Password: "correct horse battery"},
		{Email: "bob@example.com", Username: "bob", Password: "short"},
		{Email: "bob@example.com", Username: "bob", Password: "correct horse battery", Language: "Deutsch"},
	}
	for _, r := range invalid {
		if _, _, err := a.Register(r); !errors.Is(err, service.ErrInvalid) {
			t.Errorf("Register(%+v) = %v", r, err)
		}
	}
	taken := []service.Registration{
		{Email: "ALICE@example.com", Username: "bob", Password: "correct horse battery"},
		{Email: "bob@example.com", Username: "alice_1", Password: "correct horse battery"},
		{Email: "bob@example.com", Username: "local", Password: "correct horse battery"},
	}
	for _, r := range taken {
		if _, _, err := a.Register(r); !errors.Is(err, service.ErrConflict) {
			t.Errorf("Register(%+v) = %v", r, err)
		}
	}

	if _, _, err := a.Login("alice_1", "wrong password!"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("wrong password: %v", err)
	}
	if _, _, err := a.Login("nobody", "correct horse battery"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("unknown username: %v", err)
	}
	id, token2, err := a.Login(" ALICE_1 ", "correct horse battery")
	if err != nil || id != u.ID || token2 == token {
		t.Fatalf("login: %v %v", id, err)
	}

	if err := a.Logout(token2); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := a.Authenticate(token2); ok {
		t.Error("session valid after logout")
	}
	if _, ok, _ := a.Authenticate("forged"); ok {
		t.Error("forged token accepted")
	}
}

func TestExpiredSession(t *testing.T) {
	e := newEnv(t)
	_, token := e.register(t, "alice")
	// Fast-forward past the session lifetime
	if _, err := e.db.Exec(`UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := e.accounts.Authenticate(token); ok || err != nil {
		t.Errorf("expired session accepted: %v %v", ok, err)
	}
}

func TestUpdateProfile(t *testing.T) {
	e := newEnv(t)
	a := e.accounts
	alice, _ := e.register(t, "alice")
	e.register(t, "bob")

	p, err := a.UpdateProfile(alice, domain.ProfileUpdate{
		FirstName:   ptr(" Alice "),
		Bio:         ptr("Loves trains."),
		Language:    ptr("en-GB"),
		Timezone:    ptr("Europe/Berlin"),
		HomeCountry: ptr("DEU"),
		HomeAirport: ptr("BER"),
		DisplayName: ptr(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.FirstName != "Alice" || p.Timezone != "Europe/Berlin" || p.HomeAirport != "BER" || p.DisplayName != "Alice" || p.Username != "alice" {
		t.Errorf("profile = %+v", p)
	}
	if u, _, _ := a.Account(alice); u.Profile != p {
		t.Errorf("not stored: %+v", u.Profile)
	}

	invalid := []domain.ProfileUpdate{
		{Timezone: ptr("Mars/Olympus")},
		{Timezone: ptr("Local")},
		{Language: ptr("german")},
		{HomeCountry: ptr("de")},
		{HomeAirport: ptr("Berlin")},
		{Email: ptr("nope")},
		{Username: ptr("a b")},
	}
	for _, up := range invalid {
		if _, err := a.UpdateProfile(alice, up); !errors.Is(err, service.ErrInvalid) {
			t.Errorf("UpdateProfile(%+v) = %v", up, err)
		}
	}
	if _, err := a.UpdateProfile(alice, domain.ProfileUpdate{Username: ptr("bob")}); !errors.Is(err, service.ErrConflict) {
		t.Errorf("taken username: %v", err)
	}
	if _, err := a.UpdateProfile(alice, domain.ProfileUpdate{Email: ptr("bob@example.com")}); !errors.Is(err, service.ErrConflict) {
		t.Errorf("taken email: %v", err)
	}
}

func TestSaveSettings(t *testing.T) {
	a := newEnv(t).accounts
	s := domain.Settings{
		DistanceUnit:      domain.Miles,
		Theme:             domain.ThemeDark,
		DefaultView:       domain.ViewFlights,
		DefaultVisibility: domain.Public,
		ProfileVisibility: domain.Public,
	}
	if err := a.SaveSettings(local, s); err != nil {
		t.Fatal(err)
	}
	if _, got, _ := a.Account(local); got != s {
		t.Errorf("settings = %+v", got)
	}
	bad := s
	bad.Theme = "neon"
	if err := a.SaveSettings(local, bad); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("bad theme: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	e := newEnv(t)
	a := e.accounts
	alice, current := e.register(t, "alice")
	_, other, _ := a.Login("alice", "correct horse battery")

	if err := a.ChangePassword(alice, current, "wrong password!", "a new password!"); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("wrong current: %v", err)
	}
	if err := a.ChangePassword(alice, current, "correct horse battery", "short"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("short new password: %v", err)
	}
	if err := a.ChangePassword(alice, current, "correct horse battery", "a new password!"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := a.Authenticate(current); !ok {
		t.Error("own session ended")
	}
	if _, ok, _ := a.Authenticate(other); ok {
		t.Error("other session still valid")
	}
	if _, _, err := a.Login("alice", "a new password!"); err != nil {
		t.Errorf("login with new password: %v", err)
	}
}

func TestDeleteAccount(t *testing.T) {
	e := newEnv(t)
	alice, token := e.register(t, "alice")
	if err := e.accounts.DeleteAccount(alice); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := e.accounts.Authenticate(token); ok {
		t.Error("session survived account deletion")
	}
	if _, _, err := e.accounts.Login("alice", "correct horse battery"); !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("login after deletion: %v", err)
	}
}

func TestUsernameLoginWithoutEmail(t *testing.T) {
	a := newEnv(t).accounts

	if taken, err := a.UsernameTaken("carol"); err != nil || taken {
		t.Fatalf("before register: %v %v", taken, err)
	}
	u, _, err := a.Register(service.Registration{Username: "carol", Password: "correct horse battery"})
	if err != nil || u.Profile.Email != "" {
		t.Fatalf("register without email: %+v %v", u.Profile, err)
	}
	if taken, err := a.UsernameTaken("Carol"); err != nil || !taken {
		t.Errorf("after register: %v %v", taken, err)
	}
	if _, err := a.UsernameTaken("a b"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("invalid username: %v", err)
	}
	if _, _, err := a.Login("carol", "correct horse battery"); err != nil {
		t.Errorf("login: %v", err)
	}

	// Two accounts may both leave the email empty
	if _, _, err := a.Register(service.Registration{Username: "dave", Password: "correct horse battery"}); err != nil {
		t.Errorf("second account without email: %v", err)
	}

	p, err := a.UpdateProfile(u.ID, domain.ProfileUpdate{Email: ptr("carol@example.com")})
	if err != nil || p.Email != "carol@example.com" {
		t.Fatalf("set email: %+v %v", p, err)
	}
	if p, err = a.UpdateProfile(u.ID, domain.ProfileUpdate{Email: ptr("")}); err != nil || p.Email != "" {
		t.Errorf("clear email: %+v %v", p, err)
	}
}

func TestOffensiveUsernamesRejected(t *testing.T) {
	a := newEnv(t).accounts

	if _, _, err := a.Register(service.Registration{Username: "H1tler_88", Password: "correct horse battery"}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("register: %v", err)
	}
	if _, err := a.UsernameTaken("nazi_fan"); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("check: %v", err)
	}
	u, _, err := a.Register(service.Registration{Username: "erin", Password: "correct horse battery"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.UpdateProfile(u.ID, domain.ProfileUpdate{Username: ptr("fuck_off")}); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("rename: %v", err)
	}
	// Keeping the current name in an update is not a rename
	if _, err := a.UpdateProfile(u.ID, domain.ProfileUpdate{Username: ptr("erin")}); err != nil {
		t.Errorf("same name: %v", err)
	}
}

func TestProfileColor(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.register(t, "alice")
	for _, bad := range []string{"red", "#12345", "#gggggg", "3a7d5c"} {
		if _, err := e.accounts.UpdateProfile(alice, domain.ProfileUpdate{Color: ptr(bad)}); !errors.Is(err, service.ErrInvalid) {
			t.Errorf("color %q: %v", bad, err)
		}
	}
	p, err := e.accounts.UpdateProfile(alice, domain.ProfileUpdate{Color: ptr("#3A7D5C")})
	if err != nil || p.Color != "#3a7d5c" {
		t.Fatalf("set color: %q %v", p.Color, err)
	}
	if p, err = e.accounts.UpdateProfile(alice, domain.ProfileUpdate{Color: ptr("")}); err != nil || p.Color != "" {
		t.Errorf("clear color: %q %v", p.Color, err)
	}
}
