package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/stretchr/testify/assert"
)

func New() (*sql.DB, *storage.Storage, Factory) {
	return NewWithOptions(true)
}

func NewWithOptions(useMemoryDB bool) (*sql.DB, *storage.Storage, Factory) {
	if useMemoryDB {
		// in-memory DB for faster runnng tests
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			panic(err)
		}
		if err := storage.ApplyMigrations(db); err != nil {
			panic(err)
		}
		r := storage.New(db, db)
		factory := NewFactory(r, db)
		return db, r, factory
	}
	// real DB for more thorough tests
	p := filepath.Join(os.TempDir(), "evebuddy_test.sqlite")
	os.Remove(p)
	dbRW, dbRO, err := storage.InitDB("file:" + p)
	if err != nil {
		panic(err)
	}
	r := storage.New(dbRW, dbRO)
	factory := NewFactory(r, dbRO)
	return dbRW, r, factory
}

// func New() (*sql.DB, *storage.Storage, Factory) {

// }

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables(dbRW *sql.DB) {
	_, err := dbRW.Exec("PRAGMA foreign_keys = 0")
	if err != nil {
		panic(err)
	}
	sql := `SELECT name FROM sqlite_master WHERE type = "table"`
	rows, err := dbRW.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			panic(err)
		}
		tables = append(tables, name)
	}
	for _, n := range tables {
		sql := fmt.Sprintf("DELETE FROM %s;", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	for _, n := range tables {
		sql := fmt.Sprintf("DELETE FROM SQLITE_SEQUENCE WHERE name='%s'", n)
		_, err := dbRW.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	_, err = dbRW.Exec("PRAGMA foreign_keys = 1")
	if err != nil {
		panic(err)
	}
}

// AssertEqualSet asserts that two sets are equal.
func AssertEqualSet[T comparable](t *testing.T, want, got set.Set[T]) {
	assert.True(t, want.Equal(got), "got: %s, want: %s", got, want)
}
