package service

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata" // time zones even without a system database (Windows)
	"unicode/utf8"

	"travel-blog/backend/internal/domain"
)

var (
	ErrInvalid      = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrDuplicate    = errors.New("already exists")
	ErrSameAirport  = errors.New("same airport")
	ErrConflict     = errors.New("already taken")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	// ErrEmailUnverified accompanies ErrConflict when the provider does not report
	// the email of an existing account as verified
	ErrEmailUnverified = errors.New("email not verified")
)

// Country (alpha-3) and airport (IATA) codes share a format. Master data only
// exists in the frontend; here only the format is checked.
var codePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func checkCode(kind, code string) error {
	if !codePattern.MatchString(code) {
		return fmt.Errorf("%w: %s %q must be three uppercase letters", ErrInvalid, kind, code)
	}
	return nil
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func checkOptionalCode(kind, code string) error {
	if code == "" {
		return nil
	}
	return checkCode(kind, code)
}

func checkVisibility(v domain.Visibility) error {
	if !v.Valid() {
		return fmt.Errorf("%w: unknown visibility %q", ErrInvalid, v)
	}
	return nil
}

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)
	languagePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
)

func normalizeUsername(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !usernamePattern.MatchString(s) {
		return "", fmt.Errorf("%w: username must be 3-30 characters a-z, 0-9 or _", ErrInvalid)
	}
	return s, nil
}

func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 254 {
		return "", fmt.Errorf("%w: invalid email address", ErrInvalid)
	}
	return s, nil
}

// bcrypt only considers the first 72 bytes.
func checkPassword(p string) error {
	if utf8.RuneCountInString(p) < 10 || len(p) > 72 {
		return fmt.Errorf("%w: password must have at least 10 characters and at most 72 bytes", ErrInvalid)
	}
	return nil
}

func checkLanguage(s string) error {
	if !languagePattern.MatchString(s) {
		return fmt.Errorf("%w: language must be a BCP 47 tag like \"de\" or \"en-US\"", ErrInvalid)
	}
	return nil
}

func checkTimezone(s string) error {
	if s == "" || s == "Local" {
		return fmt.Errorf("%w: timezone must be an IANA name like \"Europe/Berlin\"", ErrInvalid)
	}
	if _, err := time.LoadLocation(s); err != nil {
		return fmt.Errorf("%w: unknown timezone %q", ErrInvalid, s)
	}
	return nil
}

func trimText(field, s string, max int) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		return "", fmt.Errorf("%w: %s must have at most %d characters", ErrInvalid, field, max)
	}
	return s, nil
}
