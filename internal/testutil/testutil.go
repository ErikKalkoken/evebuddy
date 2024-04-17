package testutil

import (
	"database/sql"

	"example/evebuddy/internal/storage"
)

func New() (*sql.DB, *storage.Storage, Factory) {
	db, err := storage.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	r := storage.New(db)
	factory := NewFactory(r)
	return db, r, factory
}
