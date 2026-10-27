package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

type OIDCClaims struct {
	Issuer            string
	Subject           string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Name              string
	GivenName         string
	FamilyName        string
	Locale            string
	// UILanguage is the login page's language, used when Locale is not supported
	UILanguage string
}

// LoginOIDC logs in the user of an OIDC identity and returns user and session token. Order:
// known identity, existing account with the same verified email, new account (if registration is open).
func (s *AccountService) LoginOIDC(c OIDCClaims) (domain.UserID, string, error) {
	if c.Issuer == "" || c.Subject == "" {
		return "", "", newError(ErrInvalid, "invalid_identity", "identity without issuer or subject")
	}
	u, err := s.resolveOIDC(c)
	if err != nil {
		return "", "", err
	}
	token, err := s.startSession(u)
	return u, token, err
}

func (s *AccountService) resolveOIDC(c OIDCClaims) (domain.UserID, error) {
	known, err := s.users.GetByIdentity(c.Issuer, c.Subject)
	if err != nil || known != nil {
		if known != nil {
			return known.ID, nil
		}
		return "", err
	}

	email := ""
	if n, err := normalizeEmail(c.Email); err == nil {
		email = n
	}
	if email != "" {
		existing, err := s.users.GetByEmail(email)
		if err != nil {
			return "", err
		}
		if existing != nil {
			// Without a verified email anyone could take over someone else's account
			if !c.EmailVerified {
				return "", fmt.Errorf("%w: %w by the provider", ErrConflict, ErrEmailUnverified)
			}
			if err := s.users.LinkIdentity(c.Issuer, c.Subject, existing.ID); err != nil {
				return s.afterIdentityRace(c, err)
			}
			return existing.ID, nil
		}
	}

	open, err := s.settings.RegistrationOpen()
	if err != nil {
		return "", err
	}
	if !open {
		return "", newError(ErrForbidden, "registration_closed", "registration is closed")
	}

	u := domain.User{ID: domain.UserID(uuid.NewString())}
	p := &u.Profile
	// An unverified address is not stored, so nobody can claim
	// other people's addresses
	if c.EmailVerified {
		p.Email = email
	}
	p.FirstName, _ = trimText("firstName", c.GivenName, 100)
	p.LastName, _ = trimText("lastName", c.FamilyName, 100)
	p.Language = oidcLanguage(c.Locale, c.UILanguage)
	p.Timezone = "UTC"
	p.CreatedAt = s.now().UTC()
	fullName, _ := trimText("displayName", c.Name, 100)

	base := usernameBase(c.PreferredUsername, email)
	for i := 0; i < 20; i++ {
		name, ok, err := s.freeUsername(base, i)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		p.Username = name
		p.DisplayName = defaultDisplayName(*p)
		if p.FirstName == "" && fullName != "" {
			p.DisplayName = fullName
		}
		err = s.users.CreateWithIdentity(u, c.Issuer, c.Subject)
		if err == nil {
			return u.ID, nil
		}
		if !errors.Is(err, port.ErrConflict) {
			return "", err
		}
		// Parallel login of the same identity or a username collision
		if known, gerr := s.users.GetByIdentity(c.Issuer, c.Subject); gerr != nil || known != nil {
			if known != nil {
				return known.ID, nil
			}
			return "", gerr
		}
	}
	return "", newError(ErrConflict, "no_free_username", "no free username")
}

// afterIdentityRace handles an identity that was linked in parallel:
// the existing link wins.
func (s *AccountService) afterIdentityRace(c OIDCClaims, linkErr error) (domain.UserID, error) {
	if !errors.Is(linkErr, port.ErrConflict) {
		return "", linkErr
	}
	known, err := s.users.GetByIdentity(c.Issuer, c.Subject)
	if err != nil || known == nil {
		return "", newError(ErrConflict, "identity_linked", "identity already linked")
	}
	return known.ID, nil
}

// freeUsername returns the i-th candidate (0: base, then base_2, ...);
// ok is false if it is invalid, blocked or taken.
func (s *AccountService) freeUsername(base string, i int) (string, bool, error) {
	candidate := base
	if i > 0 {
		suffix := "_" + strconv.Itoa(i+1)
		// From the tenth attempt on, a suffix no longer helps with a blocked name
		if i >= 10 {
			candidate = "user"
			suffix = "_" + uuid.NewString()[:6]
		}
		if len(candidate)+len(suffix) > 30 {
			candidate = candidate[:30-len(suffix)]
		}
		candidate += suffix
	}
	name, err := normalizeUsername(candidate)
	if err != nil || checkUsernameAllowed(name) != nil {
		return "", false, nil
	}
	existing, err := s.users.GetByUsername(name)
	return name, existing == nil, err
}

var invalidUsernameChars = regexp.MustCompile(`[^a-z0-9_]+`)

func usernameBase(preferred, email string) string {
	for _, src := range []string{preferred, strings.SplitN(email, "@", 2)[0]} {
		b := invalidUsernameChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(src)), "_")
		b = strings.Trim(b, "_")
		if len(b) >= 3 {
			if len(b) > 30 {
				b = b[:30]
			}
			return b
		}
	}
	return "user"
}

func oidcLanguage(locale, uiLanguage string) string {
	for _, tag := range []string{locale, uiLanguage} {
		if l := supportedLanguage(tag); l != "" {
			return l
		}
	}
	return "en"
}
