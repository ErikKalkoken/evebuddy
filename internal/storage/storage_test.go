package storage_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/storage"
)

func setUpDB() (*sql.DB, *storage.Storage, factory.Factory) {
	db, err := storage.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	r := storage.New(db)
	factory := factory.New(r)
	return db, r, factory
}
