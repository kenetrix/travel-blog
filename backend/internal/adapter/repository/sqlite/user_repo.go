package sqlite

import (
	"database/sql"
	"errors"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"travel-blog/backend/internal/application/port"
	"travel-blog/backend/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const userColumns = `id, username, COALESCE(email, ''), first_name, last_name,
	display_name, bio, language, timezone, home_country, home_airport, created_at,
	role, must_change_password`

func scanUser(row scanner) (*domain.User, error) {
	var u domain.User
	var created string
	p := &u.Profile
	err := row.Scan(&u.ID, &p.Username, &p.Email, &p.FirstName, &p.LastName,
		&p.DisplayName, &p.Bio, &p.Language, &p.Timezone, &p.HomeCountry,
		&p.HomeAirport, &created, &u.Role, &u.MustChangePassword)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return nil, err
	}
	return &u, nil
}

// nullIfEmpty stores an empty email as NULL (UNIQUE allows several NULLs).
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// conflict translates violated UNIQUE constraints into port.ErrConflict.
func conflict(err error) error {
	var se *sqlite.Error
	if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return port.ErrConflict
	}
	return err
}

func (r *UserRepo) Create(u domain.User, passwordHash string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertUser(tx, u, passwordHash); err != nil {
		return err
	}
	return tx.Commit()
}

func insertUser(tx *sql.Tx, u domain.User, passwordHash string) error {
	p := u.Profile
	_, err := tx.Exec(`
		INSERT INTO users (id, username, email, password_hash, first_name, last_name,
			display_name, bio, language, timezone, home_country, home_airport, created_at,
			role, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, p.Username, nullIfEmpty(p.Email), nullIfEmpty(passwordHash), p.FirstName,
		p.LastName, p.DisplayName, p.Bio, p.Language, p.Timezone, p.HomeCountry,
		p.HomeAirport, p.CreatedAt.UTC().Format(time.RFC3339Nano),
		cmpOr(u.Role, domain.RoleUser), u.MustChangePassword)
	if err != nil {
		return conflict(err)
	}
	_, err = tx.Exec(`INSERT INTO user_settings (user_id) VALUES (?)`, u.ID)
	return err
}

func (r *UserRepo) Get(id domain.UserID) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = ?`, id))
}

func (r *UserRepo) GetByUsername(username string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ?`, username))
}

func (r *UserRepo) GetByEmail(email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE email = ?`, email))
}

func (r *UserRepo) GetByIdentity(issuer, subject string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id =
		(SELECT user_id FROM oidc_identities WHERE issuer = ? AND subject = ?)`, issuer, subject))
}

func (r *UserRepo) LinkIdentity(issuer, subject string, id domain.UserID) error {
	_, err := r.db.Exec(`INSERT INTO oidc_identities (issuer, subject, user_id, created_at)
		VALUES (?, ?, ?, ?)`, issuer, subject, id, time.Now().UTC().Format(time.RFC3339Nano))
	return conflict(err)
}

func (r *UserRepo) CreateWithIdentity(u domain.User, issuer, subject string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertUser(tx, u, ""); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO oidc_identities (issuer, subject, user_id, created_at)
		VALUES (?, ?, ?, ?)`, issuer, subject, u.ID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return conflict(err)
	}
	return tx.Commit()
}

func (r *UserRepo) Credentials(username string) (domain.UserID, string, bool, error) {
	var id domain.UserID
	var hash string
	err := r.db.QueryRow(
		`SELECT id, password_hash FROM users WHERE username = ? AND password_hash IS NOT NULL`,
		username,
	).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	return id, hash, err == nil, err
}

func (r *UserRepo) PasswordHash(id domain.UserID) (string, error) {
	var hash sql.NullString
	err := r.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, id).Scan(&hash)
	return hash.String, err
}

func (r *UserRepo) SetPasswordHash(id domain.UserID, hash string) error {
	_, err := r.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

func (r *UserRepo) UpdateProfile(id domain.UserID, p domain.Profile) error {
	_, err := r.db.Exec(`
		UPDATE users SET username = ?, email = ?, first_name = ?, last_name = ?,
			display_name = ?, bio = ?, language = ?, timezone = ?, home_country = ?,
			home_airport = ?
		WHERE id = ?
	`, p.Username, nullIfEmpty(p.Email), p.FirstName, p.LastName, p.DisplayName, p.Bio,
		p.Language, p.Timezone, p.HomeCountry, p.HomeAirport, id)
	return conflict(err)
}

func (r *UserRepo) Settings(id domain.UserID) (domain.Settings, error) {
	var s domain.Settings
	err := r.db.QueryRow(`
		SELECT distance_unit, theme, default_view, default_visibility, profile_visibility
		FROM user_settings WHERE user_id = ?
	`, id).Scan(&s.DistanceUnit, &s.Theme, &s.DefaultView, &s.DefaultVisibility, &s.ProfileVisibility)
	return s, err
}

func (r *UserRepo) SaveSettings(id domain.UserID, s domain.Settings) error {
	_, err := r.db.Exec(`
		UPDATE user_settings SET distance_unit = ?, theme = ?, default_view = ?,
			default_visibility = ?, profile_visibility = ?
		WHERE user_id = ?
	`, s.DistanceUnit, s.Theme, s.DefaultView, s.DefaultVisibility, s.ProfileVisibility, id)
	return err
}

func (r *UserRepo) Delete(id domain.UserID) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func cmpOr[T comparable](v, fallback T) T {
	var zero T
	if v == zero {
		return fallback
	}
	return v
}

func (r *UserRepo) List() ([]domain.User, error) {
	rows, err := r.db.Query(`SELECT ` + userColumns + ` FROM users
		WHERE id != ? ORDER BY created_at, username`, domain.LocalUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *UserRepo) CountAdmins() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin' AND id != ?`, domain.LocalUserID).Scan(&n)
	return n, err
}

func (r *UserRepo) SetRole(id domain.UserID, role domain.Role) error {
	_, err := r.db.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, id)
	return err
}

func (r *UserRepo) SetMustChangePassword(id domain.UserID, must bool) error {
	_, err := r.db.Exec(`UPDATE users SET must_change_password = ? WHERE id = ?`, must, id)
	return err
}
