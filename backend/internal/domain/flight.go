package domain

import "time"

type Flight struct {
	ID         string
	From       string // IATA code
	To         string // IATA code
	Visibility Visibility
	CreatedAt  time.Time
}
