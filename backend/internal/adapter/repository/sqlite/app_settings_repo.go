package sqlite

import "database/sql"

type AppSettingsRepo struct {
	db *sql.DB
}

func NewAppSettingsRepo(db *sql.DB) *AppSettingsRepo {
	return &AppSettingsRepo{db: db}
}

func (r *AppSettingsRepo) RegistrationOpen() (bool, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM app_settings WHERE key = 'registration_open'`).Scan(&v)
	return v == "true", err
}

func (r *AppSettingsRepo) SetRegistrationOpen(open bool) error {
	v := "false"
	if open {
		v = "true"
	}
	_, err := r.db.Exec(`UPDATE app_settings SET value = ? WHERE key = 'registration_open'`, v)
	return err
}
