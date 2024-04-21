package storage

import (
	"database/sql"
	"errors"
	"example/evebuddy/internal/storage/queries"
	"fmt"
	"log/slog"
	"net/url"
)

var ErrNotFound = errors.New("object not found")

type Storage struct {
	q  *queries.Queries
	db *sql.DB
}

// New returns a new storage object.
func New(db *sql.DB) *Storage {
	r := &Storage{q: queries.New(db), db: db}
	return r
}

// InitDB initializes the database and returns it.
func InitDB(dsn string) (*sql.DB, error) {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_synchronous", "normal")
	dsn2 := fmt.Sprintf("%s?%s", dsn, v.Encode())
	slog.Debug("Connecting to sqlite", "dsn", dsn2)
	db, err := sql.Open("sqlite3", dsn2)
	if err != nil {
		return nil, err
	}
	slog.Info(fmt.Sprintf("Connected to database: %s", dsn))
	hasSchema, err := schemaExists(db)
	if err != nil {
		return nil, err
	}
	if !hasSchema {
		_, err = db.Exec(queries.Schema())
		if err != nil {
			return nil, err
		}
		slog.Info("Database created")
	}
	return db, nil
}

func schemaExists(db *sql.DB) (bool, error) {
	rows, err := db.Query("SELECT NAME from sqlite_master;")
	if err != nil {
		return false, err
	}
	for rows.Next() {
		return true, nil
	}
	return false, nil
}
