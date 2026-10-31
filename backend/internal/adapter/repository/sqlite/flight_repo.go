package sqlite

import (
	"database/sql"
	"strings"

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
		SELECT id, from_iata, to_iata, visibility, created_at,
			flight_number, flown_on, duration_minutes, aircraft
		FROM flights
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
		var number, date, aircraft sql.NullString
		var duration sql.NullInt64
		if err := rows.Scan(&f.ID, &f.From, &f.To, &f.Visibility, &created, &number, &date, &duration, &aircraft); err != nil {
			return nil, err
		}
		if f.CreatedAt, err = parseTime(created); err != nil {
			return nil, err
		}
		f.Number, f.Date, f.DurationMinutes, f.Aircraft = number.String, date.String, int(duration.Int64), aircraft.String
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *FlightRepo) Create(user domain.UserID, f domain.Flight) error {
	_, err := r.db.Exec(`
		INSERT INTO flights (id, user_id, from_iata, to_iata, visibility, created_at,
			flight_number, flown_on, duration_minutes, aircraft)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, f.ID, user, f.From, f.To, f.Visibility, formatTime(f.CreatedAt),
		nullString(f.Number), nullString(f.Date), nullInt(f.DurationMinutes), nullString(f.Aircraft))
	return err
}

func (r *FlightRepo) Update(user domain.UserID, id string, u domain.FlightUpdate) (bool, error) {
	var set []string
	var args []any
	add := func(column string, value any) {
		set = append(set, column+" = ?")
		args = append(args, value)
	}
	if u.Visibility != nil {
		add("visibility", *u.Visibility)
	}
	if u.Number != nil {
		add("flight_number", nullString(*u.Number))
	}
	if u.Date != nil {
		add("flown_on", nullString(*u.Date))
	}
	if u.DurationMinutes != nil {
		add("duration_minutes", nullInt(*u.DurationMinutes))
	}
	if u.Aircraft != nil {
		add("aircraft", nullString(*u.Aircraft))
	}
	if len(set) == 0 {
		return r.exists(user, id)
	}
	args = append(args, user, id)
	return affected(r.db.Exec(`UPDATE flights SET `+strings.Join(set, ", ")+` WHERE user_id = ? AND id = ?`, args...))
}

func (r *FlightRepo) exists(user domain.UserID, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM flights WHERE user_id = ? AND id = ?)`, user, id).Scan(&exists)
	return exists, err
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

// Empty values are stored as NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt(n int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(n), Valid: n != 0}
}

func affected(res sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
