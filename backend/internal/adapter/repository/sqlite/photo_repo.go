package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"travel-blog/backend/internal/domain"
)

// PhotoRepo implements port.PhotoRepository for SQLite. Photos have no user of
// their own; every query checks it via the location.
type PhotoRepo struct {
	db *sql.DB
}

func NewPhotoRepo(db *sql.DB) *PhotoRepo {
	return &PhotoRepo{db: db}
}

const photoColumns = `p.id, p.location_id, p.storage_key, p.content_type, p.caption,
	COALESCE(p.taken_at, ''), p.created_at`

const ownedBy = `FROM photos p JOIN locations l ON l.id = p.location_id WHERE l.user_id = ?`

func scanPhoto(row scanner) (domain.Photo, error) {
	var p domain.Photo
	var created string
	err := row.Scan(&p.ID, &p.LocationID, &p.StorageKey, &p.ContentType, &p.Caption, &p.TakenAt, &created)
	if err != nil {
		return p, err
	}
	p.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return p, err
}

func (r *PhotoRepo) ListByLocation(user domain.UserID, locationID string) ([]domain.Photo, error) {
	rows, err := r.db.Query(
		`SELECT `+photoColumns+` `+ownedBy+` AND p.location_id = ? ORDER BY p.created_at`,
		user, locationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Photo{}
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PhotoRepo) Get(user domain.UserID, id string) (*domain.Photo, error) {
	p, err := scanPhoto(r.db.QueryRow(`SELECT `+photoColumns+` `+ownedBy+` AND p.id = ?`, user, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PhotoRepo) Create(p domain.Photo) error {
	_, err := r.db.Exec(`
		INSERT INTO photos (id, location_id, storage_key, content_type, caption, taken_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.LocationID, p.StorageKey, p.ContentType, p.Caption, nullIfEmpty(p.TakenAt),
		p.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *PhotoRepo) Delete(user domain.UserID, id string) (bool, error) {
	return affected(r.db.Exec(`
		DELETE FROM photos
		WHERE id = ? AND location_id IN (SELECT id FROM locations WHERE user_id = ?)
	`, id, user))
}
