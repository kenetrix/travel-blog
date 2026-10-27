package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

type LocationService struct {
	repo   port.LocationRepository
	photos port.PhotoRepository
	files  port.FileStorage
	users  port.UserRepository
}

func NewLocationService(repo port.LocationRepository, photos port.PhotoRepository, files port.FileStorage, users port.UserRepository) *LocationService {
	return &LocationService{repo: repo, photos: photos, files: files, users: users}
}

func (s *LocationService) ListForCountry(user domain.UserID, countryID string) ([]domain.Location, error) {
	return s.repo.ListByCountry(user, countryID)
}

// Add creates a location; without visibility the default visibility applies.
func (s *LocationService) Add(user domain.UserID, l domain.Location) (domain.Location, error) {
	if err := checkCode("country", l.CountryID); err != nil {
		return domain.Location{}, err
	}
	l.Name = strings.TrimSpace(l.Name)
	if l.Name == "" {
		return domain.Location{}, newError(ErrInvalid, "name_required", "name is required")
	}
	if l.Visibility == "" {
		settings, err := s.users.Settings(user)
		if err != nil {
			return domain.Location{}, err
		}
		l.Visibility = settings.DefaultVisibility
	}
	if err := checkVisibility(l.Visibility); err != nil {
		return domain.Location{}, err
	}
	l.ID = uuid.NewString()
	return l, s.repo.Create(user, l)
}

// Delete removes a location with its photos - records and files.
func (s *LocationService) Delete(user domain.UserID, id string) error {
	photos, err := s.photos.ListByLocation(user, id)
	if err != nil {
		return err
	}
	found, err := s.repo.Delete(user, id)
	if err != nil {
		return err
	}
	if !found {
		return newError(ErrNotFound, "location_not_found", fmt.Sprintf("location %s", id))
	}
	for _, p := range photos {
		if err := s.files.Delete(p.StorageKey); err != nil {
			log.Printf("delete photo file %s: %v", p.StorageKey, err)
		}
	}
	return nil
}
