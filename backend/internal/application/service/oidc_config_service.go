package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

type OIDCRuntime interface {
	// Reload checks the configuration via discovery and applies it only
	// on success.
	Reload(cfg domain.OIDCConfig) error
	Clear()
	Test(ctx context.Context, cfg domain.OIDCConfig) error
	Enabled() bool
}

const (
	OIDCSourceNone     = "none"
	OIDCSourceEnv      = "env"
	OIDCSourceSettings = "settings"
)

// OIDCConfigService manages the OIDC configuration in the admin area. If the OIDC_*
// variables are complete (env), they alone apply and the saved configuration is read-only.
type OIDCConfigService struct {
	accounts *AccountService
	repo     port.OIDCSettingsRepository
	runtime  OIDCRuntime
	env      *domain.OIDCConfig
}

// NewOIDCConfigService: env is the configuration from the environment
// variables, or nil if they are incomplete.
func NewOIDCConfigService(accounts *AccountService, repo port.OIDCSettingsRepository, runtime OIDCRuntime, env *domain.OIDCConfig) *OIDCConfigService {
	return &OIDCConfigService{accounts: accounts, repo: repo, runtime: runtime, env: env}
}

type OIDCConfigView struct {
	Source          string
	Enabled         bool
	Issuer          string
	ClientID        string
	SecretSet       bool
	RedirectURL     string
	Name            string
	Scopes          string // space-separated
	DisablePassword bool
}

// OIDCConfigInput is the form data. An empty secret keeps the saved one.
type OIDCConfigInput struct {
	Issuer          string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	Name            string
	Scopes          string
	DisablePassword bool
}

const (
	keyIssuer          = "issuer"
	keyClientID        = "client_id"
	keyClientSecret    = "client_secret"
	keyRedirectURL     = "redirect_url"
	keyName            = "name"
	keyScopes          = "scopes"
	keyDisablePassword = "disable_password"
)

// Effective returns the effective configuration; pass it to the provider at
// startup. ok is false without a complete configuration.
func (s *OIDCConfigService) Effective() (cfg domain.OIDCConfig, ok bool, err error) {
	if s.env != nil {
		return *s.env, true, nil
	}
	cfg, set, err := s.stored(true)
	if err != nil || !set {
		return domain.OIDCConfig{}, false, err
	}
	return cfg, cfg.Complete(), nil
}

// decrypted secret.
func (s *OIDCConfigService) stored(withSecret bool) (cfg domain.OIDCConfig, set bool, err error) {
	v, err := s.repo.All()
	if err != nil || len(v) == 0 {
		return cfg, false, err
	}
	cfg = domain.OIDCConfig{
		Issuer: v[keyIssuer], ClientID: v[keyClientID], RedirectURL: v[keyRedirectURL],
		Name: v[keyName], Scopes: strings.Fields(v[keyScopes]),
		DisablePassword: v[keyDisablePassword] == "true",
	}
	if secret := v[keyClientSecret]; secret != "" {
		cfg.ClientSecret = "set"
		if withSecret {
			cfg.ClientSecret = secret
		}
	}
	return cfg, true, nil
}

// View returns the configuration for the form (administrators only).
// The secret is never returned, only whether it is set.
func (s *OIDCConfigService) View(actor domain.UserID) (OIDCConfigView, error) {
	if err := s.accounts.requireAdmin(actor); err != nil {
		return OIDCConfigView{}, err
	}
	view := OIDCConfigView{Source: OIDCSourceNone, Enabled: s.runtime.Enabled()}
	var cfg domain.OIDCConfig
	if s.env != nil {
		view.Source, cfg = OIDCSourceEnv, *s.env
	} else {
		stored, set, err := s.stored(false)
		if err != nil {
			return view, err
		}
		if set {
			view.Source, cfg = OIDCSourceSettings, stored
		}
	}
	view.SecretSet = cfg.ClientSecret != ""
	view.Issuer, view.ClientID, view.RedirectURL = cfg.Issuer, cfg.ClientID, cfg.RedirectURL
	view.Name, view.Scopes, view.DisablePassword = cfg.Name, strings.Join(cfg.Scopes, " "), cfg.DisablePassword
	return view, nil
}

