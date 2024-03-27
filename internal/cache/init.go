// Package cache provides a simple persistent cache.
package cache

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var db *sqlx.DB

var schema = `
	CREATE TABLE IF NOT EXISTS cache_keys (
		key text PRIMARY KEY NOT NULL,
		value blob NOT NULL,
		expires_at datetime NOT NULL
	);
	CREATE INDEX IF NOT EXISTS cache_keys_expires_at_idx ON cache_keys (expires_at);
`
var pragmas = `
	PRAGMA journal_mode = WAL;
	PRAGMA synchronous = normal;
	PRAGMA temp_store = memory;
	PRAGMA mmap_size = 30000000000;
`

// TODO: Add pragmas as DSN param
// InitDB initializes the database (needs to be called once).
func InitDB(dataSourceName string) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s?_fk=on", dataSourceName)
	myDb, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to database")
	_, err = myDb.Exec(schema)
	if err != nil {
		return nil, err
	}
	_, err = myDb.Exec(pragmas)
	if err != nil {
		return nil, err
	}
	db = myDb
	return myDb, nil
}
