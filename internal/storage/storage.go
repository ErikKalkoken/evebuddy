package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/migrate"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

var ErrNotFound = errors.New("object not found")

//go:embed migrations/*.sql
var embedMigrations embed.FS

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
	v.Add("_txlock", "immediate")
	dsn2 := fmt.Sprintf("%s?%s", dsn, v.Encode())
	slog.Debug("Connecting to sqlite", "dsn", dsn2)
	db, err := sql.Open("sqlite3", dsn2)
	if err != nil {
		return nil, err
	}
	slog.Info("Connected to database", "DSN", dsn)
	if err := migrate.Run(db, embedMigrations); err != nil {
		return nil, err
	}
	return db, nil
}
