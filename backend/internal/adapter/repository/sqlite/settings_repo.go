package sqlite

import (
	"database/sql"
	"strings"
)

const oidcPrefix = "oidc."

// SettingsRepo works on the configuration database, not the one with the travel data.
type SettingsRepo struct {
	db *sql.DB
}

func NewSettingsRepo(db *sql.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) RegistrationOpen() (bool, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = 'registration_open'`).Scan(&v)
	return v == "true", err
}

func (r *SettingsRepo) SetRegistrationOpen(open bool) error {
	v := "false"
	if open {
		v = "true"
	}
	_, err := r.db.Exec(`UPDATE settings SET value = ? WHERE key = 'registration_open'`, v)
	return err
}

func (r *SettingsRepo) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM settings WHERE key LIKE 'oidc.%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		values[strings.TrimPrefix(k, oidcPrefix)] = v
	}
	return values, rows.Err()
}

func (r *SettingsRepo) Replace(values map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM settings WHERE key LIKE 'oidc.%'`); err != nil {
		return err
	}
	for k, v := range values {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, oidcPrefix+k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SettingsRepo) Clear() error {
	_, err := r.db.Exec(`DELETE FROM settings WHERE key LIKE 'oidc.%'`)
	return err
}
