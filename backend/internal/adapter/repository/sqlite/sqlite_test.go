package sqlite_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"travel-blog/backend/internal/adapter/repository/sqlite"
)

func migrations(files ...string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for i, sql := range files {
		fsys[fmt.Sprintf("%03d.sql", i+1)] = &fstest.MapFile{Data: []byte(sql)}
	}
	return fsys
}

func TestBackupBeforePendingMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "app.db")

	// A fresh database has nothing to save
	db, err := sqlite.NewDB(path, migrations(`CREATE TABLE a (x TEXT); INSERT INTO a VALUES ('one');`))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err := os.Stat(path + ".bak-v0"); err == nil {
		t.Fatal("backup of an empty database")
	}

	// Nothing pending: no backup
	db, err = sqlite.NewDB(path, migrations(`CREATE TABLE a (x TEXT);`))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err := os.Stat(path + ".bak-v1"); err == nil {
		t.Fatal("backup without pending migrations")
	}

	// A pending migration saves the state before it
	db, err = sqlite.NewDB(path, migrations(`CREATE TABLE a (x TEXT);`, `DROP TABLE a;`))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	old, err := sqlite.NewDB(path+".bak-v1", migrations(`CREATE TABLE a (x TEXT);`))
	if err != nil {
		t.Fatalf("backup is not a database: %v", err)
	}
	defer old.Close()
	var x string
	if err := old.QueryRow(`SELECT x FROM a`).Scan(&x); err != nil || x != "one" {
		t.Errorf("backup content: %q %v", x, err)
	}
}

func TestFolderIsOneMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	fsys := fstest.MapFS{
		"001_init/02_b.sql": {Data: []byte(`INSERT INTO a VALUES ('b');`)},
		"001_init/01_a.sql": {Data: []byte(`CREATE TABLE a (x TEXT);`)},
		"README.md":         {Data: []byte(`ignored`)},
	}
	db, err := sqlite.NewDB(path, fsys)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("folder counts as %d migrations: %v", version, err)
	}
	db.Close()

	// A later file is migration 2 and a backup of version 1 is taken first
	fsys["002_more.sql"] = &fstest.MapFile{Data: []byte(`INSERT INTO a VALUES ('c');`)}
	db, err = sqlite.NewDB(path, fsys)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM a`).Scan(&n); err != nil || n != 2 {
		t.Errorf("rows: %d %v", n, err)
	}
	if _, err := os.Stat(path + ".bak-v1"); err != nil {
		t.Errorf("no backup of version 1: %v", err)
	}
}
