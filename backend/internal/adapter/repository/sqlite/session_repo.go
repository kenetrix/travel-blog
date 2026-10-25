package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"travel-blog/backend/internal/domain"
)

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Create(s domain.Session) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		s.TokenHash, s.UserID, s.CreatedAt.UTC().Format(time.RFC3339Nano),
		s.ExpiresAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (r *SessionRepo) Get(tokenHash string) (*domain.Session, error) {
	s := domain.Session{TokenHash: tokenHash}
	var created, expires string
	err := r.db.QueryRow(
		`SELECT user_id, created_at, expires_at FROM sessions WHERE token_hash = ?`, tokenHash,
	).Scan(&s.UserID, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if s.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return nil, err
	}
	if s.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Delete(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (r *SessionRepo) DeleteOthers(user domain.UserID, keep string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`, user, keep)
	return err
}

// DeleteExpired compares UTC RFC 3339 timestamps as text, which is
// equivalent to a time comparison.
func (r *SessionRepo) DeleteExpired(now time.Time) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, now.UTC().Format(time.RFC3339Nano))
	return err
}
