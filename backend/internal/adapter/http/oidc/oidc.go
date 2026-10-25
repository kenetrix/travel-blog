// Package oidc runs OpenID Connect login (authorization code flow with PKCE) against a
// configurable provider. Endpoints come from discovery; afterwards the normal session is used.
package oidc

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"travel-blog/backend/internal/adapter/http/auth"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
)

const (
	flowCookie = "oidc_flow"
	flowPath   = "/api/auth/oidc"
	flowTTL    = 10 * time.Minute
	retryAfter = 30 * time.Second
)

// Error codes the login page gets as ?error=...; raw provider texts never reach
// the browser.
const (
	ErrFailed             = "oidc_failed"
	ErrRegistrationClosed = "registration_closed"
	ErrEmailUnverified    = "email_unverified"
)

type Config = domain.OIDCConfig

// ConfigFromEnv reads the configuration; ok is false if it is incomplete
// (OIDC then stays off).
func ConfigFromEnv(getenv func(string) string) (c Config, ok bool) {
	c = Config{
		Issuer:          strings.TrimSpace(getenv("OIDC_ISSUER")),
		ClientID:        strings.TrimSpace(getenv("OIDC_CLIENT_ID")),
		ClientSecret:    getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:     strings.TrimSpace(getenv("OIDC_REDIRECT_URL")),
		Name:            strings.TrimSpace(getenv("OIDC_NAME")),
		Scopes:          strings.Fields(getenv("OIDC_SCOPES")),
		DisablePassword: strings.EqualFold(strings.TrimSpace(getenv("OIDC_DISABLE_PASSWORD")), "true"),
	}
	if c.Name == "" {
		c.Name = "SSO"
	}
	if len(c.Scopes) == 0 {
		c.Scopes = defaultScopes()
	}
	return c, c.Complete()
}

type Login interface {
	LoginOIDC(service.OIDCClaims) (domain.UserID, string, error)
}

// Provider is an OIDC provider whose configuration can change at runtime.
// Without a configuration it is off.
type Provider struct {
	login Login
	key   []byte // signs the flow cookie; random per process

	mu       sync.Mutex
	cfg      Config
	oauth    *oauth2.Config
	verifier *gooidc.IDTokenVerifier
	lastTry  time.Time
}

// New sets up the provider; an empty configuration turns it off. If the provider is
// unreachable, OIDC stays off and is retried later (see Enabled).
func New(cfg Config, login Login) *Provider {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("oidc: random key: %v", err)
	}
	p := &Provider{login: login, key: key}
	p.Set(cfg)
	return p
}

// Set takes the configuration without failing on an unreachable provider;
// Enabled retries.
func (p *Provider) Set(cfg Config) {
	p.mu.Lock()
	p.cfg, p.oauth, p.verifier, p.lastTry = cfg, nil, nil, time.Time{}
	p.mu.Unlock()
	p.Enabled()
}

// Reload checks the configuration via discovery and applies it only on success;
// otherwise the previous one stays active.
func (p *Provider) Reload(cfg Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	oauthCfg, verifier, err := discover(ctx, cfg)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.cfg, p.oauth, p.verifier, p.lastTry = cfg, oauthCfg, verifier, time.Now()
	p.mu.Unlock()
	log.Printf("oidc: login via %s enabled", cfg.Issuer)
	return nil
}

func (p *Provider) Clear() {
	p.mu.Lock()
	p.cfg, p.oauth, p.verifier = Config{}, nil, nil
	p.mu.Unlock()
}

// Test runs the discovery only; the provider stays unchanged.
func (p *Provider) Test(ctx context.Context, cfg Config) error {
	_, _, err := discover(ctx, cfg)
	return err
}

func discover(ctx context.Context, cfg Config) (*oauth2.Config, *gooidc.IDTokenVerifier, error) {
	prov, err := gooidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, nil, err
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaultScopes()
	}
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     prov.Endpoint(),
		Scopes:       scopes,
	}, prov.Verifier(&gooidc.Config{ClientID: cfg.ClientID}), nil
}

func defaultScopes() []string { return []string{gooidc.ScopeOpenID, "profile", "email"} }

// Enabled tells whether OIDC is ready. An unreachable provider is
// retried at most every 30 seconds.
func (p *Provider) Enabled() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cfg.Issuer == "" {
		return false
	}
	if p.oauth != nil {
		return true
	}
	if !p.lastTry.IsZero() && time.Since(p.lastTry) < retryAfter {
		return false
	}
	p.lastTry = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	oauthCfg, verifier, err := discover(ctx, p.cfg)
	if err != nil {
		log.Printf("oidc: discovery of %s failed, login disabled: %v", p.cfg.Issuer, err)
		return false
	}
	p.oauth, p.verifier = oauthCfg, verifier
	log.Printf("oidc: login via %s enabled", p.cfg.Issuer)
	return true
}

