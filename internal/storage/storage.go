package storage

import (
	"database/sql"
	"errors"
	"example/evebuddy/internal/sqlc"
	"fmt"
	"log/slog"
	"net/url"
)

var ErrNotFound = errors.New("object not found")

type Storage struct {
	q  *sqlc.Queries
	db *sql.DB
}

func New(db *sql.DB) *Storage {
	r := &Storage{q: sqlc.New(db), db: db}
	return r
}

// ConnectDB initializes the database and returns it.
func ConnectDB(dataSourceName string, create bool) (*sql.DB, error) {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_synchronous", "normal")
	dsn := fmt.Sprintf("%s?%s", dataSourceName, v.Encode())
	slog.Debug("Connecting to sqlite", "dsn", dsn)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to database")
	if create {
		_, err = db.Exec(sqlc.Schema())
		if err != nil {
			return nil, err
		}

	}
	return db, nil
}
