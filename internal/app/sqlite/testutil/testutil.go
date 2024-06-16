package testutil

import (
	"database/sql"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
)

func New() (*sql.DB, *sqlite.Storage, Factory) {
	db, err := sqlite.InitDB(":memory:")
	if err != nil {
		panic(err)
	}
	r := sqlite.New(db)
	factory := NewFactory(r, db)
	return db, r, factory
}

// TruncateTables will purge data from all tables. This is meant for tests.
func TruncateTables(db *sql.DB) {
	_, err := db.Exec("PRAGMA foreign_keys = 0")
	if err != nil {
		panic(err)
	}
	sql := `SELECT name FROM sqlite_master WHERE type = "table"`
	rows, err := db.Query(sql)
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
		_, err := db.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	for _, n := range tables {
		sql := fmt.Sprintf("DELETE FROM SQLITE_SEQUENCE WHERE name='%s'", n)
		_, err := db.Exec(sql)
		if err != nil {
			panic(err)
		}
	}
	_, err = db.Exec("PRAGMA foreign_keys = 1")
	if err != nil {
		panic(err)
	}
}
