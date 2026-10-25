package port

import (
	"io"

	"travel-blog/backend/internal/domain"
)

// FileStorage abstracts storing photo files (local, S3, ...). Files are kept per user under
// keys "<user>/<name>"; callers decide from the database who may access, the storage only keeps keys inside the user's area.
type FileStorage interface {
	Save(user domain.UserID, ext string, content io.Reader) (key string, err error)
	Open(key string) (io.ReadCloser, error)
	Delete(key string) error
	DeleteAll(user domain.UserID) error
}
