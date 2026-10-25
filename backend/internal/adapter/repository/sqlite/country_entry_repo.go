package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"

	"travel-blog/backend/internal/domain"
)

// Every query is limited to one user.
type CountryEntryRepo struct {
	db *sql.DB
}

func NewCountryEntryRepo(db *sql.DB) *CountryEntryRepo {
	return &CountryEntryRepo{db: db}
}

const entryColumns = `iso, status, visibility, food, sightseeing, notes, rating, tags`

type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(row scanner) (domain.CountryEntry, error) {
	var e domain.CountryEntry
	var tags string
	err := row.Scan(&e.ISO, &e.Status, &e.Visibility, &e.Details.Food,
		&e.Details.Sightseeing, &e.Details.Notes, &e.Details.Rating, &tags)
	if err != nil {
		return e, err
	}
	err = json.Unmarshal([]byte(tags), &e.Details.Tags)
	return e, err
}

func (r *CountryEntryRepo) List(user domain.UserID) ([]domain.CountryEntry, error) {
	return r.list(`WHERE user_id = ?`, user)
}

func (r *CountryEntryRepo) ListPublic(user domain.UserID) ([]domain.CountryEntry, error) {
	return r.list(`WHERE user_id = ? AND visibility = 'public'`, user)
}

func (r *CountryEntryRepo) list(where string, args ...any) ([]domain.CountryEntry, error) {
	rows, err := r.db.Query(`SELECT `+entryColumns+` FROM country_entries `+where+` ORDER BY iso`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.CountryEntry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *CountryEntryRepo) Get(user domain.UserID, iso string) (*domain.CountryEntry, error) {
	row := r.db.QueryRow(
		`SELECT `+entryColumns+` FROM country_entries WHERE user_id = ? AND iso = ?`,
		user, iso,
	)
	e, err := scanEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *CountryEntryRepo) Upsert(user domain.UserID, e domain.CountryEntry) error {
	tags := e.Details.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		INSERT INTO country_entries (user_id, `+entryColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, iso) DO UPDATE SET
			status = excluded.status,
			visibility = excluded.visibility,
			food = excluded.food,
			sightseeing = excluded.sightseeing,
			notes = excluded.notes,
			rating = excluded.rating,
			tags = excluded.tags
	`, user, e.ISO, e.Status, e.Visibility, e.Details.Food, e.Details.Sightseeing,
		e.Details.Notes, e.Details.Rating, string(tagsJSON))
	return err
}

func (r *CountryEntryRepo) Delete(user domain.UserID, iso string) error {
	_, err := r.db.Exec(`DELETE FROM country_entries WHERE user_id = ? AND iso = ?`, user, iso)
	return err
}
