package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/migrate"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Storage struct {
	MaxListEveEntitiesForIDs int // Max IDs per SQL query

	dbRO *sql.DB
	dbRW *sql.DB
	qRO  *queries.Queries
	qRW  *queries.Queries
}

// New returns a new storage object.
func New(dbRW *sql.DB, dbRO *sql.DB) *Storage {
	r := &Storage{
		dbRO:                     dbRO,
		dbRW:                     dbRW,
		MaxListEveEntitiesForIDs: 1000,
		qRO:                      queries.New(dbRO),
		qRW:                      queries.New(dbRW),
	}
	return r
}

// InitDB initializes the database and returns it.
func InitDB(dsn string) (dbRW *sql.DB, dbRO *sql.DB, err error) {
	// create RW connection
	dsn2 := sqliteDSN(dsn, false)
	slog.Info("Creating RW connection to DB", "dsn", dsn2)
	dbRW, err = sql.Open("sqlite3", dsn2)
	if err != nil {
		err = fmt.Errorf("open RW connection: %s: %w", dsn, err)
		return
	}
	dbRW.SetMaxOpenConns(1)
	if err = ApplyMigrations(dbRW); err != nil {
		err = fmt.Errorf("apply migrations: %w", err)
		return
	}
	// create RO connection
	dsn2 = sqliteDSN(dsn, true)
	slog.Info("Creating RO connection to DB", "DSN", dsn)
	dbRO, err = sql.Open("sqlite3", dsn2)
	if err != nil {
		err = fmt.Errorf("open RO connection: %s: %w", dsn, err)
		return
	}
	return
}

func ApplyMigrations(db *sql.DB) error {
	return migrate.Run(db, embedMigrations)
}

func sqliteDSN(dsn string, isReadonly bool) string {
	v := url.Values{}
	v.Add("_fk", "on")
	v.Add("_journal_mode", "WAL")
	v.Add("_busy_timeout", "5000") // 5000 = 5 seconds
	v.Add("_cache_size", "-20000") // -20000 = 20 MB
	v.Add("_synchronous", "normal")
	if isReadonly {
		v.Add("mode", "ro")
	} else {
		v.Add("_txlock", "IMMEDIATE")
		v.Add("mode", "rwc")
	}
	dsn2 := fmt.Sprintf("%s?%s", dsn, v.Encode())
	return dsn2
}

func convertGetError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		err = app.ErrNotFound
	}
	return err
}
