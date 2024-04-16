package repository_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/repository"
)

func setUpDB() (*sql.DB, *repository.Repository, factory.Factory) {
	db, err := repository.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	r := repository.New(db)
	factory := factory.New(r)
	return db, r, factory
}
