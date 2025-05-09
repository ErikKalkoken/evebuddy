package testutil

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// New creates and returns a database in memory for tests.
// Important: This variant is not suitable for DB code that runs in goroutines.
func New() (*sql.DB, *storage.Storage, Factory) {
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

// NeNewDBWithFile create and returns a new database on disk for tests.
// The caller of this function is reponsible for deleting the file when the tests have concluded.
func NewDBOnDisk(path string) (*sql.DB, *storage.Storage, Factory) {
	// real DB for more thorough tests
	p := filepath.Join(path, "evebuddy_test.sqlite")
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
