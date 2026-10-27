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

const locationColumns = `id, trip_id, position, country_id, name, lat, lng, COALESCE(visited_at, ''), notes, visibility`

func scanLocation(row scanner) (domain.Location, error) {
	var l domain.Location
	err := row.Scan(&l.ID, &l.TripID, &l.Position, &l.CountryID, &l.Name, &l.Lat, &l.Lng, &l.VisitedAt, &l.Notes, &l.Visibility)
	return l, err
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