// current returns the running configuration; nil if OIDC is not ready.
func (p *Provider) current() (*oauth2.Config, *gooidc.IDTokenVerifier) {
	if !p.Enabled() {
		return nil, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.oauth, p.verifier
}

func (p *Provider) Name() string {
	if p == nil {
		return ""
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cfg.Name
}

// PasswordLogin tells whether password login and registration are allowed.
// Always true without active OIDC, so nobody is locked out.
func (p *Provider) PasswordLogin() bool {
	if p == nil || !p.Enabled() {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.cfg.DisablePassword
}

func (p *Provider) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+flowPath+"/login", p.handleLogin)
	mux.HandleFunc("GET "+flowPath+"/callback", p.handleCallback)
}

type flow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Next     string `json:"x"`
	Expires  int64  `json:"e"`
}

func random() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (p *Provider) sign(payload []byte) string {
	m := hmac.New(sha256.New, p.key)
	m.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (p *Provider) unsign(value string) (flow, bool) {
	var f flow
	body, sig, ok := strings.Cut(value, ".")
	if !ok {
		return f, false
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(body)
	want, err2 := base64.RawURLEncoding.DecodeString(sig)
	if err1 != nil || err2 != nil {
		return f, false
	}
	m := hmac.New(sha256.New, p.key)
	m.Write(payload)
	if !hmac.Equal(m.Sum(nil), want) || json.Unmarshal(payload, &f) != nil {
		return f, false
	}
	return f, time.Now().Unix() < f.Expires
}

// flowCookieHeader is SameSite=Lax so the cookie comes along with the redirect
// back from the provider (top-level navigation).
func flowCookieHeader(value string, maxAge int) string {
	return (&http.Cookie{
		Name:     flowCookie,
		Value:    value,
		Path:     flowPath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

// safeNext only allows relative paths (no open redirect).
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, "\\") {
		return "/"
	}
	return next
}

func (p *Provider) handleLogin(w http.ResponseWriter, r *http.Request) {
	oauthCfg, _ := p.current()
	if oauthCfg == nil {
		http.NotFound(w, r)
		return
	}
	f := flow{
		State: random(), Nonce: random(), Verifier: oauth2.GenerateVerifier(),
		Next: safeNext(r.URL.Query().Get("next")), Expires: time.Now().Add(flowTTL).Unix(),
	}
	payload, _ := json.Marshal(f)
	w.Header().Add("Set-Cookie", flowCookieHeader(p.sign(payload), int(flowTTL.Seconds())))
	target := oauthCfg.AuthCodeURL(f.State, gooidc.Nonce(f.Nonce), oauth2.S256ChallengeOption(f.Verifier))
	http.Redirect(w, r, target, http.StatusFound)
}

func fail(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?error="+url.QueryEscape(code), http.StatusFound)
}

func (p *Provider) handleCallback(w http.ResponseWriter, r *http.Request) {
	oauthCfg, verifier := p.current()
	if oauthCfg == nil {
		http.NotFound(w, r)
		return
	}
	// The flow cookie is valid only once
	w.Header().Add("Set-Cookie", flowCookieHeader("", -1))
	c, err := r.Cookie(flowCookie)
	if err != nil {
		fail(w, r, ErrFailed)
		return
	}
	f, ok := p.unsign(c.Value)
	q := r.URL.Query()
	if !ok || subtle.ConstantTimeCompare([]byte(f.State), []byte(q.Get("state"))) != 1 {
		fail(w, r, ErrFailed)
		return
	}
	if e := q.Get("error"); e != "" || q.Get("code") == "" {
		log.Printf("oidc: provider returned error %q: %s", e, q.Get("error_description"))
		fail(w, r, ErrFailed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	token, err := oauthCfg.Exchange(ctx, q.Get("code"), oauth2.VerifierOption(f.Verifier))
	if err != nil {
		log.Printf("oidc: code exchange: %v", err)
		fail(w, r, ErrFailed)
		return
	}
	raw, _ := token.Extra("id_token").(string)
	idToken, err := verifier.Verify(ctx, raw)
	if err != nil {
		log.Printf("oidc: id token: %v", err)
		fail(w, r, ErrFailed)
		return
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(f.Nonce)) != 1 {
		log.Printf("oidc: nonce mismatch")
		fail(w, r, ErrFailed)
		return
	}
	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Locale            string `json:"locale"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Printf("oidc: claims: %v", err)
		fail(w, r, ErrFailed)
		return
	}

	_, session, err := p.login.LoginOIDC(service.OIDCClaims{
		Issuer: idToken.Issuer, Subject: idToken.Subject,
		Email: claims.Email, EmailVerified: claims.EmailVerified,
		PreferredUsername: claims.PreferredUsername, Name: claims.Name,
		GivenName: claims.GivenName, FamilyName: claims.FamilyName, Locale: claims.Locale,
	})
	switch {
	case errors.Is(err, service.ErrForbidden):
		fail(w, r, ErrRegistrationClosed)
		return
	case errors.Is(err, service.ErrEmailUnverified):
		fail(w, r, ErrEmailUnverified)
		return
	case err != nil:
		log.Printf("oidc: login: %v", err)
		fail(w, r, ErrFailed)
		return
	}
	w.Header().Add("Set-Cookie", auth.SessionCookie(session, service.SessionTTL))
	http.Redirect(w, r, f.Next, http.StatusFound)
}
