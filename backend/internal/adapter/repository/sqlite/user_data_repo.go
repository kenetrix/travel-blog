package sqlite

import (
	"database/sql"

	"travel-blog/backend/internal/domain"
)

type UserDataRepo struct{ db *sql.DB }

func NewUserDataRepo(db *sql.DB) *UserDataRepo { return &UserDataRepo{db: db} }

// DeleteAll removes a user's rows in one transaction; stops and photo records
// follow their trips and stops via foreign keys.
func (r *UserDataRepo) DeleteAll(user domain.UserID) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"country_entries", "flights", "trips"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE user_id = ?`, user); err != nil {
			return err
		}
	}
	return tx.Commit()
}
