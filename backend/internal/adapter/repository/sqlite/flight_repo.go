package sqlite

import (
	"database/sql"
	"time"

	"travel-blog/backend/internal/domain"
)

// Every query is limited to one user.
type FlightRepo struct {
	db *sql.DB
}

func NewFlightRepo(db *sql.DB) *FlightRepo {
	return &FlightRepo{db: db}
}

func (r *FlightRepo) List(user domain.UserID) ([]domain.Flight, error) {
	return r.list(`WHERE user_id = ?`, user)
}

func (r *FlightRepo) ListPublic(user domain.UserID) ([]domain.Flight, error) {
	return r.list(`WHERE user_id = ? AND visibility = 'public'`, user)
}

func (r *FlightRepo) list(where string, args ...any) ([]domain.Flight, error) {
	rows, err := r.db.Query(`
		SELECT id, from_iata, to_iata, visibility, created_at FROM flights
		`+where+`
		ORDER BY created_at, rowid
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Flight{}
	for rows.Next() {
		var f domain.Flight
		var created string
		if err := rows.Scan(&f.ID, &f.From, &f.To, &f.Visibility, &created); err != nil {
			return nil, err
		}
		if f.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *FlightRepo) Create(user domain.UserID, f domain.Flight) error {
	_, err := r.db.Exec(`
		INSERT INTO flights (id, user_id, from_iata, to_iata, visibility, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, f.ID, user, f.From, f.To, f.Visibility, f.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *FlightRepo) SetVisibility(user domain.UserID, id string, v domain.Visibility) (bool, error) {
	return affected(r.db.Exec(`UPDATE flights SET visibility = ? WHERE user_id = ? AND id = ?`, v, user, id))
}

func (r *FlightRepo) Delete(user domain.UserID, id string) (bool, error) {
	return affected(r.db.Exec(`DELETE FROM flights WHERE user_id = ? AND id = ?`, user, id))
}

func (r *FlightRepo) RouteExists(user domain.UserID, from, to string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM flights WHERE user_id = ? AND from_iata = ? AND to_iata = ?
		)
	`, user, from, to).Scan(&exists)
	return exists, err
}

func affected(res sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
