package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"travel-blog/backend/internal/adapter/http/auth"
	"travel-blog/backend/internal/adapter/http/handler"
	"travel-blog/backend/internal/adapter/http/router"
	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/adapter/storage"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
	configmigrations "travel-blog/backend/migrations/config"
	migrations "travel-blog/backend/migrations/data"
)

type fakeOIDC struct{ enabled, password bool }

func (f fakeOIDC) Enabled() bool       { return f.enabled }
func (f fakeOIDC) Name() string        { return "Pocket ID" }
func (f fakeOIDC) PasswordLogin() bool { return f.password }

func newOIDCServer(t *testing.T, opts ...handler.Option) http.Handler {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "test.db"), migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	configDB, err := sqlite.NewDB(filepath.Join(dir, "config.db"), configmigrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { configDB.Close() })
	settings := sqlite.NewSettingsRepo(configDB)
	files, err := storage.NewLocalStorage(filepath.Join(dir, "photos"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	users := sqlite.NewUserRepo(configDB)
	accounts := service.NewAccountService(users, sqlite.NewSessionRepo(configDB), sqlite.NewUserDataRepo(db), files, settings)
	travel := service.NewTravelService(sqlite.NewCountryEntryRepo(db), sqlite.NewFlightRepo(db), users)
	trips := service.NewTripService(sqlite.NewTripRepo(db), files, users)
	extra := func(mux *http.ServeMux) {
		mux.HandleFunc("GET /api/auth/oidc/login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://provider.example/authorize", http.StatusFound)
		})
	}
	return router.New(handler.New(travel, trips, accounts, opts...), auth.Sessions(accounts), extra)
}

func TestOIDCConfig(t *testing.T) {
	cases := []struct {
		name string
		opts []handler.Option
		want string
	}{
		{"not configured", nil, `"enabled":false`},
		{"enabled", []handler.Option{handler.WithOIDC(fakeOIDC{true, true})}, `"enabled":true,"name":"PocketID","passwordLogin":true`},
		{"password off", []handler.Option{handler.WithOIDC(fakeOIDC{true, false})}, `"passwordLogin":false`},
	}
	for _, tc := range cases {
		rec := do(t, newOIDCServer(t, tc.opts...), "GET", "/api/auth/oidc/config", "", nil)
		body := strings.ReplaceAll(rec.Body.String(), " ", "")
		if rec.Code != http.StatusOK || !strings.Contains(body, tc.want) {
			t.Errorf("%s: %d %s", tc.name, rec.Code, body)
		}
	}
}

func TestOIDCRoutesNeedNoSession(t *testing.T) {
	h := newOIDCServer(t, handler.WithOIDC(fakeOIDC{true, true}))
	rec := do(t, h, "GET", "/api/auth/oidc/login", "", nil)
	if rec.Code != http.StatusFound {
		t.Errorf("login without session: %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/user", "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("protected route: %d", rec.Code)
	}
}

func TestPasswordLoginDisabled(t *testing.T) {
	h := newOIDCServer(t, handler.WithOIDC(fakeOIDC{true, false}))
	login := do(t, h, "POST", "/api/auth/login", `{"username":"admin","password":"whatever12345"}`, nil)
	register := do(t, h, "POST", "/api/auth/register", `{"username":"alice","password":"correct horse battery"}`, nil)
	for name, rec := range map[string]*httptest.ResponseRecorder{"login": login, "register": register} {
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	// Without the switch, registration still works
	open := newOIDCServer(t, handler.WithOIDC(fakeOIDC{true, true}))
	if rec := do(t, open, "POST", "/api/auth/register", `{"username":"alice","password":"correct horse battery"}`, nil); rec.Code != http.StatusCreated {
		t.Errorf("register: %d %s", rec.Code, rec.Body)
	}
}

type fakeRuntime struct{ enabled bool }

func (f *fakeRuntime) Reload(domain.OIDCConfig) error { f.enabled = true; return nil }
func (f *fakeRuntime) Clear()                         { f.enabled = false }
func (f *fakeRuntime) Test(context.Context, domain.OIDCConfig) error {
	return nil
}
func (f *fakeRuntime) Enabled() bool { return f.enabled }

func TestAdminOIDCConfigAPI(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "test.db"), migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	configDB, err := sqlite.NewDB(filepath.Join(dir, "config.db"), configmigrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { configDB.Close() })
	settings := sqlite.NewSettingsRepo(configDB)
	files, err := storage.NewLocalStorage(filepath.Join(dir, "photos"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })
	users := sqlite.NewUserRepo(configDB)
	accounts := service.NewAccountService(users, sqlite.NewSessionRepo(configDB), sqlite.NewUserDataRepo(db), files, settings)
	travel := service.NewTravelService(sqlite.NewCountryEntryRepo(db), sqlite.NewFlightRepo(db), users)
	trips := service.NewTripService(sqlite.NewTripRepo(db), files, users)
	svc := service.NewOIDCConfigService(accounts, settings, &fakeRuntime{}, nil)
	h := router.New(handler.New(travel, trips, accounts, handler.WithOIDCConfig(svc)), auth.Sessions(accounts))

	body := `{"issuer":"https://id.example.com","clientId":"travel","clientSecret":"topsecret","redirectUrl":"https://blog.example/api/auth/oidc/callback","name":"Pocket ID"}`
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/admin/oidc"}, {"PUT", "/api/admin/oidc"}, {"DELETE", "/api/admin/oidc"}, {"POST", "/api/admin/oidc/test"},
	} {
		if rec := do(t, h, tc.method, tc.path, body, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", tc.method, tc.path, rec.Code)
		}
	}
	alice := register(t, h, "alice")
	if rec := do(t, h, "GET", "/api/admin/oidc", "", alice); rec.Code != http.StatusForbidden {
		t.Errorf("get as user: %d", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/admin/oidc", body, alice); rec.Code != http.StatusForbidden {
		t.Errorf("put as user: %d", rec.Code)
	}

	if _, err := configDB.Exec(`UPDATE users SET role = 'admin' WHERE username = 'alice'`); err != nil {
		t.Fatal(err)
	}
	if rec := do(t, h, "POST", "/api/admin/oidc/test", body, alice); rec.Code != http.StatusNoContent {
		t.Fatalf("test: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "PUT", "/api/admin/oidc", `{"issuer":"nope","clientId":"x","redirectUrl":"https://b/cb"}`, alice); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid put: %d", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/admin/oidc", body, alice); rec.Code != http.StatusNoContent {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, "GET", "/api/admin/oidc", "", alice)
	got := strings.ReplaceAll(rec.Body.String(), " ", "")
	if rec.Code != http.StatusOK || strings.Contains(got, "topsecret") || !strings.Contains(got, `"secretSet":true`) ||
		!strings.Contains(got, `"source":"settings"`) || !strings.Contains(got, `"enabled":true`) {
		t.Errorf("get: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "DELETE", "/api/admin/oidc", "", alice); rec.Code != http.StatusNoContent {
		t.Errorf("delete: %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/admin/oidc", "", alice); !strings.Contains(rec.Body.String(), `"source":"none"`) {
		t.Errorf("after delete: %s", rec.Body)
	}
}
