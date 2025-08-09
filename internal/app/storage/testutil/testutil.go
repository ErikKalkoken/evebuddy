package testutil

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// NewDBInMemory creates and returns a database in memory for tests.
// Important: This variant is not suitable for DB code that runs in goroutines.
func NewDBInMemory() (*sql.DB, *storage.Storage, Factory) {
	// in-memory DB for faster running tests
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

// NewDBOnDisk creates and returns a new temporary database on disk for tests.
// The database is automatically removed once the tests have concluded.
func NewDBOnDisk(t testing.TB) (*sql.DB, *storage.Storage, Factory) {
	// real DB for more thorough tests
	p := filepath.Join(t.TempDir(), "evebuddy_test.sqlite")
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

// DumpTables returns the current content of the given SQL tables as JSON string.
// When no tables are given all non-empty tables will be dumped.
func DumpTables(db *sql.DB, tables ...string) string {
	sql := `SELECT name FROM sqlite_master WHERE type = "table"`
	rows, err := db.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var allTables set.Set[string]
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			panic(err)
		}
		allTables.Add(name)
	}
	if len(tables) == 0 {
		tables = allTables.Slice()
	} else {
		for _, x := range tables {
			if !allTables.Contains(x) {
				panic("Table not found with name: " + x)
			}
		}
	}
	slices.Sort(tables)
	world := make(map[string]any)
	for _, table := range tables {
		sql := fmt.Sprintf("SELECT * FROM %s;", table)
		rows, err := db.Query(sql)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil {
			panic(err)
		}
		data := make([]any, 0)
		for rows.Next() {
			items := make([]any, len(cols))
			for i := range items {
				items[i] = new(any)
			}
			if err := rows.Scan(items...); err != nil {
				panic(err)
			}
			row := make(map[string]any)
			for i, v := range items {
				vv := v.(*any)
				row[cols[i]] = *vv
			}
			data = append(data, row)
		}
		if len(data) == 0 {
			continue
		}
		world[table] = data
	}
	b, err := json.MarshalIndent(world, "", "    ")
	if err != nil {
		panic(err)
	}
	return (string(b))
}

// ErrGroupDebug represents a replacement for errgroup.Group with the same API,
// but it runs the callbacks without Goroutines, which makes debugging much easier.
type ErrGroupDebug struct {
	ff []func() error
}

func (g *ErrGroupDebug) Go(f func() error) {
	if g.ff == nil {
		g.ff = make([]func() error, 0)
	}
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