// build validates the input and fills a missing secret from the
// saved configuration.
func (s *OIDCConfigService) build(in OIDCConfigInput) (domain.OIDCConfig, error) {
	cfg := domain.OIDCConfig{
		Issuer:          strings.TrimRight(strings.TrimSpace(in.Issuer), "/"),
		ClientID:        strings.TrimSpace(in.ClientID),
		ClientSecret:    in.ClientSecret,
		RedirectURL:     strings.TrimSpace(in.RedirectURL),
		Name:            strings.TrimSpace(in.Name),
		Scopes:          strings.Fields(in.Scopes),
		DisablePassword: in.DisablePassword,
	}
	for _, f := range []struct{ field, raw string }{{"issuer", cfg.Issuer}, {"redirectUrl", cfg.RedirectURL}} {
		if u, err := url.Parse(f.raw); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return cfg, fmt.Errorf("%w: %s must be an http(s) URL", ErrInvalid, f.field)
		}
	}
	if cfg.ClientID == "" {
		return cfg, fmt.Errorf("%w: clientId is required", ErrInvalid)
	}
	if len([]rune(cfg.Name)) > 40 {
		return cfg, fmt.Errorf("%w: name must have at most 40 characters", ErrInvalid)
	}
	if cfg.Name == "" {
		cfg.Name = "SSO"
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	if cfg.ClientSecret == "" && s.env == nil {
		old, _, err := s.stored(true)
		if err != nil {
			return cfg, err
		}
		cfg.ClientSecret = old.ClientSecret
	}
	if cfg.ClientSecret == "" {
		return cfg, fmt.Errorf("%w: clientSecret is required", ErrInvalid)
	}
	return cfg, nil
}

func (s *OIDCConfigService) writable(actor domain.UserID) error {
	if err := s.accounts.requireAdmin(actor); err != nil {
		return err
	}
	if s.env != nil {
		return fmt.Errorf("%w: OIDC is configured by environment variables", ErrConflict)
	}
	return nil
}

// Save stores the configuration and applies it without restart. Discovery must succeed first;
// this also covers "turn off password login" and prevents locking yourself out.
func (s *OIDCConfigService) Save(actor domain.UserID, in OIDCConfigInput) error {
	if err := s.writable(actor); err != nil {
		return err
	}
	cfg, err := s.build(in)
	if err != nil {
		return err
	}
	if err := s.runtime.Reload(cfg); err != nil {
		return fmt.Errorf("%w: the provider cannot be reached: %v", ErrInvalid, err)
	}
	return s.repo.Replace(map[string]string{
		keyIssuer: cfg.Issuer, keyClientID: cfg.ClientID, keyClientSecret: cfg.ClientSecret,
		keyRedirectURL: cfg.RedirectURL, keyName: cfg.Name, keyScopes: strings.Join(cfg.Scopes, " "),
		keyDisablePassword: fmt.Sprint(cfg.DisablePassword),
	})
}

func (s *OIDCConfigService) Delete(actor domain.UserID) error {
	if err := s.writable(actor); err != nil {
		return err
	}
	if err := s.repo.Clear(); err != nil {
		return err
	}
	s.runtime.Clear()
	return nil
}

func (s *OIDCConfigService) Test(actor domain.UserID, in OIDCConfigInput) error {
	if err := s.accounts.requireAdmin(actor); err != nil {
		return err
	}
	cfg, err := s.build(in)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.runtime.Test(ctx, cfg); err != nil {
		return fmt.Errorf("%w: the provider cannot be reached: %v", ErrInvalid, err)
	}
	return nil
}
