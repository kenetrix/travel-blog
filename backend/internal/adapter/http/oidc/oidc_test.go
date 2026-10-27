package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"travel-blog/backend/internal/adapter/http/oidc"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

// fakeProvider is a minimal OIDC provider: discovery, keys and a token
// endpoint that checks PKCE and issues an ID token for the given claims.
type fakeProvider struct {
	*httptest.Server
	key    *rsa.PrivateKey
	claims map[string]any // overrides for the ID token
	nonce  string         // nonce of the last authorization request
	// challenge is the PKCE challenge of the last authorization request
	challenge string
	tokenErr  bool
}

const clientID = "travel"

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeProvider{key: key, claims: map[string]any{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                f.URL,
			"authorization_endpoint":                f.URL + "/authorize",
			"token_endpoint":                        f.URL + "/token",
			"jwks_uri":                              f.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"},
		}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if f.tokenErr || r.Form.Get("code") != "good" ||
			base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		claims := map[string]any{
			"iss": f.URL, "sub": "user-1", "aud": clientID, "nonce": f.nonce,
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
			"preferred_username": "ada", "email": "ada@example.com", "email_verified": true,
		}
		for k, v := range f.claims {
			claims[k] = v
		}
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
			(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
		if err != nil {
			t.Error(err)
		}
		raw, err := jwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": raw})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

type fakeLogin struct {
	got service.OIDCClaims
	err error
}

func (l *fakeLogin) LoginOIDC(c service.OIDCClaims) (domain.UserID, string, error) {
	l.got = c
	return "u1", "session-token", l.err
}

type harness struct {
	t        *testing.T
	provider *fakeProvider
	login    *fakeLogin
	p        *oidc.Provider
	mux      *http.ServeMux
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	fp := newFakeProvider(t)
	login := &fakeLogin{}
	p := oidc.New(oidc.Config{
		Issuer: fp.URL, ClientID: clientID, ClientSecret: "secret",
		RedirectURL: "https://blog.example/api/auth/oidc/callback", Name: "Pocket ID",
		Scopes: []string{"openid", "profile", "email"},
	}, login)
	mux := http.NewServeMux()
	p.Routes(mux)
	return &harness{t: t, provider: fp, login: login, p: p, mux: mux}
}

func (h *harness) get(path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

// start performs GET /login and returns the flow cookie and the state; query
// is appended to the login URL.
func (h *harness) start(next string, query ...string) (*http.Cookie, string) {
	h.t.Helper()
	rec := h.get("/api/auth/oidc/login?next=" + url.QueryEscape(next) + strings.Join(query, ""))
	if rec.Code != http.StatusFound {
		h.t.Fatalf("login status %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || !strings.HasPrefix(loc.String(), h.provider.URL+"/authorize") {
		h.t.Fatalf("redirect to %q", loc)
	}
	q := loc.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" ||
		q.Get("nonce") == "" || q.Get("state") == "" || q.Get("client_id") != clientID ||
		q.Get("redirect_uri") != "https://blog.example/api/auth/oidc/callback" {
		h.t.Fatalf("authorization request: %v", q)
	}
	h.provider.nonce, h.provider.challenge = q.Get("nonce"), q.Get("code_challenge")
	var flow *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "oidc_flow" {
			flow = c
		}
	}
	if flow == nil || !flow.HttpOnly || !flow.Secure || flow.SameSite != http.SameSiteLaxMode {
		h.t.Fatalf("flow cookie: %+v", flow)
	}
	return flow, q.Get("state")
}

func location(rec *httptest.ResponseRecorder) string { return rec.Header().Get("Location") }

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

func TestLoginSuccess(t *testing.T) {
	h := newHarness(t)
	flow, state := h.start("/profile")
	rec := h.get("/api/auth/oidc/callback?code=good&state="+state, flow)
	if rec.Code != http.StatusFound || location(rec) != "/profile" {
		t.Fatalf("callback: %d %q", rec.Code, location(rec))
	}
	if c := sessionCookie(rec); c == nil || c.Value != "session-token" {
		t.Errorf("session cookie: %+v", c)
	}
	got := h.login.got
	if got.Issuer != h.provider.URL || got.Subject != "user-1" || got.Email != "ada@example.com" ||
		!got.EmailVerified || got.PreferredUsername != "ada" {
		t.Errorf("claims: %+v", got)
	}
}

func TestLoginPassesUILanguage(t *testing.T) {
	h := newHarness(t)
	h.provider.claims["locale"] = "fr-FR"
	flow, state := h.start("/", "&lang=de")
	if rec := h.get("/api/auth/oidc/callback?code=good&state="+state, flow); rec.Code != http.StatusFound {
		t.Fatalf("callback: %d", rec.Code)
	}
	if got := h.login.got; got.Locale != "fr-FR" || got.UILanguage != "de" {
		t.Errorf("claims: %+v", got)
	}
}

func assertFailed(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	if rec.Code != http.StatusFound || location(rec) != "/login?error="+code {
		t.Errorf("want redirect to error %s, got %d %q", code, rec.Code, location(rec))
	}
	if sessionCookie(rec) != nil {
		t.Error("session cookie set on failure")
	}
}

func TestCallbackRejections(t *testing.T) {
	cases := []struct {
		name  string
		setup func(h *harness)
		query func(state string) string
		flow  func(c *http.Cookie) *http.Cookie
	}{
		{name: "wrong state", query: func(string) string { return "code=good&state=other" }},
		{name: "no cookie", query: func(s string) string { return "code=good&state=" + s },
			flow: func(*http.Cookie) *http.Cookie { return nil }},
		{name: "forged cookie", query: func(s string) string { return "code=good&state=" + s },
			flow: func(c *http.Cookie) *http.Cookie {
				f := *c
				f.Value = strings.Replace(c.Value, ".", "x.", 1)
				return &f
			}},
		{name: "provider error", query: func(s string) string { return "error=access_denied&state=" + s }},
		{name: "code rejected", query: func(s string) string { return "code=bad&state=" + s }},
		{name: "wrong nonce", setup: func(h *harness) { h.provider.claims["nonce"] = "not-mine" },
			query: func(s string) string { return "code=good&state=" + s }},
		{name: "expired token", setup: func(h *harness) { h.provider.claims["exp"] = time.Now().Add(-time.Hour).Unix() },
			query: func(s string) string { return "code=good&state=" + s }},
		{name: "wrong audience", setup: func(h *harness) { h.provider.claims["aud"] = "someone-else" },
			query: func(s string) string { return "code=good&state=" + s }},
		{name: "wrong issuer", setup: func(h *harness) { h.provider.claims["iss"] = "https://evil.example" },
			query: func(s string) string { return "code=good&state=" + s }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			flow, state := h.start("/")
			if tc.setup != nil {
				tc.setup(h)
			}
			if tc.flow != nil {
				flow = tc.flow(flow)
			}
			var cookies []*http.Cookie
			if flow != nil {
				cookies = append(cookies, flow)
			}
			assertFailed(t, h.get("/api/auth/oidc/callback?"+tc.query(state), cookies...), "oidc_failed")
		})
	}
}

func TestCallbackClearsFlowCookie(t *testing.T) {
	h := newHarness(t)
	flow, state := h.start("/")
	rec := h.get("/api/auth/oidc/callback?code=good&state="+state, flow)
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "oidc_flow" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("flow cookie not cleared, callback could be replayed")
	}
}

func TestLoginErrorsMapToCodes(t *testing.T) {
	for err, code := range map[error]string{
		service.ErrForbidden:       "registration_closed",
		service.ErrEmailUnverified: "email_unverified",
		errors.New("boom"):         "oidc_failed",
	} {
		h := newHarness(t)
		h.login.err = err
		flow, state := h.start("/")
		assertFailed(t, h.get("/api/auth/oidc/callback?code=good&state="+state, flow), code)
	}
}

func TestNextMustBeRelative(t *testing.T) {
	for _, next := range []string{"https://evil.example", "//evil.example", "/\\evil.example", "evil"} {
		h := newHarness(t)
		flow, state := h.start(next)
		rec := h.get("/api/auth/oidc/callback?code=good&state="+state, flow)
		if location(rec) != "/" {
			t.Errorf("next %q redirects to %q", next, location(rec))
		}
	}
}

func TestDisabledWithoutProvider(t *testing.T) {
	// Unreachable provider: OIDC stays off, the server keeps running
	p := oidc.New(oidc.Config{Issuer: "http://127.0.0.1:1", ClientID: "x", ClientSecret: "y",
		RedirectURL: "http://localhost/cb", DisablePassword: true}, &fakeLogin{})
	if p.Enabled() {
		t.Fatal("enabled without provider")
	}
	if !p.PasswordLogin() {
		t.Error("password login must stay on while OIDC is not working")
	}
	mux := http.NewServeMux()
	p.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("login status %d", rec.Code)
	}
}

