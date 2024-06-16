// Package migrate provides a simple mechanism for dealing with migrations of a SQLite database.
package migrate

import (
	"cmp"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type MigrateFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// Run applies all unapplied migrations.
func Run(db *sql.DB, migrations MigrateFS) error {
	isEmpty, err := isEmpty(db)
	if err != nil {
		return err
	}
	if isEmpty {
		if err := createMigrationTracking(db); err != nil {
			return err
		}
	}
	if err := applyNewMigrations(db, migrations); err != nil {
		return err
	}
	return nil
}

var createMigrationTrackingSQL = `
CREATE TABLE migrations(
    id INTEGER PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name TEXT NOT NULL,
	UNIQUE (name)
);`

func createMigrationTracking(db *sql.DB) error {
	_, err := db.Exec(createMigrationTrackingSQL)
	return err
}

func recordMigration(db *sql.DB, name string) error {
	_, err := db.Exec(`INSERT INTO migrations(name) VALUES(?);`, name)
	return err
}

func listMigrationNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM migrations;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

type migration struct {
	name     string
	filename string
}

// applyNewMigrations applies any new migrations in alphabetical order.
//
// migrations is a filesystem containing the SQL files in a folder called "migrations".
func applyNewMigrations(db *sql.DB, migrations MigrateFS) error {
	names, err := listMigrationNames(db)
	if err != nil {
		return err
	}
	appliedMigrations := set.NewFromSlice(names)
	c, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	unapplied := make([]migration, 0)
	for _, entry := range c {
		fn := entry.Name()
		ext := filepath.Ext(fn)
		if ext != ".sql" {
			continue
		}
		name := strings.TrimSuffix(fn, ext)
		if appliedMigrations.Has(name) {
			continue
		}
		unapplied = append(unapplied, migration{name: name, filename: fn})
	}
	if len(unapplied) == 0 {
		slog.Info("No new migrations to apply")
		return nil
	}
	slog.Info("Applying new migrations", "count", len(unapplied))
	fmt.Print("Updating database. This may take a moment...")
	slices.SortFunc(unapplied, func(a migration, b migration) int {
		return cmp.Compare(a.name, b.name)
	})
	var count int
	for _, m := range unapplied {
		p := fmt.Sprintf("migrations/%s", m.filename) // FS uses slashes on all platforms incl. Windows
		data, err := migrations.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = db.Exec(string(data))
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return err
		}
		count++
		recordMigration(db, m.name)
		slog.Info("Successfully applied new migration", "name", m.name)
	}
	fmt.Println("DONE")
	return nil
}

// isEmpty reports wether the database is empty.
func isEmpty(db *sql.DB) (bool, error) {
	rows, err := db.Query("SELECT NAME from sqlite_master;")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		return false, nil
	}
	return true, nil
}
