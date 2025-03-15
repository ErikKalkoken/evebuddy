package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/migrate"
)

var ErrNotFound = errors.New("object not found")

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Storage struct {
	MaxListEveEntitiesForIDs int // Max IDs per SQL query

	db *sql.DB
	q  *queries.Queries
}

// New returns a new storage object.
func New(db *sql.DB) *Storage {
	r := &Storage{
		db:                       db,
		MaxListEveEntitiesForIDs: 1000,
		q:                        queries.New(db),
	}
	return r
}

// InitDB initializes the database and returns it.
func InitDB(dsn string) (*sql.DB, error) {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_busy_timeout", "5000") // 5000 = 5 seconds
	v.Add("_cache_size", "-20000") // -20000 = 20 MB
	v.Add("_synchronous", "normal")
	v.Add("_txlock", "IMMEDIATE")
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
