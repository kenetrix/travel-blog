package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func NewDB(path string, migrations fs.FS) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// Enable foreign keys on every connection (e.g. a deleted user takes
	// their data along)
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	// SQLite allows only one writer at a time
	db.SetMaxOpenConns(1)
	if err := migrate(db, migrations, path); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// migrate tracks the number of applied migrations in user_version.
func migrate(db *sql.DB, migrations fs.FS, path string) error {
	files, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var version int
	if err := conn.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version > 0 && version < len(files) {
		if err := backup(ctx, conn, path, version); err != nil {
			return fmt.Errorf("backup before migrating: %w", err)
		}
	}
	for i := version; i < len(files); i++ {
		script, err := fs.ReadFile(migrations, files[i])
		if err != nil {
			return err
		}
		if err := migrateOne(ctx, conn, string(script), i+1); err != nil {
			return fmt.Errorf("migration %s: %w", files[i], err)
		}
	}
	return nil
}

// backup saves a consistent copy of the database as <path>.bak-v<version> before pending
// migrations run, so an update can be rolled back. An existing copy of that version is kept.
func backup(ctx context.Context, conn *sql.Conn, path string, version int) error {
	dest := fmt.Sprintf("%s.bak-v%d", path, version)
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	_, err := conn.ExecContext(ctx, fmt.Sprintf(`VACUUM INTO '%s'`, strings.ReplaceAll(dest, "'", "''")))
	return err
}

// migrateOne applies one migration in a transaction with foreign keys off so tables can be
// recreated (SQLite's recommended way); before the commit it checks that all still hold.
func migrateOne(ctx context.Context, conn *sql.Conn, script string, version int) error {
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(script); err != nil {
		return err
	}
	rows, err := tx.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	violated := rows.Next()
	rows.Close()
	if violated {
		return errors.New("foreign key violation")
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version)); err != nil {
		return err
	}
	return tx.Commit()
}
