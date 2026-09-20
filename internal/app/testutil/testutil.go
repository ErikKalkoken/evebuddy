// Package testutil contains utilities for writing tests.
package testutil

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// defaultDBSetupLogLevel is the log level used while setting up a test DB
// (which includes applying migrations) unless overridden with [WithLogLevel].
// It is set above [slog.LevelInfo] to silence the otherwise very noisy
// migration log output that would otherwise appear on every test run.
const defaultDBSetupLogLevel = slog.LevelWarn

type dbOptions struct {
	logLevel slog.Level
}

// DBOption configures [NewDBInMemory] or [NewDBOnDisk].
type DBOption func(*dbOptions)

// WithLogLevel overrides the log level used while setting up a test DB,
// which is otherwise silenced to [defaultDBSetupLogLevel] to avoid
// drowning test output in migration log lines. Since this ultimately calls
// [slog.SetLogLoggerLevel], it affects the process-wide default logger for
// the duration of the DB setup, not just migration-related log records.
func WithLogLevel(level slog.Level) DBOption {
	return func(o *dbOptions) {
		o.logLevel = level
	}
}

func newDBOptions(opts []DBOption) dbOptions {
	o := dbOptions{logLevel: defaultDBSetupLogLevel}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// withDBSetupLogLevel temporarily sets the default logger's level for the
// duration of f, then restores the previous level.
//
// This mutates process-wide global state, which is safe only because DB
// setup is a short, synchronous call and this test suite does not use
// t.Parallel(). Revisit if that ever changes.
func withDBSetupLogLevel(level slog.Level, f func()) {
	prev := slog.SetLogLoggerLevel(level)
	defer slog.SetLogLoggerLevel(prev)
	f()
}

// NewDBInMemory creates and returns a database in memory for tests.
// Important: This variant is not suitable for DB code that runs in goroutines.
func NewDBInMemory(opts ...DBOption) (*sql.DB, *storage.Storage, Factory) {
	o := newDBOptions(opts)
	// in-memory DB for faster running tests
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	withDBSetupLogLevel(o.logLevel, func() {
		err = storage.ApplyMigrations(db)
	})
	if err != nil {
		panic(err)
	}
	st := storage.New(db, db)
	factory := NewFactory(st, db)
	return db, st, factory
}

// NewDBOnDisk creates and returns a new temporary database on disk for tests.
// The database is automatically removed once the tests have concluded.
func NewDBOnDisk(t testing.TB, opts ...DBOption) (*sql.DB, *storage.Storage, Factory) {
	o := newDBOptions(opts)
	// real DB for more thorough tests
	p := filepath.Join(t.TempDir(), "evebuddy_test.sqlite")
	var dbRW, dbRO *sql.DB
	var err error
	withDBSetupLogLevel(o.logLevel, func() {
		dbRW, dbRO, err = storage.InitDB("file:" + p)
	})
	if err != nil {
		panic(err)
	}
	r := storage.New(dbRW, dbRO)
	factory := NewFactory(r, dbRO)
	return dbRW, r, factory
}

// func New() (*sql.DB, *storage.Storage, Factory) {

// }

// MustTruncateTables is like [TruncateTables] but will panic on any error.
func MustTruncateTables(dbRW *sql.DB) {
	err := TruncateTables(dbRW)
	if err != nil {
		panic(err)
	}
}

// TruncateTables will purge data from all data tables. This is meant for tests.
func TruncateTables(dbRW *sql.DB) error {
	_, err := dbRW.Exec("PRAGMA foreign_keys = 0")
	if err != nil {
		return err
	}
	sql := `SELECT name FROM sqlite_master WHERE type = "table" and name != "migrations"`
	rows, err := dbRW.Query(sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	var tables set.Set[string]
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		tables.Add(name)
	}
	for n := range tables.All() {
		sql := fmt.Sprintf("DELETE FROM %s;", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			return err
		}
	}
	for n := range tables.All() {
		sql := fmt.Sprintf("DELETE FROM SQLITE_SEQUENCE WHERE name='%s'", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			return err
		}
	}
	_, err = dbRW.Exec("PRAGMA foreign_keys = 1")
	if err != nil {
		return err
	}
	return nil
}

// ErrGroupDebug represents a replacement for errgroup.Group with the same API,
// but it runs the callbacks without Goroutines, which makes debugging much easier.
type ErrGroupDebug struct {
	ff []func() error
}

func (g *ErrGroupDebug) Go(f func() error) {
	g.ff = append(g.ff, f)
}

func (g *ErrGroupDebug) Wait() error {
	for _, f := range g.ff {
		err := f()
		if err != nil {
			return err
		}
	}
	return nil
}
