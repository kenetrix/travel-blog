package router_test

import (
	"encoding/json"
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
	accountmigrations "travel-blog/backend/migrations/accounts"
	configmigrations "travel-blog/backend/migrations/config"
	migrations "travel-blog/backend/migrations/data"
)

// newServer wires the real stack.
func newServer(t *testing.T) http.Handler {
	h, _ := newStack(t)
	return h
}

// newStack is newServer plus the account service, to set up administrators.
func newStack(t *testing.T) (http.Handler, *service.AccountService) {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "test.db"), migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	accountsDB, err := sqlite.NewDB(filepath.Join(dir, "accounts.db"), accountmigrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { accountsDB.Close() })
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

	users := sqlite.NewUserRepo(accountsDB)
	accounts := service.NewAccountService(users, sqlite.NewSessionRepo(accountsDB), sqlite.NewUserDataRepo(db), files, settings)
	travel := service.NewTravelService(sqlite.NewCountryEntryRepo(db), sqlite.NewFlightRepo(db), users)
	trips := service.NewTripService(sqlite.NewTripRepo(db), files, users)
	return router.New(handler.New(travel, trips, accounts), auth.Sessions(accounts)), accounts
}

func do(t *testing.T, h http.Handler, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// register creates an account and returns the Cookie header for it.
func register(t *testing.T, h http.Handler, username string) map[string]string {
	t.Helper()
	rec := do(t, h, "POST", "/api/auth/register",
		`{"email":"`+username+`@example.com","username":"`+username+`","password":"correct horse battery"}`, nil)
	if rec.Code != 201 {
		t.Fatalf("register %s: %d %s", username, rec.Code, rec.Body)
	}
	return map[string]string{"Cookie": sessionCookie(t, rec)}
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Errorf("insecure session cookie: %+v", c)
			}
			return c.Name + "=" + c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func usernameOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var v struct{ Profile struct{ Username string } }
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("%s: %v", rec.Body, err)
	}
	return v.Profile.Username
}

