package service

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strconv"
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

// Error carries a stable code, and the values its message needs, so clients can
// show a translation; errors.Is still matches Kind.
type Error struct {
	Kind   error
	Code   string
	Params map[string]string
	Msg    string
}

func (e *Error) Error() string { return e.Kind.Error() + ": " + e.Msg }
func (e *Error) Unwrap() error { return e.Kind }

// newError takes params as key/value pairs.
func newError(kind error, code, msg string, params ...string) *Error {
	e := &Error{Kind: kind, Code: code, Msg: msg}
	if len(params) > 0 {
		e.Params = make(map[string]string, len(params)/2)
		for i := 0; i+1 < len(params); i += 2 {
			e.Params[params[i]] = params[i+1]
		}
	}
	return e
}

// Country (alpha-3) and airport (IATA) codes share a format. Master data only
// exists in the frontend; here only the format is checked.
var codePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func checkCode(kind, code string) error {
	if !codePattern.MatchString(code) {
		return newError(ErrInvalid, "invalid_code", fmt.Sprintf("%s %q must be three uppercase letters", kind, code), "kind", kind, "value", code)
	}
	return nil
}

func checkOptionalCode(kind, code string) error {
	if code == "" {
		return nil
	}
	return checkCode(kind, code)
}

func checkVisibility(v domain.Visibility) error {
	if !v.Valid() {
		return newError(ErrInvalid, "invalid_visibility", fmt.Sprintf("unknown visibility %q", v), "value", string(v))
	}
	return nil
}

// checkOptionalVisibility accepts nil, which means the default applies.
func checkOptionalVisibility(v *domain.Visibility) error {
	if v == nil {
		return nil
	}
	return checkVisibility(*v)
}

// checkRating accepts 0 (no rating) to 5 stars.
func checkRating(n int) error {
	if n < 0 || n > 5 {
		return newError(ErrInvalid, "invalid_rating", "rating must be between 0 and 5")
	}
	return nil
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// checkColor accepts an empty string (no colour) or a hex colour like #3a7d5c.
func checkColor(s string) error {
	if s != "" && !colorPattern.MatchString(s) {
		return newError(ErrInvalid, "invalid_color", "color must look like #3a7d5c")
	}
	return nil
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,30}$`)

// Languages are the UI languages of the frontend.
var Languages = []string{"en", "de", "es", "ru"}

func normalizeUsername(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !usernamePattern.MatchString(s) {
		return "", newError(ErrInvalid, "invalid_username", "username must be 3-30 characters a-z, 0-9 or _")
	}
	return s, nil
}

func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 254 {
		return "", newError(ErrInvalid, "invalid_email", "invalid email address")
	}
	return s, nil
}

// bcrypt only considers the first 72 bytes.
func checkPassword(p string) error {
	if utf8.RuneCountInString(p) < 10 || len(p) > 72 {
		return newError(ErrInvalid, "invalid_password", "password must have at least 10 characters and at most 72 bytes")
	}
	return nil
}

func checkLanguage(s string) error {
	if !slices.Contains(Languages, s) {
		return newError(ErrInvalid, "invalid_language", fmt.Sprintf("language %q must be one of en, de, es, ru", s), "value", s)
	}
	return nil
}

// supportedLanguage reduces a BCP 47 tag like "de-AT" or "es_MX" to a supported
// language, or returns "".
func supportedLanguage(tag string) string {
	primary, _, _ := strings.Cut(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"), "-")
	primary = strings.ToLower(primary)
	if slices.Contains(Languages, primary) {
		return primary
	}
	return ""
}

func checkTimezone(s string) error {
	if s == "" || s == "Local" {
		return newError(ErrInvalid, "invalid_timezone", "timezone must be an IANA name like \"Europe/Berlin\"", "value", s)
	}
	if _, err := time.LoadLocation(s); err != nil {
		return newError(ErrInvalid, "invalid_timezone", fmt.Sprintf("unknown timezone %q", s), "value", s)
	}
	return nil
}

func trimText(field, s string, max int) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > max {
		return "", newError(ErrInvalid, "text_too_long", fmt.Sprintf("%s must have at most %d characters", field, max), "field", field, "max", strconv.Itoa(max))
	}
	return s, nil
}

const (
	// The longest scheduled flights take under 20 hours
	maxFlightMinutes  = 24 * 60
	maxAircraftLength = 60
)

// checkFlightUpdate validates u and normalises its flight number and aircraft.
func checkFlightUpdate(u *domain.FlightUpdate) error {
	if err := checkOptionalVisibility(u.Visibility); err != nil {
		return err
	}
	if u.Number != nil && *u.Number != "" {
		number, ok := domain.NormalizeFlightNumber(*u.Number)
		if !ok {
			return newError(ErrInvalid, "invalid_flight_number", fmt.Sprintf("flight number %q must be an airline code and up to four digits, e.g. LH 400", *u.Number), "value", *u.Number)
		}
		u.Number = &number
	}
	if u.Date != nil {
		if err := checkDate(*u.Date); err != nil {
			return err
		}
	}
	if u.DurationMinutes != nil && (*u.DurationMinutes < 0 || *u.DurationMinutes > maxFlightMinutes) {
		return newError(ErrInvalid, "invalid_duration", fmt.Sprintf("duration must be 1 to %d minutes", maxFlightMinutes), "max", strconv.Itoa(maxFlightMinutes))
	}
	if u.Aircraft != nil {
		aircraft, err := trimText("aircraft", *u.Aircraft, maxAircraftLength)
		if err != nil {
			return err
		}
		u.Aircraft = &aircraft
	}
	return nil
}

// checkDate accepts an empty string (no date) or a day between 1900 and 2100.
func checkDate(s string) error {
	if s == "" {
		return nil
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil || d.Year() < 1900 || d.Year() > 2100 {
		return newError(ErrInvalid, "invalid_date", fmt.Sprintf("date %q must be a day between 1900 and 2100 (YYYY-MM-DD)", s), "value", s)
	}
	return nil
}
