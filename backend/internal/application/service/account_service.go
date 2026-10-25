package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

const SessionTTL = 30 * 24 * time.Hour

type AccountService struct {
	users    port.UserRepository
	sessions port.SessionRepository
	files    port.FileStorage
	settings port.AppSettingsRepository
	now      func() time.Time
	hashCost int
	// dummyHash is compared for unknown usernames so the response time does not
	// reveal whether an account exists
	dummyHash []byte
	// initialAdminFile holds the credentials of the first administrator;
	// it disappears once they change their password
	initialAdminFile string
}

func NewAccountService(users port.UserRepository, sessions port.SessionRepository, files port.FileStorage, settings port.AppSettingsRepository) *AccountService {
	s := &AccountService{users: users, sessions: sessions, files: files, settings: settings, now: time.Now, hashCost: bcrypt.DefaultCost}
	s.dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy password"), s.hashCost)
	return s
}

type Registration struct {
	Email     string
	Username  string
	Password  string
	FirstName string
	LastName  string
	Language  string // empty: "en"
}

// Register creates an account and logs it in; the second result is the session token.
func (s *AccountService) Register(r Registration) (domain.User, string, error) {
	open, err := s.settings.RegistrationOpen()
	if err != nil {
		return domain.User{}, "", err
	}
	if !open {
		return domain.User{}, "", fmt.Errorf("%w: registration is closed", ErrForbidden)
	}
	u := domain.User{ID: domain.UserID(uuid.NewString())}
	p := &u.Profile
	// The email is optional; users log in with their username
	if strings.TrimSpace(r.Email) != "" {
		if p.Email, err = normalizeEmail(r.Email); err != nil {
			return domain.User{}, "", err
		}
	}
	if p.Username, err = normalizeUsername(r.Username); err != nil {
		return domain.User{}, "", err
	}
	if err := checkUsernameAllowed(p.Username); err != nil {
		return domain.User{}, "", err
	}
	if err := checkPassword(r.Password); err != nil {
		return domain.User{}, "", err
	}
	if p.FirstName, err = trimText("firstName", r.FirstName, 100); err != nil {
		return domain.User{}, "", err
	}
	if p.LastName, err = trimText("lastName", r.LastName, 100); err != nil {
		return domain.User{}, "", err
	}
	p.Language = r.Language
	if p.Language == "" {
		p.Language = "en"
	}
	if err := checkLanguage(p.Language); err != nil {
		return domain.User{}, "", err
	}
	p.DisplayName = defaultDisplayName(*p)
	p.Timezone = "UTC"
	p.CreatedAt = s.now().UTC()

	hash, err := bcrypt.GenerateFromPassword([]byte(r.Password), s.hashCost)
	if err != nil {
		return domain.User{}, "", err
	}
	if err := s.users.Create(u, string(hash)); err != nil {
		if errors.Is(err, port.ErrConflict) {
			return domain.User{}, "", fmt.Errorf("%w: email or username", ErrConflict)
		}
		return domain.User{}, "", err
	}
	token, err := s.startSession(u.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return u, token, nil
}

func defaultDisplayName(p domain.Profile) string {
	switch {
	case p.FirstName != "" && p.LastName != "":
		return p.FirstName + " " + p.LastName
	case p.FirstName != "":
		return p.FirstName
	default:
		return p.Username
	}
}

// UsernameTaken tells whether the username exists. The login page uses it to
// choose between login and registration.
func (s *AccountService) UsernameTaken(username string) (bool, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return false, err
	}
	u, err := s.users.GetByUsername(username)
	if err != nil || u != nil {
		return u != nil, err
	}
	// A name that is already taken stays valid so its account can log in
	return false, checkUsernameAllowed(username)
}

func (s *AccountService) Login(username, password string) (domain.UserID, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	id, hash, ok, err := s.users.Credentials(username)
	if err != nil {
		return "", "", err
	}
	if !ok {
		bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", "", fmt.Errorf("%w: wrong username or password", ErrUnauthorized)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", "", fmt.Errorf("%w: wrong username or password", ErrUnauthorized)
	}
	token, err := s.startSession(id)
	return id, token, err
}

// startSession creates a random token; only its hash is stored.
func (s *AccountService) startSession(user domain.UserID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := s.now().UTC()
	if err := s.sessions.DeleteExpired(now); err != nil {
		log.Printf("delete expired sessions: %v", err)
	}
	err := s.sessions.Create(domain.Session{
		TokenHash: hashToken(token),
		UserID:    user,
		CreatedAt: now,
		ExpiresAt: now.Add(SessionTTL),
	})
	return token, err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Authenticate returns the user of a session token; ok is false for
// unknown or expired tokens.
func (s *AccountService) Authenticate(token string) (domain.UserID, bool, error) {
	if token == "" {
		return "", false, nil
	}
	session, err := s.sessions.Get(hashToken(token))
	if err != nil || session == nil {
		return "", false, err
	}
	if !s.now().Before(session.ExpiresAt) {
		return "", false, s.sessions.Delete(session.TokenHash)
	}
	return session.UserID, true, nil
}

// Logout ends the session; an empty or unknown token is no error.
func (s *AccountService) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(hashToken(token))
}

func (s *AccountService) Account(user domain.UserID) (domain.User, domain.Settings, error) {
	u, err := s.users.Get(user)
	if err != nil {
		return domain.User{}, domain.Settings{}, err
	}
	if u == nil {
		return domain.User{}, domain.Settings{}, fmt.Errorf("%w: user %s", ErrNotFound, user)
	}
	settings, err := s.users.Settings(user)
	return *u, settings, err
}

