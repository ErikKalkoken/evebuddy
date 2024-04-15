package service_test

import (
	"database/sql"
	"example/evebuddy/internal/factory"
	"example/evebuddy/internal/repository"
)

func setUpDB() (*sql.DB, *repository.Queries, factory.Factory) {
	db, err := repository.ConnectDB(":memory:", true)
	if err != nil {
		panic(err)
	}
	q := repository.New(db)
	factory := factory.New(q)
	return db, q, factory
}