func TestTravelFlow(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")

	rec := do(t, h, "GET", "/api/user", "", me)
	if rec.Code != 200 || usernameOf(t, rec) != "mia" {
		t.Fatalf("account: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "private, no-cache" {
		t.Error("missing Cache-Control: private, no-cache")
	}
	for _, path := range []string{"/api/user", "/api/countries", "/api/flights"} {
		rec := do(t, h, "GET", path, "", me)
		etag := rec.Header().Get("ETag")
		if rec.Code != 200 || etag == "" {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body)
		}
		rec = do(t, h, "GET", path, "", map[string]string{"If-None-Match": etag, "Cookie": me["Cookie"]})
		if rec.Code != 304 || rec.Body.Len() != 0 {
			t.Errorf("revalidating %s: %d %q", path, rec.Code, rec.Body)
		}
	}
	if body := do(t, h, "GET", "/api/countries", "", me).Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("empty countries: %s", body)
	}
	etag := do(t, h, "GET", "/api/flights", "", me).Header().Get("ETag")

	rec = do(t, h, "POST", "/api/flights", `{"from":"BER","to":"HND","countries":["DEU","JPN"]}`, me)
	if rec.Code != 201 {
		t.Fatalf("create flight: %d %s", rec.Code, rec.Body)
	}
	var created struct {
		Id               string
		Visibility       string
		CountriesUpdated []string
	}
	json.Unmarshal(rec.Body.Bytes(), &created)
	if strings.Join(created.CountriesUpdated, ",") != "DEU,JPN" || created.Visibility != "private" {
		t.Errorf("created = %+v", created)
	}
	if newTag := rec.Header().Get("ETag"); newTag == "" || newTag == etag {
		t.Errorf("ETag not updated: %q", newTag)
	}

	rec = do(t, h, "PATCH", "/api/flights/"+created.Id, `{"visibility":"public"}`, me)
	if rec.Code != 204 || rec.Header().Get("ETag") == "" {
		t.Errorf("update flight: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "PATCH", "/api/flights/"+created.Id, `{"flightNumber":"nh 204","date":"2026-03-14","durationMinutes":710,"aircraft":"Boeing 787-9"}`, me)
	if rec.Code != 204 {
		t.Errorf("flight details: %d %s", rec.Code, rec.Body)
	}
	body := do(t, h, "GET", "/api/flights", "", me).Body.String()
	for _, want := range []string{`"flightNumber":"NH204"`, `"date":"2026-03-14"`, `"durationMinutes":710`, `"aircraft":"Boeing 787-9"`, `"visibility":"public"`} {
		if !strings.Contains(body, want) {
			t.Errorf("flights miss %s: %s", want, body)
		}
	}
	for _, invalid := range []string{`{}`, `{"flightNumber":"400"}`, `{"date":"2026-02-30"}`, `{"durationMinutes":2000}`} {
		if rec := do(t, h, "PATCH", "/api/flights/"+created.Id, invalid, me); rec.Code != 400 {
			t.Errorf("PATCH %s: %d, want 400", invalid, rec.Code)
		}
	}
	rec = do(t, h, "PATCH", "/api/flights/"+created.Id, `{"date":""}`, me)
	if body := do(t, h, "GET", "/api/flights", "", me).Body.String(); rec.Code != 204 || strings.Contains(body, `"date"`) || !strings.Contains(body, `"NH204"`) {
		t.Errorf("clear date: %d %s", rec.Code, body)
	}

	rec = do(t, h, "PUT", "/api/countries/JPN", `{"status":"none","details":{"food":"","sightseeing":"","notes":"","rating":0,"tags":[]}}`, me)
	if rec.Code != 204 {
		t.Errorf("remove country: %d %s", rec.Code, rec.Body)
	}

	// Profile and settings changes change the account's ETag
	etag = do(t, h, "GET", "/api/user", "", me).Header().Get("ETag")
	rec = do(t, h, "PUT", "/api/user/settings", `{"distanceUnit":"mi","theme":"dark","defaultView":"flights","defaultVisibility":"public","profileVisibility":"public"}`, me)
	if rec.Code != 204 || rec.Header().Get("ETag") == etag {
		t.Errorf("save settings: %d %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "PATCH", "/api/user/profile", `{"firstName":"Ken","timezone":"Europe/Berlin"}`, me)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"timezone":"Europe/Berlin"`) {
		t.Errorf("update profile: %d %s", rec.Code, rec.Body)
	}
}

func TestLoginRequired(t *testing.T) {
	h := newServer(t)

	for _, path := range []string{"/api/user", "/api/countries", "/api/flights", "/api/user/settings"} {
		if rec := do(t, h, "GET", path, "", nil); rec.Code != 401 {
			t.Errorf("%s without session: %d", path, rec.Code)
		}
	}
	if rec := do(t, h, "GET", "/api/user", "", map[string]string{"Cookie": "session=forged"}); rec.Code != 401 {
		t.Errorf("forged session: %d", rec.Code)
	}

	alice := register(t, h, "alice")
	rec := do(t, h, "GET", "/api/user", "", alice)
	if rec.Code != 200 || usernameOf(t, rec) != "alice" {
		t.Fatalf("account with session: %d %s", rec.Code, rec.Body)
	}

	if rec := do(t, h, "GET", "/api/auth/users/alice", "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"exists":true`) {
		t.Errorf("existing username: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/auth/users/nobody", "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"exists":false`) {
		t.Errorf("unknown username: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/auth/users/a%20b", "", nil); rec.Code != 400 {
		t.Errorf("invalid username: %d", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/auth/login", `{"username":"alice","password":"nope nope nope"}`, nil); rec.Code != 401 {
		t.Errorf("wrong password: %d", rec.Code)
	}
	rec = do(t, h, "POST", "/api/auth/login", `{"username":"alice","password":"correct horse battery"}`, nil)
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	second := map[string]string{"Cookie": sessionCookie(t, rec)}

	rec = do(t, h, "POST", "/api/auth/logout", "", second)
	if rec.Code != 204 || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("logout: %d %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
	if rec := do(t, h, "GET", "/api/user", "", second); rec.Code != 401 {
		t.Errorf("session valid after logout: %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/user", "", alice); rec.Code != 200 {
		t.Errorf("other session ended by logout: %d", rec.Code)
	}

	if rec := do(t, h, "PUT", "/api/user/password", `{"currentPassword":"wrong one!!","newPassword":"a new password!"}`, alice); rec.Code != 403 {
		t.Errorf("wrong current password: %d", rec.Code)
	}
	if rec := do(t, h, "DELETE", "/api/user", "", alice); rec.Code != 204 {
		t.Fatalf("delete account: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/user", "", alice); rec.Code != 401 {
		t.Errorf("session valid after account deletion: %d", rec.Code)
	}
}

func TestTravelPerUser(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")
	alice := register(t, h, "alice")

	mine := do(t, h, "GET", "/api/user", "", me).Header().Get("ETag")
	if rec := do(t, h, "GET", "/api/user", "", alice); rec.Header().Get("ETag") == mine {
		t.Error("ETags of different users must differ")
	}
	if rec := do(t, h, "POST", "/api/flights", `{"from":"BER","to":"HND","countries":["DEU","JPN"]}`, alice); rec.Code != 201 {
		t.Fatalf("alice's flight: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"/api/countries", "/api/flights"} {
		if body := do(t, h, "GET", path, "", me).Body.String(); strings.TrimSpace(body) != "[]" {
			t.Errorf("me sees alice's data at %s: %s", path, body)
		}
	}
}

func TestPublicTravel(t *testing.T) {
	h := newServer(t)
	alice := register(t, h, "alice")
	do(t, h, "PUT", "/api/countries/JPN", `{"status":"visited","visibility":"public","details":{"food":"","sightseeing":"","notes":"","rating":0,"tags":[]}}`, alice)
	do(t, h, "PUT", "/api/countries/FRA", `{"status":"visited","details":{"food":"","sightseeing":"","notes":"","rating":0,"tags":[]}}`, alice)

	if rec := do(t, h, "GET", "/api/public/alice", "", nil); rec.Code != 404 {
		t.Errorf("private profile: %d", rec.Code)
	}
	do(t, h, "PUT", "/api/user/settings", `{"distanceUnit":"km","theme":"system","defaultView":"countries","defaultVisibility":"private","profileVisibility":"public"}`, alice)

	rec := do(t, h, "GET", "/api/public/alice", "", nil)
	if rec.Code != 200 {
		t.Fatalf("public profile: %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"JPN"`) || strings.Contains(body, `"FRA"`) || strings.Contains(body, "@example.com") {
		t.Errorf("public view leaks or misses data: %s", body)
	}

	// Public flights show the route, not when or on which flight
	rec = do(t, h, "POST", "/api/flights", `{"from":"BER","to":"HND","countries":["DEU","JPN"],"visibility":"public"}`, alice)
	var created struct{ Id string }
	json.Unmarshal(rec.Body.Bytes(), &created)
	do(t, h, "PATCH", "/api/flights/"+created.Id, `{"flightNumber":"NH204","date":"2026-03-14","durationMinutes":710,"aircraft":"Boeing 787-9"}`, alice)
	body = do(t, h, "GET", "/api/public/alice", "", nil).Body.String()
	if !strings.Contains(body, `"HND"`) {
		t.Errorf("public flight missing: %s", body)
	}
	for _, field := range []string{"flightNumber", "date", "durationMinutes", "aircraft"} {
		if strings.Contains(body, `"`+field+`"`) {
			t.Errorf("public view leaks %s: %s", field, body)
		}
	}
}

func TestProblems(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")
	register(t, h, "alice")
	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/api/flights", `{"from":"BER","to":"BER","countries":["DEU"]}`, 422},
		{"POST", "/api/flights", `{"from":"BER","to":"???","countries":[]}`, 400},
		{"POST", "/api/flights", `{"from":"BER","to":"HND","countries":["de"]}`, 400},
		{"POST", "/api/flights", `{"from":"BER","to":"HND","countries":[],"visibility":"friends"}`, 400},
		{"POST", "/api/flights", `not json`, 400},
		{"PUT", "/api/countries/deu", `{"status":"visited","details":{"food":"","sightseeing":"","notes":"","rating":0,"tags":[]}}`, 400},
		{"PUT", "/api/countries/DEU", `{"status":"visited","details":{"food":"","sightseeing":"","notes":"","rating":9,"tags":[]}}`, 400},
		{"DELETE", "/api/flights/00000000-0000-0000-0000-000000000000", "", 404},
		{"PATCH", "/api/flights/00000000-0000-0000-0000-000000000000", `{"visibility":"public"}`, 404},
		{"DELETE", "/api/flights/not-a-uuid", "", 400},
		{"POST", "/api/auth/register", `{"email":"alice@example.com","username":"other","password":"correct horse battery"}`, 409},
		{"POST", "/api/auth/register", `{"email":"x@example.com","username":"x","password":"correct horse battery"}`, 400},
		{"POST", "/api/auth/register", `{"email":"not an email","username":"xyz","password":"correct horse battery"}`, 400},
		{"PATCH", "/api/user/profile", `{"timezone":"Mars/Olympus"}`, 400},
		{"PATCH", "/api/user/profile", `{"username":"alice"}`, 409},
		{"PUT", "/api/user/settings", `{"distanceUnit":"ly","theme":"system","defaultView":"countries","defaultVisibility":"private","profileVisibility":"private"}`, 400},
		{"GET", "/api/public/nobody", "", 404},
	}
	for _, c := range cases {
		rec := do(t, h, c.method, c.path, c.body, me)
		if rec.Code != c.want || rec.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s %s: %d %s %s, want %d problem+json", c.method, c.path, rec.Code, rec.Header().Get("Content-Type"), rec.Body, c.want)
		}
	}
}

func TestProblemCodes(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")
	register(t, h, "alice")
	cases := []struct {
		method, path, body string
		header             map[string]string
		code               string
		params             map[string]string
	}{
		{"POST", "/api/auth/register", `{"username":"alice","password":"correct horse battery"}`, nil, "username_taken", nil},
		{"PATCH", "/api/user/profile", `{"language":"fr"}`, me, "invalid_language", map[string]string{"value": "fr"}},
		{"POST", "/api/flights", `{"from":"BER","to":"BER","countries":["DEU"]}`, me, "same_airport", nil},
		{"POST", "/api/flights", `not json`, me, "bad_request", nil},
		{"GET", "/api/flights", "", nil, "login_required", nil},
	}
	for _, c := range cases {
		rec := do(t, h, c.method, c.path, c.body, c.header)
		var p struct {
			Code   string            `json:"code"`
			Detail string            `json:"detail"`
			Params map[string]string `json:"params"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Code != c.code || p.Detail == "" {
			t.Errorf("%s %s: %s, want code %s", c.method, c.path, rec.Body, c.code)
		}
		for k, v := range c.params {
			if p.Params[k] != v {
				t.Errorf("%s %s: params %v, want %s=%s", c.method, c.path, p.Params, k, v)
			}
		}
	}
}

func TestTravelOmitsTextsAndCountryHasThem(t *testing.T) {
	h := newServer(t)
	me := register(t, h, "mia")
	rec := do(t, h, "PUT", "/api/countries/DEU",
		`{"status":"visited","details":{"food":"Brezel","sightseeing":"","notes":"long","rating":4,"tags":["city trip"]}}`, me)
	if rec.Code != 200 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}

	var entries []map[string]any
	json.Unmarshal(do(t, h, "GET", "/api/countries", "", me).Body.Bytes(), &entries)
	if len(entries) != 1 || entries[0]["rating"] != float64(4) {
		t.Fatalf("entries = %v", entries)
	}
	for _, k := range []string{"details", "food", "notes"} {
		if _, ok := entries[0][k]; ok {
			t.Errorf("summary has %q", k)
		}
	}

	rec = do(t, h, "GET", "/api/countries/DEU", "", me)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"notes":"long"`) {
		t.Errorf("country: %d %s", rec.Code, rec.Body)
	}
	if rec = do(t, h, "GET", "/api/countries/FRA", "", me); rec.Code != 404 {
		t.Errorf("missing country: %d", rec.Code)
	}
}
