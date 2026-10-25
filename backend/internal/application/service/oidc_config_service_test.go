package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

type fakeRuntime struct {
	cfg     domain.OIDCConfig
	enabled bool
	fail    bool
}

func (f *fakeRuntime) Reload(cfg domain.OIDCConfig) error {
	if f.fail {
		return errors.New("discovery failed")
	}
	f.cfg, f.enabled = cfg, true
	return nil
}
func (f *fakeRuntime) Clear() { f.cfg, f.enabled = domain.OIDCConfig{}, false }
func (f *fakeRuntime) Test(_ context.Context, _ domain.OIDCConfig) error {
	if f.fail {
		return errors.New("discovery failed")
	}
	return nil
}
func (f *fakeRuntime) Enabled() bool { return f.enabled }

// reverseBox "encrypts" by reversing and tagging, enough to see the
// database never holds the plain secret.
type reverseBox struct{}

func (reverseBox) Encrypt(s string) (string, error) {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return "sealed:" + string(r), nil
}
func (b reverseBox) Decrypt(s string) (string, error) {
	plain, _ := b.Encrypt(strings.TrimPrefix(s, "sealed:"))
	return strings.TrimPrefix(plain, "sealed:"), nil
}

type oidcAdminEnv struct {
	*env
	svc     *service.OIDCConfigService
	runtime *fakeRuntime
	admin   domain.UserID
	user    domain.UserID
}

func newOIDCAdminEnv(t *testing.T, fromEnv *domain.OIDCConfig) *oidcAdminEnv {
	t.Helper()
	e := newEnv(t)
	admin, _ := e.register(t, "boss")
	if _, err := e.db.Exec(`UPDATE users SET role = 'admin' WHERE id = ?`, admin); err != nil {
		t.Fatal(err)
	}
	user, _ := e.register(t, "alice")
	rt := &fakeRuntime{}
	svc := service.NewOIDCConfigService(e.accounts, sqlite.NewOIDCSettingsRepo(e.db), reverseBox{}, rt, fromEnv)
	return &oidcAdminEnv{env: e, svc: svc, runtime: rt, admin: admin, user: user}
}

var validInput = service.OIDCConfigInput{
	Issuer: "https://id.example.com/", ClientID: "travel", ClientSecret: "s3cret",
	RedirectURL: "https://blog.example/api/auth/oidc/callback", Name: "Pocket ID",
}

func TestOIDCAdminOnly(t *testing.T) {
	e := newOIDCAdminEnv(t, nil)
	if _, err := e.svc.View(e.user); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("view: %v", err)
	}
	if err := e.svc.Save(e.user, validInput); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("save: %v", err)
	}
	if err := e.svc.Test(e.user, validInput); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("test: %v", err)
	}
	if err := e.svc.Delete(e.user); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("delete: %v", err)
	}
}

func TestOIDCSaveAppliesAndHidesSecret(t *testing.T) {
	e := newOIDCAdminEnv(t, nil)
	if err := e.svc.Save(e.admin, validInput); err != nil {
		t.Fatal(err)
	}
	if !e.runtime.enabled || e.runtime.cfg.Issuer != "https://id.example.com" || e.runtime.cfg.ClientSecret != "s3cret" ||
		strings.Join(e.runtime.cfg.Scopes, " ") != "openid profile email" {
		t.Errorf("runtime: %+v", e.runtime.cfg)
	}
	var stored string
	if err := e.db.QueryRow(`SELECT value FROM oidc_settings WHERE key = 'client_secret'`).Scan(&stored); err != nil || stored == "s3cret" {
		t.Errorf("stored secret %q %v", stored, err)
	}
	view, err := e.svc.View(e.admin)
	if err != nil || view.Source != "settings" || !view.SecretSet || !view.Enabled ||
		view.Issuer != "https://id.example.com" || view.Name != "Pocket ID" {
		t.Errorf("view: %+v %v", view, err)
	}
	cfg, ok, err := e.svc.Effective()
	if err != nil || !ok || cfg.ClientSecret != "s3cret" {
		t.Errorf("effective: %+v %v %v", cfg, ok, err)
	}
}

