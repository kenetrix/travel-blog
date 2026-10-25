package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/google/uuid"

	"travel-blog/backend/internal/domain"
)

// LocalStorage keeps files on the local filesystem, one folder per user: <BaseDir>/<user>/<uuid>.<ext>.
// Implements port.FileStorage; access goes through os.Root and cannot leave the base folder.
type LocalStorage struct {
	root *os.Root
}

var (
	userPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	extPattern  = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}/[0-9a-f-]{36}\.[a-z0-9]{1,8}$`)

	ErrInvalidKey = errors.New("invalid storage key")
)

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, err
	}
	return &LocalStorage{root: root}, nil
}

func (s *LocalStorage) Close() error {
	return s.root.Close()
}

func (s *LocalStorage) Save(user domain.UserID, ext string, content io.Reader) (string, error) {
	if !userPattern.MatchString(string(user)) || !extPattern.MatchString(ext) {
		return "", ErrInvalidKey
	}
	if err := s.root.MkdirAll(string(user), 0o755); err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s/%s.%s", user, uuid.NewString(), ext)
	f, err := s.root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.root.Remove(key)
		return "", err
	}
	return key, nil
}

func (s *LocalStorage) Open(key string) (io.ReadCloser, error) {
	if !keyPattern.MatchString(key) {
		return nil, ErrInvalidKey
	}
	return s.root.Open(key)
}

func (s *LocalStorage) Delete(key string) error {
	if !keyPattern.MatchString(key) {
		return ErrInvalidKey
	}
	err := s.root.Remove(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *LocalStorage) DeleteAll(user domain.UserID) error {
	if !userPattern.MatchString(string(user)) {
		return ErrInvalidKey
	}
	return s.root.RemoveAll(string(user))
}
