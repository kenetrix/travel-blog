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
