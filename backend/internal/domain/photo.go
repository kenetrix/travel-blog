package domain

import "time"

// Photo is a memory tied to a location. It belongs to a user via the
// location and inherits its visibility.
type Photo struct {
	ID          string
	LocationID  string
	StorageKey  string // key of the file in port.FileStorage
	ContentType string
	Caption     string
	TakenAt     string
	CreatedAt   time.Time
}
