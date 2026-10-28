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

	"github.com/ErikKalkoken/go-set"
)

type MigrateFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

// Run applies all unapplied migrations.
func Run(db *sql.DB, migrations MigrateFS) error {
	if err := createMigrationTracking(db); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return applyNewMigrations(db, migrations)
}

var createMigrationTrackingSQL = `
CREATE TABLE IF NOT EXISTS migrations(
    id INTEGER PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name TEXT NOT NULL,
	UNIQUE (name)
);`

func createMigrationTracking(db *sql.DB) error {
	_, err := db.Exec(createMigrationTrackingSQL)
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
		return fmt.Errorf("list applied migrations: %w", err)
	}
	appliedMigrations := set.Collect(slices.Values(names))
	c, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	var unapplied []migration
	for _, entry := range c {
		fn := entry.Name()
		ext := filepath.Ext(fn)
		if ext != ".sql" {
			continue
		}
		name := strings.TrimSuffix(fn, ext)
		if appliedMigrations.Contains(name) {
			continue
		}
		unapplied = append(unapplied, migration{name: name, filename: fn})
	}
	if len(unapplied) == 0 {
		slog.Info("No new migrations to apply")
		return nil
	}
	slog.Info("Applying new migrations", "count", len(unapplied))
	slices.SortFunc(unapplied, func(a migration, b migration) int {
		return cmp.Compare(a.name, b.name)
	})
	for _, m := range unapplied {
		wrapErr := func(err error) error {
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		p := fmt.Sprintf("migrations/%s", m.filename) // FS uses slashes on all platforms incl. Windows
		data, err := migrations.ReadFile(p)
		if err != nil {
			return wrapErr(err)
		}
		tx, err := db.Begin()
		if err != nil {
			return wrapErr(err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			tx.Rollback()
			return wrapErr(err)
		}
		if _, err := tx.Exec(`INSERT INTO migrations(name) VALUES(?);`, m.name); err != nil {
			tx.Rollback()
			return wrapErr(err)
		}
		if err := tx.Commit(); err != nil {
			return wrapErr(err)
		}
		slog.Info("Successfully applied new migration", "name", m.name)
	}
	return nil
}
