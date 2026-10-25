package sqlite

import (
	"database/sql"
	"errors"

	"travel-blog/backend/internal/domain"
)

// Every query is limited to one user.
type LocationRepo struct {
	db *sql.DB
}

func NewLocationRepo(db *sql.DB) *LocationRepo {
	return &LocationRepo{db: db}
}

const locationColumns = `id, country_id, name, lat, lng, COALESCE(visited_at, ''), notes, visibility`

func scanLocation(row scanner) (domain.Location, error) {
	var l domain.Location
	err := row.Scan(&l.ID, &l.CountryID, &l.Name, &l.Lat, &l.Lng, &l.VisitedAt, &l.Notes, &l.Visibility)
	return l, err
}

func (r *LocationRepo) ListByCountry(user domain.UserID, countryID string) ([]domain.Location, error) {
	rows, err := r.db.Query(
		`SELECT `+locationColumns+` FROM locations WHERE user_id = ? AND country_id = ? ORDER BY name`,
		user, countryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LocationRepo) Get(user domain.UserID, id string) (*domain.Location, error) {
	l, err := scanLocation(r.db.QueryRow(
		`SELECT `+locationColumns+` FROM locations WHERE user_id = ? AND id = ?`, user, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LocationRepo) Create(user domain.UserID, l domain.Location) error {
	_, err := r.db.Exec(`
		INSERT INTO locations (id, user_id, country_id, name, lat, lng, visited_at, notes, visibility)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.ID, user, l.CountryID, l.Name, l.Lat, l.Lng, nullIfEmpty(l.VisitedAt), l.Notes, l.Visibility)
	return err
}

func (r *LocationRepo) Update(user domain.UserID, l domain.Location) (bool, error) {
	return affected(r.db.Exec(`
		UPDATE locations SET country_id = ?, name = ?, lat = ?, lng = ?, visited_at = ?,
			notes = ?, visibility = ?
		WHERE user_id = ? AND id = ?
	`, l.CountryID, l.Name, l.Lat, l.Lng, nullIfEmpty(l.VisitedAt), l.Notes, l.Visibility, user, l.ID))
}

func (r *LocationRepo) Delete(user domain.UserID, id string) (bool, error) {
	return affected(r.db.Exec(`DELETE FROM locations WHERE user_id = ? AND id = ?`, user, id))
}
