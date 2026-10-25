package sqlite

import "database/sql"

type OIDCSettingsRepo struct {
	db *sql.DB
}

func NewOIDCSettingsRepo(db *sql.DB) *OIDCSettingsRepo {
	return &OIDCSettingsRepo{db: db}
}

func (r *OIDCSettingsRepo) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM oidc_settings`)
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
		values[k] = v
	}
	return values, rows.Err()
}

func (r *OIDCSettingsRepo) Replace(values map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM oidc_settings`); err != nil {
		return err
	}
	for k, v := range values {
		if _, err := tx.Exec(`INSERT INTO oidc_settings (key, value) VALUES (?, ?)`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *OIDCSettingsRepo) Clear() error {
	_, err := r.db.Exec(`DELETE FROM oidc_settings`)
	return err
}
