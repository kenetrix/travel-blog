package domain

import (
	"regexp"
	"strings"
	"time"
)

// The optional details are entered by the user; zero values mean unknown.
type Flight struct {
	ID              string
	From            string // IATA code
	To              string // IATA code
	Visibility      Visibility
	CreatedAt       time.Time
	Number          string // canonical, e.g. "LH400"
	Date            string // YYYY-MM-DD
	DurationMinutes int
	Aircraft        string
}

// FlightUpdate changes only the non-nil fields; zero values clear them.
type FlightUpdate struct {
	Visibility      *Visibility
	Number          *string
	Date            *string
	DurationMinutes *int
	Aircraft        *string
}

func (u FlightUpdate) Empty() bool {
	return u.Visibility == nil && u.Number == nil && u.Date == nil && u.DurationMinutes == nil && u.Aircraft == nil
}

// A three-letter (ICAO) designator is only tried when the third character is a
// letter, so "LH400" is LH + 400 and "DLH400" is DLH + 400.
var (
	icaoFlightNumber = regexp.MustCompile(`^([A-Z]{3})([0-9]{1,4})([A-Z]?)$`)
	iataFlightNumber = regexp.MustCompile(`^([A-Z0-9]{2})([0-9]{1,4})([A-Z]?)$`)
	hasLetter        = regexp.MustCompile(`[A-Z]`)
)

// NormalizeFlightNumber returns the canonical form of a flight number, e.g.
// "lh 0400" → "LH400", and false if it is not a valid flight number.
func NormalizeFlightNumber(s string) (string, bool) {
	s = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(s)))
	m := icaoFlightNumber.FindStringSubmatch(s)
	if m == nil {
		m = iataFlightNumber.FindStringSubmatch(s)
		if m == nil || !hasLetter.MatchString(m[1]) {
			return "", false
		}
	}
	number := strings.TrimLeft(m[2], "0")
	if number == "" {
		return "", false
	}
	return m[1] + number + m[3], true
}