func (s *AccountService) UpdateProfile(user domain.UserID, up domain.ProfileUpdate) (domain.Profile, error) {
	u, _, err := s.Account(user)
	if err != nil {
		return domain.Profile{}, err
	}
	p := u.Profile
	if up.Username != nil {
		if p.Username, err = normalizeUsername(*up.Username); err != nil {
			return domain.Profile{}, err
		}
		// Only on change, so an old name does not block the profile
		if p.Username != u.Profile.Username {
			if err := checkUsernameAllowed(p.Username); err != nil {
				return domain.Profile{}, err
			}
		}
	}
	if up.Email != nil {
		if user == domain.LocalUserID {
			return domain.Profile{}, fmt.Errorf("%w: the local user has no email", ErrInvalid)
		}
		p.Email = ""
		if strings.TrimSpace(*up.Email) != "" {
			if p.Email, err = normalizeEmail(*up.Email); err != nil {
				return domain.Profile{}, err
			}
		}
	}
	texts := []struct {
		field string
		value *string
		dst   *string
		max   int
	}{
		{"firstName", up.FirstName, &p.FirstName, 100},
		{"lastName", up.LastName, &p.LastName, 100},
		{"displayName", up.DisplayName, &p.DisplayName, 100},
		{"bio", up.Bio, &p.Bio, 2000},
	}
	for _, t := range texts {
		if t.value == nil {
			continue
		}
		if *t.dst, err = trimText(t.field, *t.value, t.max); err != nil {
			return domain.Profile{}, err
		}
	}
	if up.Language != nil {
		if err := checkLanguage(*up.Language); err != nil {
			return domain.Profile{}, err
		}
		p.Language = *up.Language
	}
	if up.Timezone != nil {
		if err := checkTimezone(*up.Timezone); err != nil {
			return domain.Profile{}, err
		}
		p.Timezone = *up.Timezone
	}
	if up.HomeCountry != nil {
		if err := checkOptionalCode("homeCountry", *up.HomeCountry); err != nil {
			return domain.Profile{}, err
		}
		p.HomeCountry = *up.HomeCountry
	}
	if up.HomeAirport != nil {
		if err := checkOptionalCode("homeAirport", *up.HomeAirport); err != nil {
			return domain.Profile{}, err
		}
		p.HomeAirport = *up.HomeAirport
	}
	if p.DisplayName == "" {
		p.DisplayName = defaultDisplayName(p)
	}

	if err := s.users.UpdateProfile(user, p); err != nil {
		if errors.Is(err, port.ErrConflict) {
			return domain.Profile{}, fmt.Errorf("%w: email or username", ErrConflict)
		}
		return domain.Profile{}, err
	}
	return p, nil
}

func (s *AccountService) SaveSettings(user domain.UserID, set domain.Settings) error {
	switch set.DistanceUnit {
	case domain.Kilometers, domain.Miles:
	default:
		return fmt.Errorf("%w: unknown distance unit %q", ErrInvalid, set.DistanceUnit)
	}
	switch set.Theme {
	case domain.ThemeSystem, domain.ThemeLight, domain.ThemeDark:
	default:
		return fmt.Errorf("%w: unknown theme %q", ErrInvalid, set.Theme)
	}
	switch set.DefaultView {
	case domain.ViewCountries, domain.ViewFlights:
	default:
		return fmt.Errorf("%w: unknown view %q", ErrInvalid, set.DefaultView)
	}
	if err := checkVisibility(set.DefaultVisibility); err != nil {
		return err
	}
	if err := checkVisibility(set.ProfileVisibility); err != nil {
		return err
	}
	return s.users.SaveSettings(user, set)
}

// ChangePassword checks the current password, sets the new one and ends all other
// sessions; currentToken (the own session) stays valid.
func (s *AccountService) ChangePassword(user domain.UserID, currentToken, current, next string) error {
	hash, err := s.users.PasswordHash(user)
	if err != nil {
		return err
	}
	if hash == "" {
		return fmt.Errorf("%w: this account has no password", ErrForbidden)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return fmt.Errorf("%w: wrong current password", ErrForbidden)
	}
	if err := checkPassword(next); err != nil {
		return err
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(next), s.hashCost)
	if err != nil {
		return err
	}
	if err := s.users.SetPasswordHash(user, string(newHash)); err != nil {
		return err
	}
	if err := s.users.SetMustChangePassword(user, false); err != nil {
		return err
	}
	// This makes the credentials of the first administrator obsolete
	if s.initialAdminFile != "" {
		if err := os.Remove(s.initialAdminFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("remove %s: %v", s.initialAdminFile, err)
		}
	}
	return s.sessions.DeleteOthers(user, hashToken(currentToken))
}

// MustChangePassword tells whether the user must change their password before
// using the application. Errors count as no.
func (s *AccountService) MustChangePassword(user domain.UserID) bool {
	u, err := s.users.Get(user)
	return err == nil && u != nil && u.MustChangePassword
}

// DeleteAccount removes the user with all data and photos.
// The local user always stays.
func (s *AccountService) DeleteAccount(user domain.UserID) error {
	if user == domain.LocalUserID {
		return fmt.Errorf("%w: the local user cannot be deleted", ErrConflict)
	}
	// Data first (foreign keys also remove sessions, entries, flights, locations and photo
	// records), then files: if deleting files fails, only orphaned files remain.
	if err := s.users.Delete(user); err != nil {
		return err
	}
	if err := s.files.DeleteAll(user); err != nil {
		log.Printf("delete files of user %s: %v", user, err)
	}
	return nil
}
