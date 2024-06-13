package storage

import (
	"bytes"
	"database/sql"
	"embed"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

var schema = `
CREATE TABLE migrations(
    id INTEGER PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name TEXT NOT NULL,
	UNIQUE (name)
);
`

func createMigrationTracking(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

func recordMigration(db *sql.DB, name string) error {
	sql := `INSERT INTO migrations(name) VALUES(?);`
	_, err := db.Exec(sql, name)
	return err
}

func listMigrations(db *sql.DB) ([]string, error) {
	sql := `SELECT name FROM migrations;`
	rows, err := db.Query(sql)
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

func runMigrations(db *sql.DB) error {
	c, err := embedMigrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	filenames := make([]string, 0)
	for _, entry := range c {
		filenames = append(filenames, entry.Name())
	}
	slices.Sort(filenames)
	var count int
	for _, fn := range filenames {
		p := filepath.Join("migrations", fn)
		f, err := embedMigrations.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		buf := bytes.NewBuffer(nil)
		_, err = io.Copy(buf, f)
		if err != nil {
			return err
		}
		_, err = db.Exec(buf.String())
		if err != nil {
			return err
		}
		count++
	}
	slog.Info("Migrations applied", "count", count)
	return nil
}
