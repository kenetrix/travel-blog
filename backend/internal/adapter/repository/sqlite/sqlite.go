package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	// The databases hold sessions and secrets; only the owner may read them
	if err := os.Chmod(path, 0o600); err != nil {
		log.Printf("database permissions: %v", err)
	}
	return db, nil
}

// migrate tracks the number of applied migrations in user_version.
func migrate(db *sql.DB, migrations fs.FS, path string) error {
	scripts, err := loadMigrations(migrations)
	if err != nil {
		return err
	}

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
	if version > 0 && version < len(scripts) {
		if err := backup(ctx, conn, path, version); err != nil {
			return fmt.Errorf("backup before migrating: %w", err)
		}
	}
	for i := version; i < len(scripts); i++ {
		if err := migrateOne(ctx, conn, scripts[i].sql, i+1); err != nil {
			return fmt.Errorf("migration %s: %w", scripts[i].name, err)
		}
	}
	return nil
}

type migration struct{ name, sql string }

// loadMigrations returns one migration per top-level .sql file or folder, in name order.
// A folder's .sql files are joined in name order.
func loadMigrations(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		switch {
		case e.IsDir():
			files, err := fs.Glob(fsys, e.Name()+"/*.sql")
			if err != nil {
				return nil, err
			}
			sort.Strings(files)
			var parts []string
			for _, f := range files {
				b, err := fs.ReadFile(fsys, f)
				if err != nil {
					return nil, err
				}
				parts = append(parts, string(b))
			}
			if len(parts) > 0 {
				out = append(out, migration{e.Name(), strings.Join(parts, "\n")})
			}
		case strings.HasSuffix(e.Name(), ".sql"):
			b, err := fs.ReadFile(fsys, e.Name())
			if err != nil {
				return nil, err
			}
			out = append(out, migration{e.Name(), string(b)})
		}
	}
	return out, nil
}

// backup saves a consistent copy of the database as <path>.bak-v<version> before pending
// migrations run, so an update can be rolled back. An existing copy of that version is kept.
func backup(ctx context.Context, conn *sql.Conn, path string, version int) error {
	dest := fmt.Sprintf("%s.bak-v%d", path, version)
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`VACUUM INTO '%s'`, strings.ReplaceAll(dest, "'", "''"))); err != nil {
		return err
	}
	return os.Chmod(dest, 0o600)
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

// Timestamps are stored as RFC 3339 text in UTC, which sorts chronologically.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