func TestOIDCSaveWithoutSecretKeepsOld(t *testing.T) {
	e := newOIDCAdminEnv(t, nil)
	if err := e.svc.Save(e.admin, validInput); err != nil {
		t.Fatal(err)
	}
	in := validInput
	in.ClientSecret, in.Name = "", "Login"
	if err := e.svc.Save(e.admin, in); err != nil {
		t.Fatal(err)
	}
	if e.runtime.cfg.ClientSecret != "s3cret" || e.runtime.cfg.Name != "Login" {
		t.Errorf("runtime: %+v", e.runtime.cfg)
	}
	// ... but the first save needs one
	fresh := newOIDCAdminEnv(t, nil)
	if err := fresh.svc.Save(fresh.admin, in); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("first save without secret: %v", err)
	}
}

func TestOIDCSaveRejectsBadInput(t *testing.T) {
	for name, mod := range map[string]func(*service.OIDCConfigInput){
		"issuer":   func(i *service.OIDCConfigInput) { i.Issuer = "id.example.com" },
		"redirect": func(i *service.OIDCConfigInput) { i.RedirectURL = "javascript:alert(1)" },
		"client":   func(i *service.OIDCConfigInput) { i.ClientID = " " },
		"name":     func(i *service.OIDCConfigInput) { i.Name = strings.Repeat("x", 41) },
	} {
		e := newOIDCAdminEnv(t, nil)
		in := validInput
		mod(&in)
		if err := e.svc.Save(e.admin, in); !errors.Is(err, service.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
		if e.runtime.enabled {
			t.Errorf("%s: runtime changed", name)
		}
	}
}

func TestOIDCFailedDiscoveryKeepsRunningConfig(t *testing.T) {
	e := newOIDCAdminEnv(t, nil)
	if err := e.svc.Save(e.admin, validInput); err != nil {
		t.Fatal(err)
	}
	e.runtime.fail = true
	in := validInput
	in.Issuer, in.DisablePassword = "https://other.example.com", true
	if err := e.svc.Save(e.admin, in); !errors.Is(err, service.ErrInvalid) {
		t.Fatalf("save: %v", err)
	}
	if e.runtime.cfg.Issuer != "https://id.example.com" {
		t.Errorf("runtime replaced: %+v", e.runtime.cfg)
	}
	if view, _ := e.svc.View(e.admin); view.Issuer != "https://id.example.com" || view.DisablePassword {
		t.Errorf("stored config replaced: %+v", view)
	}
	if err := e.svc.Test(e.admin, in); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("test: %v", err)
	}
	e.runtime.fail = false
	if err := e.svc.Test(e.admin, in); err != nil {
		t.Errorf("test ok: %v", err)
	}
	if view, _ := e.svc.View(e.admin); view.Issuer != "https://id.example.com" {
		t.Error("test must not save")
	}
}

func TestOIDCDelete(t *testing.T) {
	e := newOIDCAdminEnv(t, nil)
	if err := e.svc.Save(e.admin, validInput); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Delete(e.admin); err != nil {
		t.Fatal(err)
	}
	view, _ := e.svc.View(e.admin)
	if e.runtime.enabled || view.Source != "none" || view.SecretSet {
		t.Errorf("after delete: %+v enabled=%v", view, e.runtime.enabled)
	}
	if _, ok, _ := e.svc.Effective(); ok {
		t.Error("effective config after delete")
	}
}

func TestOIDCEnvWins(t *testing.T) {
	envCfg := &domain.OIDCConfig{Issuer: "https://env.example", ClientID: "e", ClientSecret: "es",
		RedirectURL: "https://blog/cb", Name: "Env", Scopes: []string{"openid"}, DisablePassword: true}
	e := newOIDCAdminEnv(t, envCfg)
	view, err := e.svc.View(e.admin)
	if err != nil || view.Source != "env" || view.Issuer != "https://env.example" || !view.SecretSet || !view.DisablePassword {
		t.Errorf("view: %+v %v", view, err)
	}
	if err := e.svc.Save(e.admin, validInput); !errors.Is(err, service.ErrConflict) {
		t.Errorf("save: %v", err)
	}
	if err := e.svc.Delete(e.admin); !errors.Is(err, service.ErrConflict) {
		t.Errorf("delete: %v", err)
	}
	cfg, ok, err := e.svc.Effective()
	if err != nil || !ok || cfg.Issuer != "https://env.example" {
		t.Errorf("effective: %+v %v %v", cfg, ok, err)
	}
}
