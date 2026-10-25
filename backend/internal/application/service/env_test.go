package service_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"travel-blog/backend/internal/adapter/repository/sqlite"
	"travel-blog/backend/internal/adapter/storage"
	"travel-blog/backend/internal/application/service"
	"travel-blog/backend/internal/domain"
	configmigrations "travel-blog/backend/migrations/config"
	migrations "travel-blog/backend/migrations/data"
)

// local is an ordinary user that newEnv creates, for tests that need one without registering.
const local = domain.UserID("local")

// env wires all services on a fresh database and an empty photo folder.
type env struct {
	db        *sql.DB
	configDB  *sql.DB
	settings  *sqlite.SettingsRepo
	photoDir  string
	accounts  *service.AccountService
	travel    *service.TravelService
	locations *service.LocationService
	photos    *service.PhotoService
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "test.db"), migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	configDB, err := sqlite.NewDB(filepath.Join(dir, "config.db"), configmigrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { configDB.Close() })
	settings := sqlite.NewSettingsRepo(configDB)
	photoDir := filepath.Join(dir, "photos")
	files, err := storage.NewLocalStorage(photoDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { files.Close() })

	users := sqlite.NewUserRepo(db)
	locationRepo := sqlite.NewLocationRepo(db)
	photoRepo := sqlite.NewPhotoRepo(db)
	if err := users.Create(domain.User{
		ID:      local,
		Profile: domain.Profile{Username: "local", Language: "en", Timezone: "UTC", CreatedAt: time.Now()},
	}, ""); err != nil {
		t.Fatal(err)
	}
	return &env{
		db:        db,
		configDB:  configDB,
		settings:  settings,
		photoDir:  photoDir,
		accounts:  service.NewAccountService(users, sqlite.NewSessionRepo(db), files, settings),
		travel:    service.NewTravelService(sqlite.NewCountryEntryRepo(db), sqlite.NewFlightRepo(db), users),
		locations: service.NewLocationService(locationRepo, photoRepo, files, users),
		photos:    service.NewPhotoService(locationRepo, photoRepo, files),
	}
}

// register creates a user and returns ID and session token.
func (e *env) register(t *testing.T, username string) (domain.UserID, string) {
	t.Helper()
	u, token, err := e.accounts.Register(service.Registration{
		Email:    username + "@example.com",
		Username: username,
		Password: "correct horse battery",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u.ID, token
}