func TestPasswordLoginDisabled(t *testing.T) {
	h := newHarness(t)
	if !h.p.PasswordLogin() {
		t.Error("password login should be on by default")
	}
	cfg, _ := oidc.ConfigFromEnv(func(k string) string {
		return map[string]string{
			"OIDC_ISSUER": h.provider.URL, "OIDC_CLIENT_ID": clientID, "OIDC_CLIENT_SECRET": "s",
			"OIDC_REDIRECT_URL": "http://localhost/cb", "OIDC_DISABLE_PASSWORD": "true",
		}[k]
	})
	p := oidc.New(cfg, &fakeLogin{})
	if !p.Enabled() || p.PasswordLogin() || p.Name() != "SSO" {
		t.Errorf("enabled=%v password=%v name=%q", p.Enabled(), p.PasswordLogin(), p.Name())
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{"OIDC_ISSUER": "https://id", "OIDC_CLIENT_ID": "a", "OIDC_CLIENT_SECRET": "b"}
	get := func(k string) string { return env[k] }
	if _, ok := oidc.ConfigFromEnv(get); ok {
		t.Error("incomplete configuration accepted")
	}
	env["OIDC_REDIRECT_URL"] = "https://blog/api/auth/oidc/callback"
	env["OIDC_NAME"] = "Pocket ID"
	cfg, ok := oidc.ConfigFromEnv(get)
	if !ok || cfg.Name != "Pocket ID" || len(cfg.Scopes) != 3 || cfg.DisablePassword {
		t.Errorf("config: %+v %v", cfg, ok)
	}
}

func TestReloadAndClear(t *testing.T) {
	h := newHarness(t)
	second := newFakeProvider(t)
	cfg := oidc.Config{Issuer: second.URL, ClientID: clientID, ClientSecret: "s",
		RedirectURL: "https://blog.example/api/auth/oidc/callback", Name: "Second"}
	if err := h.p.Reload(cfg); err != nil || h.p.Name() != "Second" {
		t.Fatalf("reload: %v %q", err, h.p.Name())
	}
	rec := h.get("/api/auth/oidc/login")
	if !strings.HasPrefix(location(rec), second.URL+"/authorize") {
		t.Errorf("login redirects to %q", location(rec))
	}

	// A failing reload keeps the running configuration
	bad := cfg
	bad.Issuer = "http://127.0.0.1:1"
	bad.Name = "Broken"
	if err := h.p.Reload(bad); err == nil {
		t.Fatal("reload of an unreachable provider succeeded")
	}
	if !h.p.Enabled() || h.p.Name() != "Second" {
		t.Errorf("config replaced: enabled=%v name=%q", h.p.Enabled(), h.p.Name())
	}
	if err := h.p.Test(context.Background(), bad); err == nil {
		t.Error("test of an unreachable provider succeeded")
	}

	h.p.Clear()
	if h.p.Enabled() || h.get("/api/auth/oidc/login").Code != http.StatusNotFound || !h.p.PasswordLogin() {
		t.Error("clear did not switch OIDC off")
	}
}

func TestFlowStartedBeforeReloadFails(t *testing.T) {
	h := newHarness(t)
	flow, state := h.start("/")
	second := newFakeProvider(t)
	if err := h.p.Reload(oidc.Config{Issuer: second.URL, ClientID: clientID, ClientSecret: "s",
		RedirectURL: "https://blog.example/api/auth/oidc/callback", Name: "Second"}); err != nil {
		t.Fatal(err)
	}
	// The old provider's code means nothing to the new one
	assertFailed(t, h.get("/api/auth/oidc/callback?code=good&state="+state, flow), "oidc_failed")
}
