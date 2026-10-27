package service

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

const MaxPhotoBytes = 20 << 20

var photoTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
	"image/heic": "heic",
}

// PhotoService stores photos for locations. Every access checks via the location that the
// photo belongs to the user; files live in the user's area of the storage.
type PhotoService struct {
	locations port.LocationRepository
	photos    port.PhotoRepository
	files     port.FileStorage
	now       func() time.Time
}

func NewPhotoService(locations port.LocationRepository, photos port.PhotoRepository, files port.FileStorage) *PhotoService {
	return &PhotoService{locations: locations, photos: photos, files: files, now: time.Now}
}

func (s *PhotoService) Upload(user domain.UserID, locationID, contentType string, content io.Reader, caption, takenAt string) (domain.Photo, error) {
	ext, ok := photoTypes[contentType]
	if !ok {
		return domain.Photo{}, newError(ErrInvalid, "unsupported_image", fmt.Sprintf("unsupported image type %q", contentType), "value", contentType)
	}
	caption, err := trimText("caption", caption, 500)
	if err != nil {
		return domain.Photo{}, err
	}
	loc, err := s.locations.Get(user, locationID)
	if err != nil {
		return domain.Photo{}, err
	}
	if loc == nil {
		return domain.Photo{}, newError(ErrNotFound, "location_not_found", fmt.Sprintf("location %s", locationID))
	}

	limited := &countingReader{r: io.LimitReader(content, MaxPhotoBytes+1)}
	key, err := s.files.Save(user, ext, limited)
	if err != nil {
		return domain.Photo{}, err
	}
	if limited.n > MaxPhotoBytes {
		s.removeFile(key)
		return domain.Photo{}, newError(ErrInvalid, "photo_too_large", fmt.Sprintf("photo is larger than %d MB", MaxPhotoBytes>>20), "max", strconv.Itoa(MaxPhotoBytes>>20))
	}

	p := domain.Photo{
		ID:          uuid.NewString(),
		LocationID:  loc.ID,
		StorageKey:  key,
		ContentType: contentType,
		Caption:     caption,
		TakenAt:     strings.TrimSpace(takenAt),
		CreatedAt:   s.now().UTC(),
	}
	if err := s.photos.Create(p); err != nil {
		s.removeFile(key)
		return domain.Photo{}, err
	}
	return p, nil
}

// Open returns a photo of the user with its file; the caller closes it.
func (s *PhotoService) Open(user domain.UserID, id string) (domain.Photo, io.ReadCloser, error) {
	p, err := s.photos.Get(user, id)
	if err != nil {
		return domain.Photo{}, nil, err
	}
	if p == nil {
		return domain.Photo{}, nil, newError(ErrNotFound, "photo_not_found", fmt.Sprintf("photo %s", id))
	}
	f, err := s.files.Open(p.StorageKey)
	if err != nil {
		return domain.Photo{}, nil, err
	}
	return *p, f, nil
}

func (s *PhotoService) Delete(user domain.UserID, id string) error {
	p, err := s.photos.Get(user, id)
	if err != nil {
		return err
	}
	if p == nil {
		return newError(ErrNotFound, "photo_not_found", fmt.Sprintf("photo %s", id))
	}
	if _, err := s.photos.Delete(user, id); err != nil {
		return err
	}
	s.removeFile(p.StorageKey)
	return nil
}

// removeFile deletes a file; a failure only leaves an orphaned file.
func (s *PhotoService) removeFile(key string) {
	if err := s.files.Delete(key); err != nil {
		log.Printf("delete photo file %s: %v", key, err)
	}
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
