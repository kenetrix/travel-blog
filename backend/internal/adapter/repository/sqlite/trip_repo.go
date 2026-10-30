package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"travel-blog/backend/internal/domain"
)

// Every query is limited to one user.
type TripRepo struct {
	db *sql.DB
}

func NewTripRepo(db *sql.DB) *TripRepo {
	return &TripRepo{db: db}
}

const tripColumns = `id, title, COALESCE(starts_on, ''), COALESCE(ends_on, ''), notes, visibility, created_at, updated_at`

func scanTrip(row scanner) (domain.Trip, error) {
	var t domain.Trip
	var created, updated string
	err := row.Scan(&t.ID, &t.Title, &t.StartsOn, &t.EndsOn, &t.Notes, &t.Visibility, &created, &updated)
	if err != nil {
		return t, err
	}
	if t.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return t, err
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return t, err
}

func (r *TripRepo) List(user domain.UserID) ([]domain.TripSummary, error) {
	rows, err := r.db.Query(`SELECT `+tripColumns+` FROM trips WHERE user_id = ? ORDER BY created_at, rowid`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.TripSummary{}
	index := map[string]int{}
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		index[t.ID] = len(out)
		out = append(out, domain.TripSummary{
			ID: t.ID, Title: t.Title, StartsOn: t.StartsOn, EndsOn: t.EndsOn,
			Visibility: t.Visibility, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
			Countries: []string{},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stops, err := r.db.Query(`SELECT trip_id, country_id FROM locations WHERE user_id = ? ORDER BY trip_id, position`, user)
	if err != nil {
		return nil, err
	}
	defer stops.Close()
	byTrip := map[string][]domain.Location{}
	for stops.Next() {
		var trip, country string
		if err := stops.Scan(&trip, &country); err != nil {
			return nil, err
		}
		byTrip[trip] = append(byTrip[trip], domain.Location{CountryID: country})
	}
	if err := stops.Err(); err != nil {
		return nil, err
	}
	for id, list := range byTrip {
		if i, ok := index[id]; ok {
			out[i].Countries = domain.TripCountries(list)
			out[i].StopCount = len(list)
		}
	}
	return out, nil
}

func (r *TripRepo) Get(user domain.UserID, id string) (*domain.Trip, error) {
	t, err := scanTrip(r.db.QueryRow(`SELECT `+tripColumns+` FROM trips WHERE user_id = ? AND id = ?`, user, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`SELECT `+locationColumns+` FROM locations WHERE user_id = ? AND trip_id = ? ORDER BY position`, user, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	t.Stops = []domain.Location{}
	for rows.Next() {
		l, err := scanLocation(rows)
		if err != nil {
			return nil, err
		}
		t.Stops = append(t.Stops, l)
	}
	return &t, rows.Err()
}

func (r *TripRepo) Create(user domain.UserID, t domain.Trip) error {
	return inTx(r.db, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`
			INSERT INTO trips (id, user_id, title, starts_on, ends_on, notes, visibility, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, t.ID, user, t.Title, nullIfEmpty(t.StartsOn), nullIfEmpty(t.EndsOn), t.Notes, t.Visibility,
			formatTime(t.CreatedAt), formatTime(t.UpdatedAt)); err != nil {
			return err
		}
		for i, s := range t.Stops {
			if err := insertStop(tx, user, t, i, s); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *TripRepo) Replace(user domain.UserID, t domain.Trip) (bool, []string, error) {
	found := false
	var removed []string
	err := inTx(r.db, func(tx *sql.Tx) error {
		var err error
		found, err = affected(tx.Exec(`
			UPDATE trips SET title = ?, starts_on = ?, ends_on = ?, notes = ?, visibility = ?, updated_at = ?
			WHERE user_id = ? AND id = ?
		`, t.Title, nullIfEmpty(t.StartsOn), nullIfEmpty(t.EndsOn), t.Notes, t.Visibility, formatTime(t.UpdatedAt), user, t.ID))
		if err != nil || !found {
			return err
		}

		// Stops of the trip that are not in the new list
		gone := `l.user_id = ? AND l.trip_id = ?`
		args := []any{user, t.ID}
		if len(t.Stops) > 0 {
			gone += ` AND l.id NOT IN (?` + strings.Repeat(`, ?`, len(t.Stops)-1) + `)`
			for _, s := range t.Stops {
				args = append(args, s.ID)
			}
		}
		if removed, err = photoKeys(tx, `SELECT p.storage_key FROM photos p JOIN locations l ON l.id = p.location_id WHERE `+gone, args...); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM locations AS l WHERE `+gone, args...); err != nil {
			return err
		}
		// Free every position first, or moving a stop would collide with UNIQUE (trip_id, position)
		if _, err := tx.Exec(`UPDATE locations SET position = -position - 1 WHERE user_id = ? AND trip_id = ?`, user, t.ID); err != nil {
			return err
		}
		for i, s := range t.Stops {
			updated, err := affected(tx.Exec(`
				UPDATE locations SET position = ?, country_id = ?, name = ?, lat = ?, lng = ?, visited_at = ?,
					notes = ?, rating = ?, category = ?, links = ?, visibility = ?
				WHERE user_id = ? AND trip_id = ? AND id = ?
			`, i, s.CountryID, s.Name, s.Lat, s.Lng, nullIfEmpty(s.VisitedAt), s.Notes, s.Rating, s.Category, linksJSON(s.Links), t.Visibility, user, t.ID, s.ID))
			if err != nil {
				return err
			}
			if !updated {
				if err := insertStop(tx, user, t, i, s); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil || !found {
		return found, nil, err
	}
	return true, removed, nil
}

func (r *TripRepo) Delete(user domain.UserID, id string) (bool, []string, error) {
	found := false
	var removed []string
	err := inTx(r.db, func(tx *sql.Tx) error {
		var err error
		if removed, err = photoKeys(tx, `
			SELECT p.storage_key FROM photos p JOIN locations l ON l.id = p.location_id
			WHERE l.user_id = ? AND l.trip_id = ?
		`, user, id); err != nil {
			return err
		}
		found, err = affected(tx.Exec(`DELETE FROM trips WHERE user_id = ? AND id = ?`, user, id))
		return err
	})
	if err != nil || !found {
		return found, nil, err
	}
	return true, removed, nil
}

// Stops take the trip's visibility, so a private trip never has public stops.
func insertStop(tx *sql.Tx, user domain.UserID, t domain.Trip, position int, s domain.Location) error {
	_, err := tx.Exec(`
		INSERT INTO locations (id, user_id, trip_id, position, country_id, name, lat, lng, visited_at, notes, rating, category, links, visibility)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, user, t.ID, position, s.CountryID, s.Name, s.Lat, s.Lng, nullIfEmpty(s.VisitedAt), s.Notes, s.Rating, s.Category, linksJSON(s.Links), t.Visibility)
	return err
}

func linksJSON(links []domain.StopLink) string {
	if links == nil {
		return "[]"
	}
	b, _ := json.Marshal(links)
	return string(b)
}

func photoKeys(tx *sql.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func inTx(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
